"""S3-compatible storage (AWS S3 production, MinIO development)."""

from __future__ import annotations

import boto3
from botocore.config import Config

from config import ServiceConfig

# Hard cap: reject any single download that exceeds this to prevent OOM from HTTP responses.
_HTTP_MAX_RESPONSE_BYTES = 50 * 1024 * 1024  # 50 MB ceiling for raw object reads


class S3Gateway:
    def __init__(self, config: ServiceConfig) -> None:
        client_kw: dict = {
            "region_name": config.aws_region,
            "config": Config(signature_version="s3v4"),
        }
        if config.s3_endpoint_url:
            client_kw["endpoint_url"] = config.s3_endpoint_url
        if config.aws_access_key_id and config.aws_secret_access_key:
            client_kw["aws_access_key_id"] = config.aws_access_key_id
            client_kw["aws_secret_access_key"] = config.aws_secret_access_key
        self._client = boto3.client("s3", **client_kw)
        self.bucket = config.s3_bucket
        self._public_base = config.s3_public_base_url.rstrip("/") if config.s3_public_base_url else ""

    def head_bucket(self) -> None:
        self._client.head_bucket(Bucket=self.bucket)

    def get_object_bytes(self, key: str) -> bytes:
        """Download object body from this service bucket by key (staging / processed paths)."""
        key = key.strip().lstrip("/")
        resp = self._client.get_object(Bucket=self.bucket, Key=key)
        body = resp["Body"].read()
        if len(body) > _HTTP_MAX_RESPONSE_BYTES:
            raise ValueError(
                f"S3 object exceeded {_HTTP_MAX_RESPONSE_BYTES // (1024 * 1024)} MB limit"
            )
        return body

    def delete_object(self, key: str) -> None:
        key = key.strip().lstrip("/")
        self._client.delete_object(Bucket=self.bucket, Key=key)

    def upload_png(self, key: str, data: bytes, content_type: str = "image/png") -> str:
        self._client.put_object(
            Bucket=self.bucket,
            Key=key,
            Body=data,
            ContentType=content_type,
        )
        base = self._public_base
        if base:
            return f"{base}/{key}"
        return self._client.generate_presigned_url(
            "get_object",
            Params={"Bucket": self.bucket, "Key": key},
            ExpiresIn=86400,
        )
