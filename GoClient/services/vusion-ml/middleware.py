from config import ServiceConfig
from kafka_gateway import KafkaGateway
from keycloak_client import KeycloakVerifier


class Middleware:
    def __init__(self, config: ServiceConfig) -> None:
        self.keycloak = KeycloakVerifier(config)
        self.kafka = KafkaGateway(config)

    def readiness(self) -> dict[str, str]:
        status: dict[str, str] = {"keycloak": "down", "kafka": "down"}
        self.keycloak.healthcheck()
        status["keycloak"] = "up"
        self.kafka.healthcheck()
        status["kafka"] = "up"
        return status
