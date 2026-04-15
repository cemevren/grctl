---
title: Overview
---

# Ground Control

Ground Control is a lightweight workflow orchestration engine built for fail-safe execution. Coordinate microservices, background tasks, AI agents, and edge services with built-in durability, retries, and fault tolerance. built on [NATS.io](https://nats.io).

!!! warning "Status: Pre-alpha"

    The API is currently unstable and subject to change.

## Why

You have a process that spans multiple services. A sequence of API calls, database writes, and external requests that need to complete together. Today you're wiring this with queues, retry logic, state tracking, and idempotency checks spread across services. It works, until a worker crashes midway. Now you're debugging partial state, writing compensation logic, and building monitoring to catch the cases your retry logic doesn't cover.

Ground Control replaces that infrastructure. You write the process as a straight-line function. The engine records each step's result as it happens. If a worker crashes, another picks up and completed work is not re-executed. State is always consistent, either a step's results are fully saved or not at all.

## What it looks like
Ground Control uses a clean, decorator-based API similar to FastAPI, organizing logic into Workflows, Tasks, and Steps.

```python {filename="process_order.py"}
order = Workflow(workflow_type="ProcessOrder")


# A Task is a unit of work within a step, marked with @task.
# Tasks are recorded in history. If a step re-executes after a crash,
# completed tasks return their recorded result instead of running again.
@task
async def charge_payment(order_id: str, amount: float) -> str:
    return await payment_gateway.charge(order_id, amount)

@task
async def reserve_inventory(item_id: str, qty: int) -> None:
    await inventory_service.reserve(item_id, qty)


# A Step is a transaction boundary within a workflow.
# Only one step runs at a time per workflow — no concurrent steps.
# When a step completes, the engine saves its outcome before moving to the next.
# Each step handler decides what happens next: transition to another step,
# wait for an event, complete, or fail.
@order.start()
async def start(ctx: Context, order_id: str, item_id: str, qty: int, amount: float) -> Directive:
    ctx.store.put("order_id", order_id)

    # Multiple tasks can run in parallel within a step
    payment_id, _ = await asyncio.gather(
        charge_payment(order_id, amount),
        reserve_inventory(item_id, qty),
    )
    ctx.store.put("payment_id", payment_id)

    # Move to another step
    return ctx.next.step(ship)


@order.step()
async def ship(ctx: Context) -> Directive:
    order_id = await ctx.store.get("order_id")

    # Deterministic functions (ctx.now, ctx.random, ctx.uuid4, ctx.sleep)
    # wrap non-deterministic operations. Their results are recorded in history
    # so that during replay, the same values are returned.
    shipped_at = await ctx.now()
    tracking_id = await ctx.uuid4()

    await create_shipment(order_id, str(tracking_id))
    ctx.store.put("shipped_at", shipped_at.isoformat())

    # Complete workflow execution with result. The result will be returned to the client
    return ctx.next.complete({"tracking_id": str(tracking_id)})
```

## The Ground Control Difference

-   **Step-Based Orchestration**: Ground Control organizes logic around `Steps`. Unlike engines that queue individual tasks, Ground Control distributes step execution. This increases performance and provides a cleaner way to organize complex workflow code.
-   **Stateless Workers**: Most engines run internal state logic on workers. Ground Control does the opposite: workers only execute your code, while the engine state machine runs on the Ground Control server. This makes workers easier to scale, update, and manage.
-   **NATS-Native Architecture**: Instead of relying on traditional relational databases, Ground Control uses NATS JetStream. This provides high write throughput for event-heavy workflows and allows Ground Control to run as a **single binary** with no external dependencies (no separate DB, no load balancers).
-   **Distributed by Design**: Built on NATS, Ground Control works seamlessly across geographic locations, edge devices, or multi-region cloud environments. If you can connect to NATS, you can run or monitor Ground Control workflows.

## Components

- **Server** — The Go server is the coordinator. It runs embedded NATS, manages the workflow state machine, and handles all state transitions. It never executes user code.
- **Worker** — Workers execute your workflow code. They connect to the server via NATS, pull step assignments, run the matching handler, and report results back. Workers hold no internal workflow state, so they can crash, restart, or scale without consequence. 
- **Client** — How an application interacts with workflows. It triggers workflow runs, sends events to running workflows, and receives results.
- **CLI** — A command-line and TUI tool for operating and inspecting active and completed workflows.
