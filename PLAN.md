# WeiCloth — Vision Pipeline Plan

---

## 1. Model Investigation

### Detection + Segmentation Candidates

| Model | Size | mAP (seg) | CPU Inf. | Notes |
|---|---|---|---|---|
| **YOLO26n-seg** | 6 MB | 39.6% | ~200ms | Ultra-light, dev/edge |
| **YOLO26s-seg** | 22 MB | 47.4% | ~300ms | Dev default |
| **YOLO26l-seg** | 61 MB | 54.4% | ~800ms | Prod balanced |
| **YOLO26x-seg** | 136 MB | 56.4% | ~1.5s | Prod max accuracy |
| Grounded-SAM2 | ~300 MB | High (zero-shot) | ~1.5s/GPU | No training needed, slower |
| YOLOv9 (improved) | 50-100 MB | ~53% | ~600ms | Research-grade clothing boost |

> YOLO26 released Jan 2026 by Ultralytics. NMS-free end-to-end inference, 43% faster CPU than prior series.

### Metadata Extraction Candidates

| Model | Task | Notes |
|---|---|---|
| **CLIP ViT-B/32** | Category, color, pattern (zero-shot) | Dev — fast, no training |
| **CLIP ViT-L/14** | Same, higher accuracy | Prod |
| ResNet50 + CBAM | Multi-attribute (50 categories, 1000 attrs) | Fine-tune on DeepFashion — +1.72% accuracy vs base |

---

## 2. Model Decision

### Segmentation: YOLO26-seg (Ultralytics)

- **Dev**: `yolo26s-seg` (22 MB) — CPU-friendly, fast iteration
- **Prod**: `yolo26l-seg` (61 MB) on GPU — 54.4% mAP, real-time capable
- Native instance segmentation → mask → transparent PNG in one pass
- Fine-tunable on **ModaNet** or **DeepFashion2** datasets if accuracy insufficient
- Ultralytics ecosystem: ONNX/TensorRT export for deployment

### Metadata Extraction: CLIP (OpenAI)

- Zero-shot: no labeled training data needed at launch
- Attributes extracted via text prompt classification:
  - Category: shirt / pants / dress / jacket / shoes / …
  - Color palette: dominant color + secondary
  - Pattern: solid / striped / checkered / floral / …
  - Style: casual / formal / sportswear / …
- Upgrade path: fine-tune on DeepFashion if zero-shot recall < 70%

### Background Removal: OpenCV + YOLO mask

- YOLO26-seg produces polygon mask per detected garment
- Apply mask with alpha channel → RGBA PNG output
- No extra model needed

---

## 3. Pipeline (implemented wiring)

Central schema: [`GoClient/db/init.sql`](GoClient/db/init.sql) — `users`, `clothes`, `styles`, `clothe_detections`.

```text
[Client] → POST garment (Go API) → INSERT clothes (status=queued, image_url=original)
              │
              ├──► Kafka topic `vusion.analysis.request`
              │         payload JSON: { garment_id, image_url, user_id }
              │
              ▼
[vusion-ml Python] Consumer → SELECT/DOWNLOAD original (HTTP/S3)
              │
              ├──► YOLO seg + OpenCV alpha + Pillow compress (≥80% resolution / side)
              ├──► CLIP ViT-B/32 zero-shot category (configurable)
              ├──► PUT processed PNG → S3/MinIO → public/presigned URL
              └──► UPDATE clothes (image_url, metadata, status=completed)
                    INSERT clothe_detections (all YOLO boxes; normalized bbox)
```

- **Go API** (`RegisterClothe`): after Postgres insert, publishes analysis payload (`internal/core/services/clothe_service.go`).
- **Python** persists **`clothes.image_url`** as the **processed** garment PNG URL used by the UI.
- **Dev**: Docker Compose adds **MinIO**, PostgreSQL 18, Redpanda; `S3_ENDPOINT_URL` points at MinIO.
- **Prod**: **Amazon Aurora PostgreSQL** (same SQL); **Amazon S3** (empty `S3_ENDPOINT_URL`, optional **IAM** on EC2 instead of access keys); Go API and **vusion-ml** on **separate EC2** instances, same DB endpoint + broker (**MSK** / self-hosted Kafka).

---

## 4. Implementation Plan

### Dev Mode (Docker Compose under `GoClient/`)

| Config | Value |
|---|---|
| Model | `YOLO_MODEL` (default `yolov8n-seg.pt`, CPU) |
| Storage | MinIO (`minio` service); keys `garments/{user_id}/{id}/processed.png` |
| DB | PostgreSQL 18 (`postgres` service); init `db/init.sql` |
| Events | Redpanda Kafka; topic `KAFKA_TOPIC_ANALYSIS` (default `vusion.analysis.request`) |
| Services | `api` (Go), `vusion-ml` (Flask/Gunicorn consumer + HTTP) |
| GPU | Optional |

**Stack (dev):**
```text
Go API → Kafka → vusion-ml (YOLO + CLIP) → MinIO + Postgres
```

### Prod Mode

| Config | Value |
|---|---|
| Model | Larger seg weights + GPU (`YOLO_MODEL`, e.g. `yolo26l-seg.pt` when available) |
| Storage | Amazon S3 (+ CloudFront optional); boto3 with VPC endpoint optional |
| DB | **Amazon Aurora PostgreSQL** (`DB_SSLMODE=require` typical) |
| Events | **MSK** or Kafka reachable from both EC2 instances |
| Topology | **Go API EC2** + **Python ML EC2**; shared Aurora cluster + S3 bucket |

**Stack (prod):**
```text
Go API (EC2) → Kafka → vusion-ml (EC2) → S3 + Aurora
```

### Performance Targets

| Metric | Dev | Prod |
|---|---|---|
| p50 latency | < 2s | < 500ms |
| p95 latency | < 5s | < 1.5s |
| Segmentation mAP | 47.4% | 54.4% |
| Metadata accuracy | ~70% zero-shot | ~80%+ fine-tuned |
| Image compression | ≥80% original res | ≥80% original res |

---

## 5. Efficiency Strategies (≥70% acceptance)

| Strategy | Acceptance | Description |
|---|---|---|
| **Async queue** | 95% | Upload → ACK immediately → process async → notify done. UX unblocked. |
| **Pre-validation gate** | 90% | Reject early: file size > 10MB, wrong MIME, image has no detectable garment. Saves full pipeline cost. |
| **Model quantization INT8** | 85% | YOLO TensorRT INT8 export: ~50% memory reduction, <5% mAP drop. |
| **Result deduplication** | 80% | SHA256 hash input image → skip full pipeline if already processed. |
| **Tiered model routing** | 78% | Small images (<500px): yolo26n-seg. Large images: yolo26l-seg. Save GPU cycles. |
| **Progressive metadata** | 75% | Return fast preview (category + dominant color) in ~100ms, full attributes async. |
| **Batch inference** | 72% | Group user uploads within 200ms window → single model forward pass. |

---

## 6. Testing Strategy

### Unit Tests

- `mask_generator`: assert output is RGBA PNG, alpha channel is binary mask
- `metadata_extractor`: assert dict keys present, confidence in [0,1]
- `compressor`: assert output size ≤ input, resolution ≥ 80% original
- `s3_uploader`: mock boto3, assert correct bucket/key format

### Integration Tests

- Full pipeline: sample image → DB row created + S3 object exists
- Pre-validation: reject oversized files, non-clothing images
- Deduplication: same image twice → same garment_id, no double processing

### Model Accuracy Tests (offline)

- Benchmark YOLO26s-seg on ModaNet test split → assert mAP ≥ 45%
- Benchmark CLIP zero-shot on DeepFashion Category split → assert top-1 ≥ 65%

### Performance Tests

- Locust load test: 50 concurrent uploads → assert p95 < 5s (dev), < 1.5s (prod)
- Memory profiling: confirm model stays within VRAM budget

### Edge Cases

- Dark / low-contrast images
- Multiple garments in one photo (pick highest confidence)
- Partial garment (shirt cropped at edge)
- Garment on mannequin vs person
- Accessories (bag, shoes) — verify category classification

---

## 7. Next Steps

1. Run stack: `docker compose` from `GoClient/` (`.env` with DB + Keycloak secrets).
2. E2E test: register garment with `image_url` pointing to upload in MinIO → verify Kafka → completed row + `clothe_detections`.
3. Tune `YOLO_MODEL` / CLIP / compression; add integration tests (mock S3/DB).
4. Security review (IAM least privilege, Kafka ACLs, secrets).
5. Prod hardening: Aurora failover tests, S3 lifecycle, DLQ for failed messages (optional).
