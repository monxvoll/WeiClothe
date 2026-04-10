import logging
import threading
import time
from typing import Any

import requests
from jwt import PyJWKClient, decode as jwt_decode

from config import ServiceConfig

logger = logging.getLogger(__name__)


class KeycloakVerifier:
    """Validates JWTs using the dedicated ``vision-ml`` Keycloak client.

    The client lives in the same ``weiclothe`` realm as ``go-api``.
    An audience mapper on the ``vision-ml`` client ensures access tokens
    contain ``"aud": "vision-ml"`` so this service can enforce audience
    validation.

    OpenID discovery is loaded lazily with retries so the container can start
    while Keycloak is still importing the realm (docker compose race).
    """

    def __init__(self, config: ServiceConfig) -> None:
        self._config = config
        issuer_base = config.keycloak_base_url.rstrip("/")
        self._discovery_url = f"{issuer_base}/realms/{config.keycloak_realm}/.well-known/openid-configuration"
        self._timeout = config.request_timeout_seconds
        self.client_id = config.keycloak_client_id
        self.client_secret = config.keycloak_client_secret
        self.issuer: str | None = None
        self._jwk_client: PyJWKClient | None = None
        self._bootstrap_lock = threading.Lock()

    def _bootstrap(self) -> None:
        with self._bootstrap_lock:
            if self._jwk_client is not None:
                return
            attempts = max(1, self._config.keycloak_bootstrap_attempts)
            delay = max(0.5, self._config.keycloak_bootstrap_delay_seconds)
            last_exc: BaseException | None = None
            for attempt in range(attempts):
                try:
                    discovery = requests.get(self._discovery_url, timeout=self._timeout)
                    discovery.raise_for_status()
                    openid = discovery.json()
                    self.issuer = openid["issuer"]
                    self._jwk_client = PyJWKClient(openid["jwks_uri"], cache_jwk_for=300)
                    logger.info("Keycloak OpenID discovery OK (%s)", self._discovery_url)
                    return
                except Exception as exc:
                    last_exc = exc
                    logger.warning(
                        "Keycloak discovery attempt %s/%s failed: %s",
                        attempt + 1,
                        attempts,
                        exc,
                    )
                    if attempt < attempts - 1:
                        time.sleep(delay)
            raise RuntimeError(
                f"Keycloak unreachable after {attempts} attempts: {self._discovery_url}"
            ) from last_exc

    def healthcheck(self) -> None:
        self._bootstrap()

    def validate_access_token(self, token: str) -> dict[str, Any]:
        self._bootstrap()
        if self._jwk_client is None or self.issuer is None:
            raise RuntimeError("Keycloak verifier not initialized")
        signing_key = self._jwk_client.get_signing_key_from_jwt(token).key
        return jwt_decode(
            token,
            signing_key,
            algorithms=["RS256"],
            audience=self.client_id,
            issuer=self.issuer,
            options={"require": ["exp", "iat", "iss", "sub"]},
        )
