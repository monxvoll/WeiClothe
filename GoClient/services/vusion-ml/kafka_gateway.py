import json
from typing import Any

from confluent_kafka import Producer
from confluent_kafka.admin import AdminClient

from config import ServiceConfig


class KafkaGateway:
    """Thin producer wrapper over the same Redpanda/Kafka broker the Go API uses.

    Connects to the EXTERNAL listener (default ``localhost:9093``) defined in
    docker-compose. The ``KAFKA_BROKERS`` env var must match the value used by
    the Go service.
    """

    def __init__(self, config: ServiceConfig) -> None:
        kafka_conf = {"bootstrap.servers": config.kafka_brokers}
        self.topic_analysis = config.kafka_topic_analysis
        self._producer = Producer(kafka_conf)
        self._admin = AdminClient(kafka_conf)

    def healthcheck(self) -> None:
        self._admin.list_topics(timeout=5)

    def publish_analysis_request(self, payload: dict[str, Any], user_sub: str | None) -> None:
        key = (user_sub or "anonymous").encode("utf-8")
        self._producer.produce(
            topic=self.topic_analysis,
            key=key,
            value=json.dumps(payload).encode("utf-8"),
            headers={"event_type": "analysis_requested"},
        )
        self._producer.poll(0)

    def close(self) -> None:
        self._producer.flush(5)
