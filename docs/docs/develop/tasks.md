---
title: Tasks
---

A task is an async function decorated with `@task`. Tasks handle the side-effecting work inside a step, calling APIs, writing to databases, sending messages. Tasks run in the same process as the step that calls them. There is no remote dispatch and no separate execution environment. You call a task like any other async function and it executes right there. 

Task inputs and outputs are serialized so the engine can record them to the workflow history. The engine records each task's outcome so during the replay, completed tasks are skipped and their recorded result is returned immediately.

## Basic Usage

```python {filename="workflow.py"}
import asyncio
from grctl.worker.task import task
from grctl.worker.context import Context
from grctl.models import Directive
from grctl.workflow import Workflow

order = Workflow(workflow_type="ProcessOrder")

@task
async def charge_payment(amount: float) -> str:
    # Call your payment API here
    return "txn_abc123"

@task
async def send_confirmation(txn_id: str, email: str) -> None:
    # Send confirmation email
    pass

@order.start()
async def start(ctx: Context, amount: float, email: str) -> Directive:
    txn_id = await charge_payment(amount)
    await send_confirmation(txn_id, email)
    return ctx.next.complete({"txn_id": txn_id})
```

Tasks are plain async functions. No `ctx`, no special return type. Call them with `await` like any other coroutine.

## Replay Behavior

When a worker picks up a workflow after a crash, it re-executes the step from the beginning. Tasks that already completed are **not re-executed** — the engine looks up the recorded outcome and returns it immediately.

```
First execution:                  Re-execution after crash:
    charge_payment()  →  runs         charge_payment()  →  skipped, returns "txn_abc123"
    send_confirmation() →  runs       send_confirmation() →  skipped, returns None
```

If a task failed terminally on the first execution, the same error is re-raised on re-execution without calling the function again.

## Retries

Pass a `RetryPolicy` to retry a task on failure:

```python
from grctl.models.directive import RetryPolicy

@task(retry_policy=RetryPolicy(
    max_attempts=5,
    initial_delay_ms=500,
    backoff_multiplier=2.0,
    max_delay_ms=30_000,
))
async def call_external_api(url: str) -> dict:
    # This will be retried up to 5 times on failure
    ...
```

Retries happen within the same step execution. Each failed attempt is recorded before the next one starts, so if the worker crashes mid-retry the next worker picks up from the correct attempt count.

### Retryable and Non-Retryable Errors

Control which error types trigger a retry:

```python
# Only retry these specific errors
@task(retry_policy=RetryPolicy(
    max_attempts=3,
    retryable_errors=["TimeoutError", "ConnectionError"],
))
async def fetch_data(url: str) -> dict:
    ...

# Retry everything except these errors
@task(retry_policy=RetryPolicy(
    max_attempts=3,
    non_retryable_errors=["ValueError", "AuthenticationError"],
))
async def process_record(record_id: str) -> dict:
    ...
```

Error matching uses the exception class name (not the full module path).

If neither `retryable_errors` nor `non_retryable_errors` is set, all errors are retried up to `max_attempts`.

## Error Handling

When a task raises an exception, what happens next depends on whether the task has a retry policy.

**No retry policy:** the exception propagates immediately to the step handler.

**Retry policy exhausted:** after all attempts fail, the final exception propagates to the step handler.

In both cases, you can catch the exception with a normal `try/except` in the step:

```python
@order.step()
async def charge(ctx: Context) -> Directive:
    amount = await ctx.store.get("amount")
    order_id = await ctx.store.get("order_id")

    try:
        txn_id = await charge_payment(amount)
    except PaymentDeclinedError:
        await notify_customer(order_id, "payment_failed")
        return ctx.next.complete({"status": "payment_failed"})

    return ctx.next.complete({"txn_id": txn_id, "status": "charged"})
```

Use `ctx.next.fail(error)` to explicitly fail the workflow with structured error details:

```python
from grctl.models import ErrorDetails

@order.step()
async def charge(ctx: Context) -> Directive:
    amount = await ctx.store.get("amount")

    try:
        txn_id = await charge_payment(amount)
    except Exception as e:
        return ctx.next.fail(ErrorDetails(
            type=type(e).__name__,
            message=f"Payment failed: {e}",
        ))

    return ctx.next.complete({"txn_id": txn_id})
```

If the exception goes uncaught, it propagates out of the step handler and the workflow fails with a `StepFailed` error.

### Compensating Actions

When a task fails partway through a step that already completed other tasks, you need to undo the completed work. Do this in the `except` block before returning a directive:

```python
@order.step()
async def fulfill(ctx: Context) -> Directive:
    order_id = await ctx.store.get("order_id")

    reservation_id = await reserve_inventory(order_id)

    try:
        await charge_payment(order_id)
    except PaymentDeclinedError:
        await release_inventory(reservation_id)  # undo the reservation
        return ctx.next.complete({"status": "payment_failed"})

    return ctx.next.complete({"status": "fulfilled"})
```

Compensating tasks are recorded to history like any other task, so they won't re-run on replay.

## Parallel Tasks

Tasks run in-process, so any standard asyncio concurrency primitive works. `asyncio.gather()`, `asyncio.TaskGroup`, `asyncio.wait()`, manual `asyncio.create_task()`, etc.

```python
@order.step()
async def notify(ctx: Context) -> Directive:
    order_id = await ctx.store.get("order_id")
    email = await ctx.store.get("email")

    # gather
    await asyncio.gather(
        send_email(email, order_id),
        update_crm(order_id),
        log_analytics(order_id),
    )

    # or TaskGroup for structured concurrency
    async with asyncio.TaskGroup() as tg:
        tg.create_task(send_email(email, order_id))
        tg.create_task(update_crm(order_id))

    return ctx.next.complete("done")
```

Each task is recorded independently, so replay correctly skips only the ones that completed.

## Reference

### `@task` Decorator

| Form | Description |
|---|---|
| `@task` | No retries. Single attempt. |
| `@task(retry_policy=...)` | Retry on failure according to policy. |

### `RetryPolicy`

| Field | Type | Default | Description |
|---|---|---|---|
| `max_attempts` | `int | None` | `1` | Total attempts including the first. |
| `initial_delay_ms` | `int | None` | `100` | Delay before the second attempt, in milliseconds. |
| `backoff_multiplier` | `float | None` | `2.0` | Multiplier applied to delay after each attempt. |
| `max_delay_ms` | `int | None` | `5000` | Upper bound on delay, in milliseconds. |
| `jitter` | `float | None` | `0.0` | Random fraction added to delay (`0.0`–`1.0`). |
| `retryable_errors` | `list[str] | None` | `None` | Only retry errors whose class name is in this list. |
| `non_retryable_errors` | `list[str] | None` | `None` | Never retry errors whose class name is in this list. |
