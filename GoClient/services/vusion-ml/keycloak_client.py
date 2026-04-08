from typing import Any

import requests
from jwt import PyJWKClient, decode as jwt_decode

from config import ServiceConfig


class KeycloakVerifier:
    """Validates JWTs using the dedicated ``vision-ml`` Keycloak client.

    The client lives in the same ``weiclothe`` realm as ``go-api``.
    An audience mapper on the ``vision-ml`` client ensures access tokens
    contain ``"aud": "vision-ml"`` so this service can enforce audience
    validation.
    """

    def __init__(self, config: ServiceConfig) -> None:
        issuer = f"{config.keycloak_base_url}/realms/{config.keycloak_realm}"
        self._discovery_url = f"{issuer}/.well-known/openid-configuration"
        self._timeout = config.request_timeout_seconds
        self.client_id = config.keycloak_client_id
        self.client_secret = config.keycloak_client_secret
        discovery = requests.get(self._discovery_url, timeout=self._timeout)
        discovery.raise_for_status()
        openid = discovery.json()
        self.issuer = openid["issuer"]
        self._token_endpoint = openid["token_endpoint"]
        self._jwk_client = PyJWKClient(openid["jwks_uri"])

    def healthcheck(self) -> None:
        response = requests.get(self._discovery_url, timeout=self._timeout)
        response.raise_for_status()

    def validate_access_token(self, token: str) -> dict[str, Any]:
        signing_key = self._jwk_client.get_signing_key_from_jwt(token).key
        return jwt_decode(
            token,
            signing_key,
            algorithms=["RS256"],
            audience=self.client_id,
            issuer=self.issuer,
            options={"require": ["exp", "iat", "iss", "sub"]},
        )
