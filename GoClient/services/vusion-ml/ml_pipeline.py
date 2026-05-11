"""YOLO instance segmentation + CLIP metadata + PNG compression (>=80% resolution)."""

from __future__ import annotations

import io
from dataclasses import dataclass

import numpy as np

from config import ServiceConfig

_yolo_model = None
_clip_processor = None
_clip_model = None
_torch = None


@dataclass
class DetectionRecord:
    class_name: str
    confidence: float
    bbox_x: float
    bbox_y: float
    bbox_w: float
    bbox_h: float
    mask_url: str | None


@dataclass
class PipelineResult:
    processed_png_bytes: bytes
    image_width: int
    image_height: int
    garment_type: str
    category: str | None
    subcategory: str | None
    color: str | None
    pattern: str | None
    confidence: float
    model_name: str
    model_version: str
    detections: list[DetectionRecord]
    classification_id: str | None = None


def _get_yolo(model_path: str):
    global _yolo_model
    if _yolo_model is None:
        from ultralytics import YOLO

        _yolo_model = YOLO(model_path)
    return _yolo_model


def _get_clip(config: ServiceConfig):
    global _clip_processor, _clip_model, _torch
    if _clip_model is None:
        import torch
        from transformers import CLIPModel, CLIPProcessor

        _torch = torch
        _clip_processor = CLIPProcessor.from_pretrained(config.clip_model)
        _clip_model = CLIPModel.from_pretrained(config.clip_model)
        _clip_model.eval()
    return _clip_processor, _clip_model, _torch


def _clip_category(rgb_crop: np.ndarray, config: ServiceConfig) -> tuple[str, float]:
    processor, model, torch = _get_clip(config)
    from PIL import Image

    if rgb_crop.size == 0:
        return "unknown", 0.0
    pil = Image.fromarray(rgb_crop)
    labels = [
        "a photo of a shirt",
        "a photo of pants or trousers",
        "a photo of a dress",
        "a photo of shoes",
        "a photo of a jacket or coat",
        "a photo of a bag",
        "a photo of a hat",
        "a photo of clothing",
    ]
    inputs = processor(text=labels, images=pil, return_tensors="pt", padding=True)
    with torch.no_grad():
        outputs = model(**inputs)
        logits = outputs.logits_per_image
        probs = logits.softmax(dim=1)[0]
        best = int(probs.argmax())
        conf = float(probs[best])
    mapping = ["shirt", "pants", "dress", "shoes", "jacket", "bag", "hat", "other"]
    return mapping[best], conf


def _compress_png_min_resolution(img, min_ratio: float = 0.8) -> tuple[bytes, int, int]:
    from PIL import Image

    w, h = img.size
    new_w = max(1, int(w * min_ratio))
    new_h = max(1, int(h * min_ratio))
    if new_w >= w and new_h >= h:
        buf = io.BytesIO()
        img.save(buf, format="PNG", optimize=True)
        raw = buf.getvalue()
    else:
        resized = img.resize((new_w, new_h), Image.LANCZOS)
        buf = io.BytesIO()
        resized.save(buf, format="PNG", optimize=True)
        raw = buf.getvalue()
    out = Image.open(io.BytesIO(raw))
    return raw, out.width, out.height


def run_pipeline(image_bytes: bytes, config: ServiceConfig) -> PipelineResult:
    import cv2
    from PIL import Image

    if len(image_bytes) > config.max_image_size_bytes:
        raise ValueError(
            f"image exceeds maximum allowed size of "
            f"{config.max_image_size_bytes // (1024 * 1024)} MB "
            f"(received {len(image_bytes) // (1024 * 1024)} MB)"
        )

    buf = np.frombuffer(image_bytes, dtype=np.uint8)
    im_bgr = cv2.imdecode(buf, cv2.IMREAD_COLOR)
    if im_bgr is None:
        raise ValueError("could not decode image bytes")
    h0, w0 = im_bgr.shape[:2]

    model = _get_yolo(config.yolo_model)
    results = model(im_bgr, verbose=False)
    r = results[0]

    if r.boxes is None or len(r.boxes) == 0:
        raise ValueError("YOLO found no detections")

    boxes = r.boxes
    best_i = int(boxes.conf.argmax())
    names_map = r.names if hasattr(r, "names") and r.names else getattr(model, "names", {})

    detections: list[DetectionRecord] = []
    for i in range(len(boxes)):
        xyxy = boxes.xyxy[i].cpu().numpy()
        x1, y1, x2, y2 = (float(xyxy[0]), float(xyxy[1]), float(xyxy[2]), float(xyxy[3]))
        conf_f = float(boxes.conf[i].cpu().numpy())
        cls_id = int(boxes.cls[i].cpu().numpy())
        if isinstance(names_map, dict):
            cname = str(names_map.get(cls_id, str(cls_id)))
        else:
            cname = str(names_map[cls_id])
        bw, bh = x2 - x1, y2 - y1
        detections.append(
            DetectionRecord(
                class_name=cname,
                confidence=conf_f,
                bbox_x=x1 / w0,
                bbox_y=y1 / h0,
                bbox_w=bw / w0,
                bbox_h=bh / h0,
                mask_url=None,
            )
        )

    im_rgb = cv2.cvtColor(im_bgr, cv2.COLOR_BGR2RGB)
    alpha = np.zeros((h0, w0), dtype=np.uint8)
    if getattr(r, "masks", None) is not None and r.masks is not None:
        md = r.masks.data
        if md is not None and len(md) > best_i:
            m = md[best_i].cpu().numpy()
            if m.ndim == 2 and m.shape != (h0, w0):
                m = cv2.resize(m, (w0, h0), interpolation=cv2.INTER_NEAREST)
            if m.max() <= 1.0:
                alpha = (m * 255.0).clip(0, 255).astype(np.uint8)
            else:
                alpha = m.astype(np.uint8)
    if not alpha.any():
        xyxy = boxes.xyxy[best_i].cpu().numpy()
        x1, y1, x2, y2 = [int(round(v)) for v in xyxy]
        x1, y1 = max(0, x1), max(0, y1)
        x2, y2 = min(w0, x2), min(h0, y2)
        alpha[y1:y2, x1:x2] = 255

    rgba = np.dstack([im_rgb, alpha])
    pil_rgba = Image.fromarray(rgba, mode="RGBA")
    png_bytes, fw, fh = _compress_png_min_resolution(pil_rgba, 0.8)

    xyxy_b = boxes.xyxy[best_i].cpu().numpy()
    bx1, by1, bx2, by2 = [int(round(v)) for v in xyxy_b]
    bx1, by1 = max(0, bx1), max(0, by1)
    bx2, by2 = min(w0, bx2), min(h0, by2)
    crop = im_rgb[by1:by2, bx1:bx2]
    cat, cconf = _clip_category(crop if crop.size else im_rgb, config)

    cls_best = int(boxes.cls[best_i].cpu().numpy())
    if isinstance(names_map, dict):
        best_name = str(names_map.get(cls_best, "unknown"))
    else:
        best_name = str(names_map[cls_best])

    return PipelineResult(
        processed_png_bytes=png_bytes,
        image_width=fw,
        image_height=fh,
        garment_type=best_name,
        category=cat,
        subcategory=None,
        color=None,
        pattern="solid",
        confidence=cconf,
        model_name=config.yolo_model,
        model_version="ultralytics",
        detections=detections,
        classification_id=None,
    )
