import json
from typing import Any

from confluent_kafka import Producer
from confluent_kafka.admin import AdminClient

from config import ServiceConfig


class KafkaGateway:
    """Thin producer wrapper over the same Redpanda/Kafka broker the Go API uses.

    In Docker Compose use the internal listener, e.g. ``kafka:9092`` (same as
    the Go API). From the host only, use ``localhost:9093`` (EXTERNAL listener).
    """

    def __init__(self, config: ServiceConfig) -> None:
        kafka_conf = {"bootstrap.servers": config.kafka_brokers}
        self.topic_analysis = config.kafka_topic_analysis
        self._producer = Producer(kafka_conf)
        self._admin = AdminClient(kafka_conf)

    def healthcheck(self) -> None:
        self._admin.list_topics(timeout=15)

    def publish_analysis_request(self, payload: dict[str, Any], user_sub: str | None) -> None:
        key = (user_sub or "anonymous").encode("utf-8")
        self._producer.produce(
            topic=self.topic_analysis,
            key=key,
            value=json.dumps(payload).encode("utf-8"),
            headers={"event_type": "analysis_requested"},
        )
        self._producer.poll(0)

    def publish_json(self, topic: str, key: str | None, payload: dict[str, Any]) -> None:
        """Produce a JSON message to an arbitrary topic (retries / DLQ)."""
        kb = (key or "").encode("utf-8")
        self._producer.produce(
            topic=topic,
            key=kb,
            value=json.dumps(payload).encode("utf-8"),
        )
        self._producer.poll(0)

    def close(self) -> None:
        self._producer.flush(5)
