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

# --- CLIP zero-shot label sets (text prompts → canonical values) ---

CATEGORY_LABELS = [
    "a photo of a shirt",
    "a photo of pants or trousers",
    "a photo of a dress",
    "a photo of shoes",
    "a photo of a jacket or coat",
    "a photo of a bag",
    "a photo of a hat",
    "a photo of clothing",
]
CATEGORY_MAPPING = ["shirt", "pants", "dress", "shoes", "jacket", "bag", "hat", "other"]

COLOR_LABELS = [
    "a photo of a red garment",
    "a photo of a blue garment",
    "a photo of a black garment",
    "a photo of a white garment",
    "a photo of a green garment",
    "a photo of a yellow garment",
    "a photo of a pink garment",
    "a photo of an orange garment",
    "a photo of a purple garment",
    "a photo of a brown garment",
    "a photo of a gray garment",
    "a photo of a beige or cream garment",
    "a photo of a navy garment",
    "a photo of a multicolor garment",
]
COLOR_MAPPING = [
    "red",
    "blue",
    "black",
    "white",
    "green",
    "yellow",
    "pink",
    "orange",
    "purple",
    "brown",
    "gray",
    "beige",
    "navy",
    "multicolor",
]

MATERIAL_LABELS = [
    "a garment made of cotton fabric",
    "a garment made of denim or jeans fabric",
    "a garment made of leather",
    "a garment made of silk or satin fabric",
    "a garment made of wool or knit fabric",
    "a garment made of polyester or synthetic fabric",
    "a garment made of linen fabric",
    "a garment made of suede or velvet fabric",
]
MATERIAL_MAPPING = ["cotton", "denim", "leather", "silk", "wool", "polyester", "linen", "suede"]

PATTERN_LABELS = [
    "a garment with a solid plain color",
    "a garment with stripes",
    "a garment with a checkered or plaid pattern",
    "a garment with a floral pattern",
    "a garment with polka dots",
    "a garment with a camouflage pattern",
    "a garment with a graphic or print design",
    "a garment with an animal print pattern",
]
PATTERN_MAPPING = [
    "solid",
    "striped",
    "checkered",
    "floral",
    "polka_dots",
    "camouflage",
    "graphic",
    "animal_print",
]

SEASON_LABELS = [
    "a garment suitable for summer or warm weather",
    "a garment suitable for winter or cold weather",
    "a garment suitable for spring weather",
    "a garment suitable for autumn or fall weather",
    "a garment suitable for all seasons",
]
SEASON_MAPPING = ["summer", "winter", "spring", "fall", "all_season"]

OCCASION_LABELS = [
    "a garment for casual everyday wear",
    "a garment for formal or business wear",
    "a garment for sportswear or athletic use",
    "a garment for party or evening wear",
    "a garment for outdoor or adventure wear",
    "a garment for beachwear",
]
OCCASION_MAPPING = ["casual", "formal", "sport", "party", "outdoor", "beach"]

# category → (text prompts, canonical subcategory slugs)
SUBCATEGORY_MAP: dict[str, tuple[list[str], list[str]]] = {
    "shirt": (
        [
            "a photo of a t-shirt",
            "a photo of a polo shirt",
            "a photo of a dress shirt",
            "a photo of a blouse",
            "a photo of a tank top",
            "a photo of a hoodie",
        ],
        ["t_shirt", "polo", "dress_shirt", "blouse", "tank_top", "hoodie"],
    ),
    "pants": (
        [
            "a photo of jeans",
            "a photo of chino pants",
            "a photo of jogger pants",
            "a photo of shorts",
            "a photo of dress pants",
            "a photo of cargo pants",
        ],
        ["jeans", "chinos", "joggers", "shorts", "dress_pants", "cargo_pants"],
    ),
    "dress": (
        [
            "a photo of a maxi dress",
            "a photo of a mini dress",
            "a photo of a midi dress",
            "a photo of a sundress",
            "a photo of a cocktail dress",
        ],
        ["maxi_dress", "mini_dress", "midi_dress", "sundress", "cocktail_dress"],
    ),
    "jacket": (
        [
            "a photo of a blazer",
            "a photo of a bomber jacket",
            "a photo of a denim jacket",
            "a photo of a parka",
            "a photo of a cardigan",
            "a photo of a vest",
        ],
        ["blazer", "bomber", "denim_jacket", "parka", "cardigan", "vest"],
    ),
    "shoes": (
        [
            "a photo of sneakers",
            "a photo of boots",
            "a photo of sandals",
            "a photo of high heels",
            "a photo of loafers",
            "a photo of flat shoes",
        ],
        ["sneakers", "boots", "sandals", "heels", "loafers", "flats"],
    ),
    "bag": (
        [
            "a photo of a handbag",
            "a photo of a backpack",
            "a photo of a tote bag",
            "a photo of a crossbody bag",
        ],
        ["handbag", "backpack", "tote", "crossbody"],
    ),
    "hat": (
        [
            "a photo of a baseball cap",
            "a photo of a beanie",
            "a photo of a sun hat",
            "a photo of a fedora",
        ],
        ["baseball_cap", "beanie", "sun_hat", "fedora"],
    ),
    "other": (
        [
            "a photo of a scarf",
            "a photo of a belt",
            "a photo of gloves",
            "a photo of general clothing",
        ],
        ["scarf", "belt", "gloves", "general"],
    ),
}


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
    material: str | None
    season: str | None
    occasion: str | None
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


def _clip_classify(
    rgb_crop: np.ndarray,
    labels: list[str],
    mapping: list[str],
    config: ServiceConfig,
) -> tuple[str, float]:
    """Generic CLIP zero-shot: one forward pass per attribute group."""
    if rgb_crop.size == 0:
        return mapping[0] if mapping else "unknown", 0.0
    if len(labels) != len(mapping):
        raise ValueError("labels and mapping must have the same length")

    processor, model, torch = _get_clip(config)
    from PIL import Image

    pil = Image.fromarray(rgb_crop)
    inputs = processor(text=labels, images=pil, return_tensors="pt", padding=True)
    with torch.no_grad():
        outputs = model(**inputs)
        logits = outputs.logits_per_image
        probs = logits.softmax(dim=1)[0]
        best = int(probs.argmax())
    return mapping[best], float(probs[best])


def _clip_category(rgb_crop: np.ndarray, config: ServiceConfig) -> tuple[str, float]:
    return _clip_classify(rgb_crop, CATEGORY_LABELS, CATEGORY_MAPPING, config)


def _clip_color(rgb_crop: np.ndarray, config: ServiceConfig) -> tuple[str, float]:
    return _clip_classify(rgb_crop, COLOR_LABELS, COLOR_MAPPING, config)


def _clip_material(rgb_crop: np.ndarray, config: ServiceConfig) -> tuple[str, float]:
    return _clip_classify(rgb_crop, MATERIAL_LABELS, MATERIAL_MAPPING, config)


def _clip_pattern(rgb_crop: np.ndarray, config: ServiceConfig) -> tuple[str, float]:
    return _clip_classify(rgb_crop, PATTERN_LABELS, PATTERN_MAPPING, config)


def _clip_season(rgb_crop: np.ndarray, config: ServiceConfig) -> tuple[str, float]:
    return _clip_classify(rgb_crop, SEASON_LABELS, SEASON_MAPPING, config)


def _clip_occasion(rgb_crop: np.ndarray, config: ServiceConfig) -> tuple[str, float]:
    return _clip_classify(rgb_crop, OCCASION_LABELS, OCCASION_MAPPING, config)


def _clip_subcategory(
    rgb_crop: np.ndarray,
    category: str,
    config: ServiceConfig,
) -> tuple[str | None, float]:
    entry = SUBCATEGORY_MAP.get(category)
    if entry is None:
        return None, 0.0
    labels, mapping = entry
    return _clip_classify(rgb_crop, labels, mapping, config)


def _extract_metadata(rgb_crop: np.ndarray, config: ServiceConfig) -> dict[str, str | None]:
    """Run all CLIP attribute classifiers on the garment crop."""
    cat, cat_conf = _clip_category(rgb_crop, config)
    color, _ = _clip_color(rgb_crop, config)
    material, _ = _clip_material(rgb_crop, config)
    pattern, _ = _clip_pattern(rgb_crop, config)
    season, _ = _clip_season(rgb_crop, config)
    occasion, _ = _clip_occasion(rgb_crop, config)
    subcat, _ = _clip_subcategory(rgb_crop, cat, config)
    return {
        "category": cat,
        "category_confidence": cat_conf,
        "subcategory": subcat,
        "color": color,
        "material": material,
        "pattern": pattern,
        "season": season,
        "occasion": occasion,
    }


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


def _pipeline_result_full_frame(im_bgr, config: ServiceConfig) -> PipelineResult:
    """Fallback when YOLO finds no boxes: segment whole frame and run CLIP on it."""
    import cv2
    from PIL import Image

    h0, w0 = im_bgr.shape[:2]
    im_rgb = cv2.cvtColor(im_bgr, cv2.COLOR_BGR2RGB)
    alpha = np.full((h0, w0), 255, dtype=np.uint8)
    rgba = np.dstack([im_rgb, alpha])
    pil_rgba = Image.fromarray(rgba, mode="RGBA")
    png_bytes, fw, fh = _compress_png_min_resolution(pil_rgba, 0.8)
    meta = _extract_metadata(im_rgb, config)
    category = meta["category"] or "unknown"
    return PipelineResult(
        processed_png_bytes=png_bytes,
        image_width=fw,
        image_height=fh,
        garment_type=category,
        category=category,
        subcategory=meta["subcategory"],
        color=meta["color"],
        pattern=meta["pattern"],
        material=meta["material"],
        season=meta["season"],
        occasion=meta["occasion"],
        confidence=meta["category_confidence"],
        model_name=config.yolo_model,
        model_version="ultralytics",
        detections=[],
        classification_id=None,
    )


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
        return _pipeline_result_full_frame(im_bgr, config)

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
    meta = _extract_metadata(crop if crop.size else im_rgb, config)

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
        category=meta["category"],
        subcategory=meta["subcategory"],
        color=meta["color"],
        pattern=meta["pattern"],
        material=meta["material"],
        season=meta["season"],
        occasion=meta["occasion"],
        confidence=meta["category_confidence"],
        model_name=config.yolo_model,
        model_version="ultralytics",
        detections=detections,
        classification_id=None,
    )
