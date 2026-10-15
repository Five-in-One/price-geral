from __future__ import annotations

import json
from types import SimpleNamespace
from unittest.mock import MagicMock

import pytest

from nextprice import collect_api, worker
from nextprice.cli import main as cli_main


def _event(**overrides) -> bytes:
    body = {
        "event_id": "e1",
        "job_id": "job-1",
        "market_id": "mercado-novo-abc123",
        "market_name": "Mercado Novo",
        "url": "https://mercadonovo.com.br/",
        "requested_at": "2026-10-13T18:00:00Z",
    }
    body.update(overrides)
    return json.dumps(body).encode()


def test_handle_collect_requested_runs_collection_and_reports_success(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    calls = []

    def fake_run(market_name: str, url: str) -> collect_api.CollectResult:
        calls.append((market_name, url))
        return collect_api.CollectResult(
            blocks_found=10, products_extracted=8, inserted=[{}, {}, {}]
        )

    monkeypatch.setattr(collect_api, "run_collection", fake_run)

    finished = worker.handle_collect_requested(_event())

    assert calls == [("Mercado Novo", "https://mercadonovo.com.br/")]
    assert finished["job_id"] == "job-1"
    assert finished["status"] == "succeeded"
    assert finished["blocks_found"] == 10
    assert finished["products_extracted"] == 8
    assert finished["products_inserted"] == 3
    assert finished["error"] is None
    assert finished["event_id"] and finished["finished_at"]


def test_handle_collect_requested_unreachable_url_reports_failed(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def fake_run(market_name: str, url: str):
        raise collect_api.CollectError("timeout")

    monkeypatch.setattr(collect_api, "run_collection", fake_run)

    finished = worker.handle_collect_requested(_event())

    assert finished["status"] == "failed"
    assert finished["error"] == "url_unreachable"


def test_handle_collect_requested_unexpected_error_still_closes_job(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def fake_run(market_name: str, url: str):
        raise RuntimeError("boom")

    monkeypatch.setattr(collect_api, "run_collection", fake_run)

    finished = worker.handle_collect_requested(_event())

    assert finished["status"] == "failed"
    assert finished["error"] == "internal_error"


@pytest.mark.parametrize("body", [b"not json", _event(job_id=""), _event(url="")])
def test_handle_collect_requested_invalid_message_raises(body: bytes) -> None:
    with pytest.raises(worker.InvalidMessage):
        worker.handle_collect_requested(body)


def test_finish_publishes_collect_finished_then_acks() -> None:
    channel = MagicMock()
    finished = {"event_id": "e2", "job_id": "job-1", "status": "succeeded", "products_inserted": 3}

    worker._finish(channel, delivery_tag=7, finished=finished)

    channel.basic_publish.assert_called_once()
    kwargs = channel.basic_publish.call_args.kwargs
    assert kwargs["exchange"] == worker.EXCHANGE
    assert kwargs["routing_key"] == worker.COLLECT_FINISHED_KEY
    assert json.loads(kwargs["body"]) == finished
    channel.basic_ack.assert_called_once_with(delivery_tag=7)
    channel.basic_nack.assert_not_called()


def test_finish_invalid_message_is_dead_lettered() -> None:
    channel = MagicMock()

    worker._finish(channel, delivery_tag=7, finished=None)

    channel.basic_nack.assert_called_once_with(delivery_tag=7, requeue=False)
    channel.basic_publish.assert_not_called()


def test_on_collect_requested_hands_ack_back_to_connection_thread(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setattr(
        collect_api,
        "run_collection",
        lambda market_name, url: collect_api.CollectResult(1, 1, [{}]),
    )

    class InlineThread:
        def __init__(self, target, daemon):
            self.target = target

        def start(self):
            self.target()

    monkeypatch.setattr(worker.threading, "Thread", InlineThread)
    connection, channel = MagicMock(), MagicMock()
    connection.add_callback_threadsafe.side_effect = lambda cb: cb()

    worker._on_collect_requested(
        connection, channel, SimpleNamespace(delivery_tag=3), None, _event()
    )

    connection.add_callback_threadsafe.assert_called_once()
    channel.basic_ack.assert_called_once_with(delivery_tag=3)


def test_declare_topology_binds_queue_to_collect_requested() -> None:
    channel = MagicMock()

    worker.declare_topology(channel)

    channel.queue_bind.assert_any_call(
        queue=worker.COLLECT_QUEUE,
        exchange=worker.EXCHANGE,
        routing_key=worker.COLLECT_REQUESTED_KEY,
    )
    queue_args = {
        c.kwargs["queue"]: c.kwargs.get("arguments") for c in channel.queue_declare.call_args_list
    }
    assert queue_args[worker.COLLECT_QUEUE] == {
        "x-dead-letter-exchange": worker.DEAD_LETTER_EXCHANGE
    }


def test_cli_dispatches_worker(monkeypatch: pytest.MonkeyPatch) -> None:
    chamado = MagicMock()
    monkeypatch.setattr(worker, "main", chamado)

    cli_main.main(["worker"])

    chamado.assert_called_once()
