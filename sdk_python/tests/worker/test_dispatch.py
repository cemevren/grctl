import dataclasses
from unittest.mock import MagicMock

import msgspec
import pytest
from pydantic import BaseModel

from grctl.worker.codec import CodecRegistry
from grctl.worker.runner import _dispatch_handler
from grctl.workflow.workflow import HandlerSpec


@dataclasses.dataclass
class DCPayload:
    name: str
    value: int


class PydanticPayload(BaseModel):
    name: str
    value: int


class StructPayload(msgspec.Struct):
    name: str
    value: int


@pytest.fixture
def codec():
    return CodecRegistry()


@pytest.fixture
def ctx():
    return MagicMock()


@pytest.mark.asyncio
async def test_no_params_calls_handler_with_ctx_only(codec, ctx):
    received = {}

    async def handler(ctx) -> None:
        received["called"] = True

    spec = HandlerSpec(params={})
    await _dispatch_handler(codec, spec, ctx, handler, None)

    assert received["called"] is True


@pytest.mark.asyncio
async def test_single_pydantic_param_receives_instance(codec, ctx):
    received = {}

    async def handler(ctx, payload: PydanticPayload) -> None:
        received["payload"] = payload

    spec = HandlerSpec(params={"payload": PydanticPayload})
    await _dispatch_handler(codec, spec, ctx, handler, {"payload": {"name": "x", "value": 1}})

    assert isinstance(received["payload"], PydanticPayload)
    assert received["payload"].name == "x"
    assert received["payload"].value == 1


@pytest.mark.asyncio
async def test_single_dataclass_param_receives_instance(codec, ctx):
    received = {}

    async def handler(ctx, payload: DCPayload) -> None:
        received["payload"] = payload

    spec = HandlerSpec(params={"payload": DCPayload})
    await _dispatch_handler(codec, spec, ctx, handler, {"payload": {"name": "y", "value": 2}})

    assert isinstance(received["payload"], DCPayload)
    assert received["payload"].name == "y"
    assert received["payload"].value == 2


@pytest.mark.asyncio
async def test_single_struct_param_receives_instance(codec, ctx):
    received = {}

    async def handler(ctx, payload: StructPayload) -> None:
        received["payload"] = payload

    spec = HandlerSpec(params={"payload": StructPayload})
    await _dispatch_handler(codec, spec, ctx, handler, {"payload": {"name": "z", "value": 3}})

    assert isinstance(received["payload"], StructPayload)
    assert received["payload"].name == "z"
    assert received["payload"].value == 3


@pytest.mark.asyncio
async def test_multiple_typed_params_each_received_correctly(codec, ctx):
    received = {}

    async def handler(ctx, name: str, count: int) -> None:
        received["name"] = name
        received["count"] = count

    spec = HandlerSpec(params={"name": str, "count": int})
    await _dispatch_handler(codec, spec, ctx, handler, {"name": "Alice", "count": 5})

    assert received["name"] == "Alice"
    assert received["count"] == 5


@pytest.mark.asyncio
async def test_missing_required_param_raises_key_error(codec, ctx):
    async def handler(ctx, x: int) -> None: ...

    spec = HandlerSpec(params={"x": int})

    with pytest.raises(KeyError):
        await _dispatch_handler(codec, spec, ctx, handler, {})


@pytest.mark.asyncio
async def test_wrong_type_raises_validation_error(codec, ctx):
    async def handler(ctx, x: int) -> None: ...

    spec = HandlerSpec(params={"x": int})

    with pytest.raises(msgspec.ValidationError):
        await _dispatch_handler(codec, spec, ctx, handler, {"x": "not-an-int"})
