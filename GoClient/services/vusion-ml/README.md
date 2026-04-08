# vision-ml

Servicio Flask de analisis de imagen con cliente Keycloak propio (`vision-ml`) y productor Kafka al broker compartido.

## Estructura

- `ai_service.py`: entrypoint.
- `app_factory.py`: factory Flask, endpoints `/healthz` y `/readyz`.
- `config.py`: carga y validacion de variables de entorno.
- `keycloak_client.py`: verificacion JWT via JWKS con audience `vision-ml`.
- `kafka_gateway.py`: productor Kafka hacia el broker compartido.
- `middleware.py`: composicion de clientes externos.
- `routes.py`: rutas HTTP protegidas y logica mock.

## Variables de entorno

```bash
# Requeridas
KAFKA_BROKERS=localhost:9093
KEYCLOAK_BASE_URL=http://localhost:9090
KEYCLOAK_REALM=weiclothe
KEYCLOAK_CLIENT_ID=vision-ml
KEYCLOAK_CLIENT_SECRET=V1s10nMlS3cr3tK3y2026xYz

# Opcionales
KAFKA_TOPIC_ANALYSIS=vusion.analysis.request
MIDDLEWARE_STRICT_STARTUP=true
MIDDLEWARE_HTTP_TIMEOUT_SECONDS=3.0
LOG_LEVEL=INFO
FLASK_DEBUG=false
```

## Endpoints

| Metodo | Ruta | Auth | Descripcion |
|--------|------|------|-------------|
| GET | `/healthz` | No | Liveness |
| GET | `/readyz` | No | Readiness (Keycloak + Kafka) |
| POST | `/analizar` | Bearer JWT | Analisis mock + evento Kafka |

## Ejecucion

```bash
cd GoClient/services/vusion-ml
pip install -r requirements.txt
cp .env-example .env   # ajustar valores si es necesario
python ai_service.py
```

En AWS, las variables se inyectan via ECS Task Definition, SSM Parameter Store o Secrets Manager. El `.env` solo es para desarrollo local.

Requiere Keycloak y Kafka corriendo (ver README raiz).
