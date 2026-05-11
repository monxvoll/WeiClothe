import atexit
import logging
import os
from typing import Any

from flask import Flask, jsonify

from config import ServiceConfig
from db_gateway import DbGateway
from kafka_consumer import AnalysisConsumerThread
from kafka_gateway import KafkaGateway
from middleware import Middleware
from routes import ai_bp
from s3_gateway import S3Gateway


def create_app() -> Flask:
    logging.basicConfig(level=os.getenv("LOG_LEVEL", "INFO"))
    app = Flask(__name__)

    from flask_cors import CORS

    # Restrict CORS to explicitly allowed origins. Wildcard is insecure for a credentialed API.
    # Set CORS_ALLOWED_ORIGINS to a comma-separated list in .env (e.g. https://app.example.com).
    # Defaults to empty string which means no origin is allowed — override in every environment.
    raw_origins = os.getenv("CORS_ALLOWED_ORIGINS", "").strip()
    allowed_origins = [o.strip() for o in raw_origins.split(",") if o.strip()] if raw_origins else []
    CORS(app, origins=allowed_origins)

    config = ServiceConfig.from_env()
    kafka_gw = KafkaGateway(config)

    db = None
    s3gw = None
    consumer: AnalysisConsumerThread | None = None

    if config.enable_analysis_consumer:
        db = DbGateway(config)
        s3gw = S3Gateway(config)
        consumer = AnalysisConsumerThread(config, db, s3gw, kafka_gw)
        consumer.start()
        atexit.register(consumer.stop)
        atexit.register(db.close)

    middleware = Middleware(config, db=db, s3=s3gw, kafka=kafka_gw)
    app.config["middleware"] = middleware
    app.config["service_config"] = config
    app.register_blueprint(ai_bp)

    # Cerrar Kafka al terminar el proceso. NO usar teardown_appcontext: con Gunicorn
    # eso se ejecuta tras cada request y deja el producer cerrado.
    atexit.register(middleware.kafka.close)

    @app.get("/readyz")
    def ready() -> Any:
        try:
            status = middleware.readiness()
            return jsonify({"status": "ready", "middleware": status}), 200
        except Exception as exc:  # noqa: BLE001
            app.logger.error("readiness check failed: %s", exc)
            return jsonify({"status": "not_ready", "error": "service dependencies not ready"}), 503

    @app.get("/healthz")
    def health() -> Any:
        return jsonify({"status": "up"}), 200

    if config.strict_startup:
        middleware.readiness()
        app.logger.info("Middleware startup checks passed")

    return app
