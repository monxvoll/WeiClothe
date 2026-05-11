# vision-ml

Flask service with Keycloak JWT (`vision-ml`), Kafka producer (legacy `/analizar`), and **Kafka consumer** for garment analysis: download image → YOLO segmentation + CLIP → **S3/MinIO** upload → **PostgreSQL/Aurora** (`clothes`, `clothe_detections` per `GoClient/db/init.sql`).

## Structure

- `ai_service.py`: entrypoint (Gunicorn loads `ai_service:app`).
- `app_factory.py`: Flask factory; `/healthz`, `/readyz`; starts **analysis consumer thread** when `ENABLE_ANALYSIS_CONSUMER=true`.
- `config.py`: env validation (Kafka, Keycloak, DB, S3).
- `kafka_consumer.py`: consumes `KAFKA_TOPIC_ANALYSIS` (JSON `{ garment_id, staging_key, user_id, attempt? }`); after 3 failures publishes to `KAFKA_TOPIC_ANALYSIS_DLQ`.
- `ml_pipeline.py`: Ultralytics YOLO-seg + CLIP + PNG compression (≥80% resolution).
- `db_gateway.py`: psycopg2 pool → UPDATE `clothes`, INSERT `clothe_detections`.
- `s3_gateway.py`: boto3 (AWS S3 or MinIO via `S3_ENDPOINT_URL`).
- `kafka_gateway.py`: Kafka producer (existing HTTP routes).
- `middleware.py`: readiness Keycloak + Kafka + optional Postgres + S3.
- `routes.py`: protected routes (mock/demo `/analizar`).

## Object storage lifecycle (raw staging)

- Raw uploads use keys `raw/{user_id}/{garment_id}/original.*`. The worker deletes the staging object after a successful DB commit.
- **Safety net:** configure bucket lifecycle to expire `raw/` after 24–48h (crashed workers). Examples:
  - **AWS S3:** Lifecycle rule — prefix `raw/`, expiration 1 day.
  - **MinIO:** `mc ilm add ... --prefix "raw/" --expire-days 1` (see MinIO ILM docs).

## Environment

See `.env-example` for full list. Minimum when consumer enabled:

- Kafka + Keycloak (as before).
- `DB_*`, `S3_BUCKET`; optional `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` (omit on EC2 with IAM).
- `S3_ENDPOINT_URL` empty for real AWS; `http://minio:9000` for Compose MinIO.
- `S3_PUBLIC_BASE_URL` for browser-visible URLs (path-style MinIO dev URL).

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/healthz` | No | Liveness |
| GET | `/readyz` | No | Keycloak + Kafka + Postgres + S3 (if consumer on) |
| POST | `/analizar` | Bearer JWT | Legacy mock + Kafka publish |

## Run locally

```bash
cd GoClient/services/vusion-ml
# Dockerfile installs CPU PyTorch first; then pip install -r requirements.txt
pip install torch torchvision --index-url https://download.pytorch.org/whl/cpu
pip install -r requirements.txt
cp .env-example .env   # edit DB, MinIO/S3
python ai_service.py
```

Production: inject env via ECS task definition, SSM, or Secrets Manager; Go API sends multipart `POST /clothes`, stages raw objects under `raw/` on S3, then publishes `{ garment_id, staging_key, user_id }` to `KAFKA_TOPIC_ANALYSIS`.
