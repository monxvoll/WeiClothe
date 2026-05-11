"""Kafka consumer for garment analysis jobs (topic from KAFKA_TOPIC_ANALYSIS)."""

from __future__ import annotations

import json
import logging
import threading
import traceback

from confluent_kafka import Consumer

from config import ServiceConfig
from db_gateway import DbGateway
from kafka_gateway import KafkaGateway
from ml_pipeline import run_pipeline
from s3_gateway import S3Gateway

LOG = logging.getLogger(__name__)


class AnalysisConsumerThread(threading.Thread):
    """Background worker: download image from staged S3 key, run ML pipeline, persist to Postgres + S3."""

    def __init__(
        self,
        config: ServiceConfig,
        db: DbGateway,
        s3: S3Gateway,
        kafka: KafkaGateway,
    ) -> None:
        super().__init__(daemon=True, name="vusion-analysis-consumer")
        self._config = config
        self._db = db
        self._s3 = s3
        self._kafka = kafka
        self._stop = threading.Event()
        self._consumer: Consumer | None = None

    def stop(self) -> None:
        self._stop.set()
        if self._consumer is not None:
            try:
                self._consumer.close()
            except Exception:  # noqa: BLE001
                LOG.exception("consumer close")

    def run(self) -> None:
        conf = {
            "bootstrap.servers": self._config.kafka_brokers,
            "group.id": self._config.kafka_group_id,
            "auto.offset.reset": "earliest",
            # Manual commit: offset is committed only after successful processing (at-least-once).
            "enable.auto.commit": False,
        }
        consumer = Consumer(conf)
        self._consumer = consumer
        consumer.subscribe([self._config.kafka_topic_analysis])
        LOG.info("analysis consumer subscribed to %s", self._config.kafka_topic_analysis)

        while not self._stop.is_set():
            msg = consumer.poll(1.0)
            if msg is None:
                continue
            if msg.error():
                LOG.warning("kafka poll error: %s", msg.error())
                continue
            try:
                payload = json.loads(msg.value().decode("utf-8"))
                # Not named "_handle": threading.Thread sets self._handle (ThreadHandle) in 3.13+,
                # which shadows a method with that name → TypeError: not callable.
                self._process_analysis_payload(payload)
                consumer.commit(msg)
            except Exception:  # noqa: BLE001
                LOG.exception("failed to process analysis message")

    def _process_analysis_payload(self, payload: dict) -> None:
        garment_id = int(payload["garment_id"])
        staging_key = str(payload.get("staging_key") or "").strip()
        user_id = str(payload["user_id"])
        attempt = int(payload.get("attempt", 0))

        if not staging_key:
            raise ValueError("staging_key is required in analysis payload")

        try:
            self._db.update_status(garment_id, "processing", None)
            raw = self._s3.get_object_bytes(staging_key)
            result = run_pipeline(raw, self._config)
            proc_key = f"garments/{user_id}/{garment_id}/processed.png"
            out_url = self._s3.upload_png(proc_key, result.processed_png_bytes)

            self._db.save_classification_complete(
                garment_id,
                image_url=out_url,
                image_width=result.image_width,
                image_height=result.image_height,
                garment_type=result.garment_type,
                name=result.category,
                classification_id=result.classification_id,
                category=result.category,
                subcategory=result.subcategory,
                color=result.color,
                pattern=result.pattern,
                material=None,
                season=None,
                occasion=None,
                confidence=result.confidence,
                source="ai",
                model_name=result.model_name,
                model_version=result.model_version,
                status="completed",
            )
            for det in result.detections:
                self._db.insert_clothe_detection(
                    garment_id,
                    class_name=det.class_name,
                    confidence=det.confidence,
                    bbox_x=det.bbox_x,
                    bbox_y=det.bbox_y,
                    bbox_w=det.bbox_w,
                    bbox_h=det.bbox_h,
                    mask_url=det.mask_url,
                )

            self._s3.delete_object(staging_key)
            LOG.info("garment %s analysis completed", garment_id)
        except Exception as exc:  # noqa: BLE001
            err_text = f"{exc}\n{traceback.format_exc()}"[:8000]
            LOG.exception("garment %s analysis failed (attempt %s)", garment_id, attempt)
            if attempt >= 2:
                dlq_payload = {
                    **payload,
                    "error": err_text[:4000],
                    "final_attempt": attempt,
                }
                self._kafka.publish_json(
                    self._config.kafka_topic_dlq,
                    str(garment_id),
                    dlq_payload,
                )
                try:
                    self._db.update_status(garment_id, "failed", err_text[:2000])
                except Exception:  # noqa: BLE001
                    LOG.exception("could not persist failure status for garment %s", garment_id)
                return

            retry_payload = {**payload, "attempt": attempt + 1}
            self._kafka.publish_json(
                self._config.kafka_topic_analysis,
                user_id,
                retry_payload,
            )
            try:
                self._db.update_status(garment_id, "queued", None)
            except Exception:  # noqa: BLE001
                LOG.exception("could not reset status for garment %s retry", garment_id)
