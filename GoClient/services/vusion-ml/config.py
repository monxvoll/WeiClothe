import os
from dataclasses import dataclass
from pathlib import Path

from dotenv import load_dotenv

load_dotenv(Path(__file__).resolve().parent / ".env")


@dataclass(frozen=True)
class ServiceConfig:
    kafka_brokers: str
    kafka_topic_analysis: str
    kafka_topic_dlq: str
    kafka_group_id: str
    keycloak_base_url: str
    keycloak_realm: str
    keycloak_client_id: str
    keycloak_client_secret: str
    strict_startup: bool
    request_timeout_seconds: float
    keycloak_bootstrap_attempts: int
    keycloak_bootstrap_delay_seconds: float
    enable_analysis_consumer: bool
    # Postgres (central DB / Aurora)
    db_host: str
    db_port: int
    db_user: str
    db_password: str
    db_name: str
    db_sslmode: str
    # S3 / MinIO
    aws_access_key_id: str
    aws_secret_access_key: str
    aws_region: str
    s3_bucket: str
    s3_endpoint_url: str
    s3_public_base_url: str
    # ML
    yolo_model: str
    clip_model: str
    max_image_size_bytes: int

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

        enable_analysis_consumer = os.getenv("ENABLE_ANALYSIS_CONSUMER", "true").strip().lower() == "true"

        db_host = os.getenv("DB_HOST", "").strip()
        db_port_raw = os.getenv("DB_PORT", "5432").strip()
        db_user = os.getenv("DB_USER", "").strip()
        db_password = os.getenv("DB_PASSWORD", "").strip()
        db_name = os.getenv("DB_NAME", "").strip()
        db_sslmode = os.getenv("DB_SSLMODE", "disable").strip()

        aws_access_key_id = os.getenv("AWS_ACCESS_KEY_ID", "").strip()
        aws_secret_access_key = os.getenv("AWS_SECRET_ACCESS_KEY", "").strip()
        aws_region = os.getenv("AWS_REGION", "us-east-1").strip()
        s3_bucket = os.getenv("S3_BUCKET", "").strip()
        s3_endpoint_url = os.getenv("S3_ENDPOINT_URL", "").strip()
        s3_public_base_url = os.getenv("S3_PUBLIC_BASE_URL", "").strip()

        if enable_analysis_consumer:
            missing = [
                name
                for name, val in [
                    ("DB_HOST", db_host),
                    ("DB_USER", db_user),
                    ("DB_PASSWORD", db_password),
                    ("DB_NAME", db_name),
                    ("S3_BUCKET", s3_bucket),
                ]
                if not val
            ]
            if missing:
                raise RuntimeError(
                    "ENABLE_ANALYSIS_CONSUMER=true requires: " + ", ".join(missing)
                    + " (optional AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY on EC2 use IAM role)"
                )

        try:
            db_port = int(db_port_raw)
        except ValueError as exc:
            raise RuntimeError(f"Invalid DB_PORT: {db_port_raw}") from exc

        return cls(
            kafka_brokers=kafka_brokers,
            kafka_topic_analysis=os.getenv("KAFKA_TOPIC_ANALYSIS", "vusion.analysis.request").strip(),
            kafka_topic_dlq=os.getenv("KAFKA_TOPIC_ANALYSIS_DLQ", "vusion.analysis.dlq").strip(),
            kafka_group_id=os.getenv("KAFKA_GROUP_ID", "vusion-ml-analysis").strip(),
            keycloak_base_url=keycloak_base_url,
            keycloak_realm=keycloak_realm,
            keycloak_client_id=keycloak_client_id,
            keycloak_client_secret=keycloak_client_secret,
            strict_startup=os.getenv("MIDDLEWARE_STRICT_STARTUP", "true").strip().lower() == "true",
            request_timeout_seconds=float(os.getenv("MIDDLEWARE_HTTP_TIMEOUT_SECONDS", "10")),
            keycloak_bootstrap_attempts=int(os.getenv("KEYCLOAK_BOOTSTRAP_ATTEMPTS", "40")),
            keycloak_bootstrap_delay_seconds=float(os.getenv("KEYCLOAK_BOOTSTRAP_DELAY_SEC", "3")),
            enable_analysis_consumer=enable_analysis_consumer,
            db_host=db_host,
            db_port=db_port,
            db_user=db_user,
            db_password=db_password,
            db_name=db_name,
            db_sslmode=db_sslmode,
            aws_access_key_id=aws_access_key_id,
            aws_secret_access_key=aws_secret_access_key,
            aws_region=aws_region,
            s3_bucket=s3_bucket,
            s3_endpoint_url=s3_endpoint_url,
            s3_public_base_url=s3_public_base_url,
            yolo_model=os.getenv("YOLO_MODEL", "yolov8n-seg.pt").strip(),
            clip_model=os.getenv("CLIP_MODEL", "openai/clip-vit-base-patch32").strip(),
            max_image_size_bytes=int(os.getenv("MAX_IMAGE_SIZE_MB", "20")) * 1024 * 1024,
        )
