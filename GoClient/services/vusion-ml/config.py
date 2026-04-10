import os
from dataclasses import dataclass
from pathlib import Path

from dotenv import load_dotenv

load_dotenv(Path(__file__).resolve().parent / ".env")


@dataclass(frozen=True)
class ServiceConfig:
    kafka_brokers: str
    kafka_topic_analysis: str
    keycloak_base_url: str
    keycloak_realm: str
    keycloak_client_id: str
    keycloak_client_secret: str
    strict_startup: bool
    request_timeout_seconds: float
    keycloak_bootstrap_attempts: int
    keycloak_bootstrap_delay_seconds: float

    @classmethod
    def from_env(cls) -> "ServiceConfig":
        kafka_brokers = os.getenv("KAFKA_BROKERS", "").strip()
        keycloak_base_url = os.getenv("KEYCLOAK_BASE_URL", "").strip().rstrip("/")
        keycloak_realm = os.getenv("KEYCLOAK_REALM", "").strip()
        keycloak_client_id = os.getenv("KEYCLOAK_CLIENT_ID", "").strip()
        keycloak_client_secret = os.getenv("KEYCLOAK_CLIENT_SECRET", "").strip()
        if not all([kafka_brokers, keycloak_base_url, keycloak_realm, keycloak_client_id, keycloak_client_secret]):
            raise RuntimeError(
                "Missing required env vars: KAFKA_BROKERS, KEYCLOAK_BASE_URL, "
                "KEYCLOAK_REALM, KEYCLOAK_CLIENT_ID, KEYCLOAK_CLIENT_SECRET"
            )

        return cls(
            kafka_brokers=kafka_brokers,
            kafka_topic_analysis=os.getenv("KAFKA_TOPIC_ANALYSIS", "vusion.analysis.request").strip(),
            keycloak_base_url=keycloak_base_url,
            keycloak_realm=keycloak_realm,
            keycloak_client_id=keycloak_client_id,
            keycloak_client_secret=keycloak_client_secret,
            strict_startup=os.getenv("MIDDLEWARE_STRICT_STARTUP", "true").strip().lower() == "true",
            request_timeout_seconds=float(os.getenv("MIDDLEWARE_HTTP_TIMEOUT_SECONDS", "10")),
            keycloak_bootstrap_attempts=int(os.getenv("KEYCLOAK_BOOTSTRAP_ATTEMPTS", "40")),
            keycloak_bootstrap_delay_seconds=float(os.getenv("KEYCLOAK_BOOTSTRAP_DELAY_SEC", "3")),
        )
