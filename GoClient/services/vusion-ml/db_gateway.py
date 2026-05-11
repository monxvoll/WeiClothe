"""Postgres access aligned with GoClient/db/init.sql (clothes, clothe_detections)."""

from __future__ import annotations

from contextlib import contextmanager

from psycopg2.pool import ThreadedConnectionPool

from config import ServiceConfig


class DbGateway:
    def __init__(self, config: ServiceConfig) -> None:
        self._config = config
        self._pool = ThreadedConnectionPool(
            minconn=1,
            maxconn=8,
            host=config.db_host,
            port=config.db_port,
            user=config.db_user,
            password=config.db_password,
            dbname=config.db_name,
            sslmode=config.db_sslmode,
        )

    def close(self) -> None:
        self._pool.closeall()

    def ping(self) -> None:
        with self._conn() as conn:
            with conn.cursor() as cur:
                cur.execute("SELECT 1")

    @contextmanager
    def _conn(self):
        conn = self._pool.getconn()
        try:
            yield conn
            conn.commit()
        except Exception:
            conn.rollback()
            raise
        finally:
            self._pool.putconn(conn)

    def update_status(self, garment_id: int, status: str, processing_error: str | None = None) -> None:
        with self._conn() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    """
                    UPDATE clothes
                    SET status = %s,
                        processing_error = %s,
                        updated_at = CURRENT_TIMESTAMP
                    WHERE id = %s
                    """,
                    (status, processing_error, garment_id),
                )

    def save_classification_complete(
        self,
        garment_id: int,
        *,
        image_url: str,
        image_width: int | None,
        image_height: int | None,
        garment_type: str,
        name: str | None,
        classification_id: str | None,
        category: str | None,
        subcategory: str | None,
        color: str | None,
        pattern: str | None,
        material: str | None,
        season: str | None,
        occasion: str | None,
        confidence: float | None,
        source: str,
        model_name: str | None,
        model_version: str | None,
        status: str = "completed",
    ) -> None:
        with self._conn() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    """
                    UPDATE clothes
                    SET
                        image_url = %s,
                        image_width = %s,
                        image_height = %s,
                        garment_type = %s,
                        name = %s,
                        classification_id = %s,
                        category = %s,
                        subcategory = %s,
                        color = %s,
                        pattern = %s,
                        material = %s,
                        season = %s,
                        occasion = %s,
                        confidence = %s,
                        source = %s,
                        model_name = %s,
                        model_version = %s,
                        status = %s,
                        processing_error = NULL,
                        processed_at = CURRENT_TIMESTAMP,
                        updated_at = CURRENT_TIMESTAMP
                    WHERE id = %s
                    """,
                    (
                        image_url,
                        image_width,
                        image_height,
                        garment_type,
                        name,
                        classification_id,
                        category,
                        subcategory,
                        color,
                        pattern,
                        material,
                        season,
                        occasion,
                        confidence,
                        source,
                        model_name,
                        model_version,
                        status,
                        garment_id,
                    ),
                )
                if cur.rowcount == 0:
                    raise ValueError(f"clothe id {garment_id} not found")

    def insert_clothe_detection(
        self,
        clothe_id: int,
        *,
        class_name: str,
        confidence: float,
        bbox_x: float,
        bbox_y: float,
        bbox_w: float,
        bbox_h: float,
        mask_url: str | None,
    ) -> None:
        with self._conn() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    """
                    INSERT INTO clothe_detections (
                        clothe_id, class_name, confidence,
                        bbox_x, bbox_y, bbox_w, bbox_h, mask_url
                    )
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
                    """,
                    (
                        clothe_id,
                        class_name,
                        confidence,
                        bbox_x,
                        bbox_y,
                        bbox_w,
                        bbox_h,
                        mask_url,
                    ),
                )
