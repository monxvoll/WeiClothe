import atexit
import logging
import os
from typing import Any

from flask import Flask, jsonify
from flask_cors import CORS

from config import ServiceConfig
from middleware import Middleware
from routes import ai_bp


def create_app() -> Flask:
    logging.basicConfig(level=os.getenv("LOG_LEVEL", "INFO"))
    app = Flask(__name__)
    CORS(app)
    config = ServiceConfig.from_env()
    middleware = Middleware(config)
    app.config["middleware"] = middleware
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
            return jsonify({"status": "not_ready", "error": str(exc)}), 503

    @app.get("/healthz")
    def health() -> Any:
        return jsonify({"status": "up"}), 200

    if config.strict_startup:
        middleware.readiness()
        app.logger.info("Middleware startup checks passed")

    return app
