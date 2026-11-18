"""Broker consumers of the bot (Entrega 3: comunicação assíncrona).

`python main.py worker` — the collection worker. Listens on the
``bot.collect`` queue, bound to ``collect.requested`` on the ``price.events``
topic exchange. price-back publishes that event when the Admin panel asks
for a collection, answers 202 right away and never learns who consumes it.
For each message the worker runs the same collection as POST /collect
(nextprice.collect_api.run_collection) and publishes ``collect.finished``
with the outcome, which price-back uses to close the job.

Several workers can run at once: they compete for the same queue and,
with prefetch=1, each message goes to exactly one of them (point-to-point
with load balancing).

`python main.py audit` — a second, independent subscriber bound to
``collect.*``. It gets its own copy of every collection event and only logs
it, to show publish/subscribe on the same exchange.
"""

from __future__ import annotations

import functools
import json
import logging
import os
import threading
import time
import uuid
from datetime import datetime, timezone

import pika
from dotenv import find_dotenv, load_dotenv
from pika.adapters.blocking_connection import BlockingChannel
from pydantic import BaseModel, Field, ValidationError

from nextprice import collect_api

logger = logging.getLogger(__name__)

EXCHANGE = "price.events"
COLLECT_REQUESTED_KEY = "collect.requested"
COLLECT_FINISHED_KEY = "collect.finished"

COLLECT_QUEUE = "bot.collect"
# Messages the worker can't even parse are dead-lettered here, so they stay
# visible for inspection instead of being retried forever or lost.
DEAD_LETTER_EXCHANGE = "price.events.dlx"
COLLECT_DLQ = "bot.collect.dlq"

AUDIT_QUEUE = "audit.collect"


class CollectRequestedEvent(BaseModel):
    event_id: str
    job_id: str = Field(min_length=1)
    market_id: str
    market_name: str = Field(min_length=1)
    url: str = Field(min_length=1)
    requested_at: datetime | None = None


class InvalidMessage(Exception):
    """The message body is not a valid collect.requested event."""


def handle_collect_requested(body: bytes) -> dict:
    """Processes one collect.requested message and returns the
    collect.finished event to publish. Business failures (site down, crash
    inside the collection) become a ``failed`` event, so the job is always
    closed; only an unparseable message raises InvalidMessage."""
    try:
        event = CollectRequestedEvent.model_validate_json(body)
    except ValidationError as exc:
        raise InvalidMessage(str(exc)) from exc

    logger.info(
        "[worker pid=%d] job %s — coletando %s (%s)",
        os.getpid(),
        event.job_id,
        event.market_name,
        event.url,
    )

    finished = {
        "event_id": uuid.uuid4().hex,
        "job_id": event.job_id,
        "status": "succeeded",
        "blocks_found": 0,
        "products_extracted": 0,
        "products_inserted": 0,
        "error": None,
    }
    try:
        result = collect_api.run_collection(event.market_name, event.url)
        finished.update(
            blocks_found=result.blocks_found,
            products_extracted=result.products_extracted,
            products_inserted=len(result.inserted),
        )
    except collect_api.CollectError:
        finished.update(status="failed", error="url_unreachable")
    except Exception:
        logger.exception("[worker] job %s falhou com erro inesperado", event.job_id)
        finished.update(status="failed", error="internal_error")

    finished["finished_at"] = datetime.now(timezone.utc).isoformat()
    return finished


def _publish(channel: BlockingChannel, routing_key: str, event: dict) -> None:
    channel.basic_publish(
        exchange=EXCHANGE,
        routing_key=routing_key,
        body=json.dumps(event).encode(),
        properties=pika.BasicProperties(
            content_type="application/json",
            delivery_mode=pika.DeliveryMode.Persistent,
            message_id=event["event_id"],
        ),
    )


def _finish(channel: BlockingChannel, delivery_tag: int, finished: dict | None) -> None:
    """Runs on the connection's thread: publishes the result, then acks."""
    if finished is None:
        channel.basic_nack(delivery_tag=delivery_tag, requeue=False)
        return
    _publish(channel, COLLECT_FINISHED_KEY, finished)
    channel.basic_ack(delivery_tag=delivery_tag)
    logger.info(
        "[worker pid=%d] job %s concluído: %s (%d inseridos)",
        os.getpid(),
        finished["job_id"],
        finished["status"],
        finished["products_inserted"],
    )


def _on_collect_requested(connection: pika.BlockingConnection, channel, method, _props, body):
    # A collection can take minutes (browser + AI). Running it on the
    # connection's thread would stop heartbeats and the broker would drop
    # the connection, so it runs in a thread and hands the publish/ack back
    # with add_callback_threadsafe (pika channels aren't thread-safe).
    def work() -> None:
        try:
            finished = handle_collect_requested(body)
        except InvalidMessage as exc:
            logger.error("[worker] mensagem inválida descartada para a DLQ: %s", exc)
            finished = None
        connection.add_callback_threadsafe(
            functools.partial(_finish, channel, method.delivery_tag, finished)
        )

    threading.Thread(target=work, daemon=True).start()


def declare_topology(channel: BlockingChannel) -> None:
    """Idempotently declares the exchange, the worker's queue and its DLQ."""
    channel.exchange_declare(exchange=EXCHANGE, exchange_type="topic", durable=True)
    channel.exchange_declare(exchange=DEAD_LETTER_EXCHANGE, exchange_type="fanout", durable=True)
    channel.queue_declare(queue=COLLECT_DLQ, durable=True)
    channel.queue_bind(queue=COLLECT_DLQ, exchange=DEAD_LETTER_EXCHANGE)
    channel.queue_declare(
        queue=COLLECT_QUEUE,
        durable=True,
        arguments={"x-dead-letter-exchange": DEAD_LETTER_EXCHANGE},
    )
    channel.queue_bind(queue=COLLECT_QUEUE, exchange=EXCHANGE, routing_key=COLLECT_REQUESTED_KEY)


def _connection_params() -> pika.URLParameters:
    url = os.environ.get("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
    params = pika.URLParameters(url)
    params.heartbeat = 60
    return params


def _run_forever(consume) -> None:
    """Calls consume(connection, channel) and reconnects with backoff when
    the broker is down or the connection drops."""
    backoff = 1
    while True:
        try:
            connection = pika.BlockingConnection(_connection_params())
            backoff = 1
            channel = connection.channel()
            consume(connection, channel)
        except KeyboardInterrupt:
            logger.info("Encerrando.")
            return
        except pika.exceptions.AMQPError as exc:
            logger.warning(
                "Sem conexão com o RabbitMQ (%s) — tentando de novo em %ds", exc, backoff
            )
            time.sleep(backoff)
            backoff = min(backoff * 2, 30)


def _consume_collect(connection: pika.BlockingConnection, channel: BlockingChannel) -> None:
    declare_topology(channel)
    # One unacked message per worker: the next one goes to whichever worker
    # is free, which is what balances the load across instances.
    channel.basic_qos(prefetch_count=1)
    channel.basic_consume(
        queue=COLLECT_QUEUE,
        on_message_callback=functools.partial(_on_collect_requested, connection),
    )
    logger.info("[worker pid=%d] aguardando mensagens em '%s'...", os.getpid(), COLLECT_QUEUE)
    channel.start_consuming()


def _consume_audit(_connection: pika.BlockingConnection, channel: BlockingChannel) -> None:
    channel.exchange_declare(exchange=EXCHANGE, exchange_type="topic", durable=True)
    channel.queue_declare(queue=AUDIT_QUEUE, durable=True)
    channel.queue_bind(queue=AUDIT_QUEUE, exchange=EXCHANGE, routing_key="collect.*")

    def on_message(ch, method, _props, body):
        logger.info("[audit] %s %s", method.routing_key, body.decode(errors="replace"))
        ch.basic_ack(delivery_tag=method.delivery_tag)

    channel.basic_consume(queue=AUDIT_QUEUE, on_message_callback=on_message)
    logger.info("[audit] registrando todos os eventos 'collect.*' em '%s'...", AUDIT_QUEUE)
    channel.start_consuming()


def main() -> None:
    load_dotenv(find_dotenv(usecwd=True))
    logging.basicConfig(level=logging.INFO, format="%(message)s")
    _run_forever(_consume_collect)


def audit_main() -> None:
    load_dotenv(find_dotenv(usecwd=True))
    logging.basicConfig(level=logging.INFO, format="%(message)s")
    _run_forever(_consume_audit)


if __name__ == "__main__":
    main()
