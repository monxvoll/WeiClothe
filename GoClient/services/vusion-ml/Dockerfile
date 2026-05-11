FROM python:3.13-slim-trixie

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PATH="/root/.local/bin:${PATH}"

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        libglib2.0-0 \
    && curl -LsSf https://astral.sh/uv/install.sh | sh \
    && rm -rf /var/lib/apt/lists/*

# Torch + torchvision CPU — own cached layer; survives requirements.txt changes.
RUN --mount=type=cache,target=/root/.cache/uv \
    uv pip install --system torch torchvision --index-url https://download.pytorch.org/whl/cpu

COPY requirements.txt .

# 1. Install headless OpenCV first.
# 2. Install remaining requirements (ultralytics may pull opencv-python GUI as a dep).
# 3. Force-reinstall headless to evict any GUI variant ultralytics injected.
RUN --mount=type=cache,target=/root/.cache/uv \
    uv pip install --system opencv-python-headless \
    && uv pip install --system -r requirements.txt \
    && uv pip uninstall --system opencv-python opencv-contrib-python \
    && uv pip install --system --reinstall opencv-python-headless

COPY . .

EXPOSE 5000

# Single worker keeps one shared Kafka producer process-safe.
CMD ["gunicorn", "--bind", "0.0.0.0:5000", "--workers", "1", "--threads", "4", "--timeout", "300", "--graceful-timeout", "30", "ai_service:app"]
