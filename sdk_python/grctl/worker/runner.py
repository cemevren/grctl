import functools
import traceback
from collections.abc import Awaitable, Callable
from datetime import UTC, datetime
from typing import Any

import msgspec

from grctl.logging_config import get_logger
from grctl.models import (
    Directive,
    ErrorDetails,
    Event,
    HistoryKind,
    Start,
    Step,
    StepStarted,
)
from grctl.models.directive import StepResult
from grctl.models.run_info_helper import RunInfoManager
from grctl.worker.codec import CodecRegistry
from grctl.worker.context import Context
from grctl.worker.errors import NextDirectiveMissingError
from grctl.worker.runtime import StepRuntime, set_step_runtime
from grctl.workflow.workflow import HandlerSpec

logger = get_logger(__name__)


def workflow_error_handler(func):  # noqa: ANN001, ANN201
    """Handle exceptions in workflow methods and publish fail directive to server."""

    @functools.wraps(func)
    async def wrapper(self, *args: Any, **kwargs: Any):  # noqa: ANN001, ANN202
        try:
            return await func(self, *args, **kwargs)
        except Exception as e:
            stack_trace = traceback.format_exc()
            logger.exception(f"Workflow execution failed in {func.__name__}")

            ctx = self.runtime.get_step_context()
            fail_directive = ctx.next.fail(
                ErrorDetails(
                    type=type(e).__name__,
                    message=str(e),
                    stack_trace=stack_trace,
                ),
            )
            await self.runtime.publisher.publish_next_directive(self.runtime.run_info, fail_directive)
            raise

    return wrapper


"""
Runner pulls the messages from NATS for a single workflow. no need for the distribution logic for now.
It's not efficient to have multiple workers pulling messages for the same workflow run.
But it's okay for now as we are focusing on correctness and not efficiency.
"""


class WorkflowRunner:
    """Orchestrates workflow run lifecycle.

    Manages the complete workflow run lifecycle including:
    - Validating run requests
    - Publishing lifecycle events (started, completed/failed/timeout)
    - Executing workflow with timeout enforcement
    - Tracking execution duration
    """

    _result = None

    def __init__(self, runtime: StepRuntime) -> None:
        self.runtime = runtime
        set_step_runtime(runtime)
        self.workflow = runtime.workflow

    async def handle_directive(self, directive: Directive) -> None:
        """Dispatch directive to appropriate handler."""
        msg = directive.msg
        if isinstance(msg, Start):
            await self.handle_start(msg.input)
        elif isinstance(msg, Event):
            await self.handle_event(msg.event_name, msg.payload)
        elif isinstance(msg, Step):
            await self.handle_step(msg)
        else:
            logger.warning(f"Unknown command type: {type(directive)}")

    @workflow_error_handler
    async def handle_start(self, payload: Any | None) -> None:
        start_config = self.workflow._start_handler  # noqa: SLF001
        if start_config is None:
            raise ValueError("Workflow start handler is not defined.")

        self.runtime.run_info = RunInfoManager.start(self.runtime.run_info, datetime.now(UTC))
        self.runtime.step_name = "start"
        await _execute_step(self.runtime, start_config.handler, start_config.spec, payload)

    @workflow_error_handler
    async def handle_event(self, event_name: str, payload: Any | None) -> None:
        """Handle an event by executing its corresponding handler."""
        handler_config = self.workflow._on_event_handlers.get(event_name)  # noqa: SLF001
        if handler_config is None:
            logger.warning(f"No handler registered for event '{event_name}'")
            return

        self.runtime.step_name = event_name
        await _execute_step(self.runtime, handler_config.handler, handler_config.spec, payload)

    @workflow_error_handler
    async def handle_step(self, step: Step) -> None:
        """Execute workflow step."""
        logger.debug(f"Executing step: {step.step_name} for run {self.runtime.run_info.id}")

        step_config = self.workflow._step_handlers.get(step.step_name)  # noqa: SLF001
        if step_config is None:
            raise ValueError(f"Step handler '{step.step_name}' is not defined.")
        self.runtime.step_name = step.step_name
        await _execute_step(self.runtime, step_config.handler, step_config.spec, None)

    def _get_event_name(self, handler: Any) -> str | None:
        """Check if handler is an @on_event handler and return its event name."""
        for event_name, event_config in self.workflow._on_event_handlers.items():  # noqa: SLF001
            if handler == event_config.handler:
                return event_name
        return None


async def _dispatch_handler(
    codec: CodecRegistry,
    spec: HandlerSpec,
    ctx: Context,
    handler: Callable[..., Awaitable[Directive]],
    payload: Any | None,
) -> Directive:
    if not spec.params:
        return await handler(ctx)
    if not isinstance(payload, dict):
        raise TypeError(f"Handler expects params {list(spec.params)} but payload is not a dict: {type(payload)}")
    typed_kwargs = {
        name: msgspec.convert(payload[name], param_type, dec_hook=codec.dec_hook)
        for name, param_type in spec.params.items()
    }
    return await handler(ctx, **typed_kwargs)


async def _execute_step(
    runtime: StepRuntime,
    handler: Callable[..., Awaitable[Directive]],
    spec: HandlerSpec,
    payload: Any | None,
) -> None:
    ctx = runtime.get_step_context()
    start_time = datetime.now(UTC)

    await _publish_step_started_event(runtime)

    directive = await _dispatch_handler(runtime.codec, spec, ctx, handler, payload)

    await _publish_next_directive(runtime, directive, start_time)


async def _publish_step_started_event(runtime: StepRuntime) -> None:
    # Step started event will be send by the worker
    if runtime.step_history is None or len(runtime.step_history) == 0:
        await runtime.record(
            kind=HistoryKind.step_started,
            payload=StepStarted(step_name=runtime.step_name),
            operation_id="",
        )


async def _publish_next_directive(
    runtime: StepRuntime,
    directive: Directive,
    step_start_time: datetime | None = None,
) -> None:
    """Process Directive returned by a handler."""
    if not isinstance(directive, Directive):
        raise NextDirectiveMissingError(f"Step did not return a Directive. {directive=}", runtime.step_name)

    if step_start_time is not None and isinstance(directive.msg, StepResult):
        directive.msg.duration_ms = int((datetime.now(UTC) - step_start_time).total_seconds() * 1000)

    pending_updates = runtime.store.get_pending_updates()
    if pending_updates:
        directive.kv_revs = pending_updates

    await runtime.publisher.publish_next_directive(runtime.run_info, directive)
