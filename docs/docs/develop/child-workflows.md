---
title: Child Workflows
---

A workflow can launch other workflows as children. The parent and child run independently. Each has its own state and step execution. The parent typically waits for a signal back from the child by returning `ctx.next.wait_for_event()` and listening for the child to call `ctx.send_to_parent()`.

## Starting a Child Workflow

Call `await ctx.start()` from any handler to launch a child workflow:

```python {filename="workflow.py"}
from datetime import timedelta

@order_wf.event()
async def send_to_payment(ctx: Context) -> Directive:
    amount = await ctx.store.get("amount", int)

    payment_handle = await ctx.start(
        "Payment",
        workflow_id="payment-order-abc",
        workflow_input={"amount": amount},
        workflow_timeout=timedelta(minutes=5),
    )

    ctx.store.put("payment_run_id", payment_handle.run_info.id)
    return ctx.next.wait_for_event()
```

`ctx.start()` returns a `WorkflowHandle`. The child starts immediately and runs in parallel with the parent.

The `workflow_id` is a stable identifier for the child. Use it to prevent duplicate starts if the parent's step re-executes (it acts as a deduplication key).

## Sending an Event to the Parent

From inside a child workflow, call `await ctx.send_to_parent()` to notify the parent:

```python
@payment_wf.step()
async def payment_record(ctx: Context) -> Directive:
    transaction_id = await ctx.store.get("transaction_id")
    amount = await ctx.store.get("amount")

    status = await record_transaction(transaction_id, amount)

    await ctx.send_to_parent(
        event_name="payment_completed",
        payload={"status": status, "transaction_id": transaction_id},
    )
    return ctx.next.complete({"status": status})
```

The event is delivered to the parent's inbox. If the parent is in `WaitEvent` state, the event is dispatched immediately. If not, it waits in the inbox until the parent transitions to `WaitEvent`.

## Handling the Child's Response

Register an event handler on the parent workflow to process the child's signal:

```python
@order_wf.event(name="payment_completed")
async def handle_payment_result(ctx: Context, status: str, transaction_id: str) -> Directive:
    ctx.store.put("payment_status", status)
    ctx.store.put("transaction_id", transaction_id)
    return ctx.next.complete(f"order done, txn={transaction_id}")
```

The payload dict sent by `ctx.send_to_parent()` is unpacked as keyword arguments. The event handler's parameter names must match the payload keys.

## Complete Example

A parent `Order` workflow starts a child `Payment` workflow and waits for its result:

```python
payment_wf = Workflow(workflow_type="Payment")
order_wf = Workflow(workflow_type="Order")

# --- Child workflow ---

@payment_wf.start()
async def payment_start(ctx: Context, amount: float) -> Directive:
    ctx.store.put("amount", amount)
    return ctx.next.step(payment_process)

@payment_wf.step()
async def payment_process(ctx: Context) -> Directive:
    amount = await ctx.store.get("amount")
    txn_id = await charge_card(amount)
    ctx.store.put("txn_id", txn_id)
    return ctx.next.step(payment_done)

@payment_wf.step()
async def payment_done(ctx: Context) -> Directive:
    txn_id = await ctx.store.get("txn_id")
    await ctx.send_to_parent("payment_completed", payload={"txn_id": txn_id})
    return ctx.next.complete(txn_id)

# --- Parent workflow ---

@order_wf.start()
async def order_start(ctx: Context, order_id: str, amount: float) -> Directive:
    ctx.store.put("order_id", order_id)
    ctx.store.put("amount", amount)
    return ctx.next.wait_for_event()

@order_wf.event()
async def start_payment(ctx: Context) -> Directive:
    amount = await ctx.store.get("amount")
    order_id = await ctx.store.get("order_id")

    await ctx.start(
        "Payment",
        workflow_id=f"payment-{order_id}",
        workflow_input={"amount": amount},
    )
    return ctx.next.wait_for_event()

@order_wf.event(name="payment_completed")
async def on_payment_done(ctx: Context, txn_id: str) -> Directive:
    ctx.store.put("txn_id", txn_id)
    return ctx.next.complete(txn_id)
```

Both workflows are registered on the same worker:

```python
worker = Worker(workflows=[order_wf, payment_wf], connection=connection)
```

## Reference

### `await ctx.start()`

| Parameter | Type | Default | Description |
|---|---|---|---|
| `workflow_type` | `str` | — | The `workflow_type` of the child workflow to start. |
| `workflow_id` | `str` | — | Stable identifier for the child run. Used for deduplication. |
| `workflow_input` | `dict \| None` | `None` | Input passed to the child's start handler. |
| `workflow_timeout` | `timedelta \| None` | `None` | Maximum duration for the child workflow. |

Returns a `WorkflowHandle` with a `run_info` field containing the child's run metadata.

### `await ctx.send_to_parent()`

| Parameter | Type | Default | Description |
|---|---|---|---|
| `event_name` | `str` | — | Name of the event to deliver to the parent. Must match a `@workflow.event()` handler on the parent. |
| `payload` | `Any \| None` | `None` | Data to pass to the parent's event handler. |

Raises `RuntimeError` if the workflow has no parent.
