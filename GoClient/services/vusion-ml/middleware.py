from __future__ import annotations

from typing import TYPE_CHECKING

from config import ServiceConfig
from kafka_gateway import KafkaGateway
from keycloak_client import KeycloakVerifier

if TYPE_CHECKING:
    from db_gateway import DbGateway
    from s3_gateway import S3Gateway


class Middleware:
    def __init__(
        self,
        config: ServiceConfig,
        *,
        db: "DbGateway | None" = None,
        s3: "S3Gateway | None" = None,
        kafka: "KafkaGateway | None" = None,
    ) -> None:
        self.keycloak = KeycloakVerifier(config)
        self.kafka = kafka if kafka is not None else KafkaGateway(config)
        self.db = db
        self.s3 = s3

    def readiness(self) -> dict[str, str]:
        status: dict[str, str] = {"keycloak": "down", "kafka": "down"}
        self.keycloak.healthcheck()
        status["keycloak"] = "up"
        self.kafka.healthcheck()
        status["kafka"] = "up"
        if self.db:
            self.db.ping()
            status["postgres"] = "up"
        if self.s3:
            self.s3.head_bucket()
            status["s3"] = "up"
        return status
