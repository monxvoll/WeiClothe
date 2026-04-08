# WeiCloth

## Infraestructura

```bash
cd GoClient
docker compose up -d
```

Levanta: Keycloak (`:9090`), Kafka/Redpanda (`:9093`), Postgres (`:5432`), Kafka UI (`:8080`).

Servicio individual:

```bash
docker compose up kafka -d
docker compose up keycloak -d
docker compose up postgres -d
```

Bajar todo:

```bash
docker compose down -v
```

## Go API

```bash
cd GoClient
cp .env-example .env   # completar valores
go run cmd/api/main.go
```

Tests:

```bash
go test ./... -v
go test ./... -tags=integration -v
```

## vision-ml (Python)

Requiere Keycloak y Kafka levantados. Tiene su propio cliente Keycloak (`vision-ml`) en el mismo realm `weiclothe`.

```bash
cd GoClient/services/vusion-ml
pip install -r requirements.txt
cp .env-example .env   # ajustar valores si es necesario
python ai_service.py
```

### Probar con Postman

**1. Obtener token (Keycloak)**

```
POST http://localhost:9090/realms/weiclothe/protocol/openid-connect/token
```

Body (`x-www-form-urlencoded`):

| Key | Value |
|-----|-------|
| `client_id` | `vision-ml` |
| `client_secret` | `V1s10nMlS3cr3tK3y2026xYz` |
| `grant_type` | `password` |
| `username` | tu email registrado |
| `password` | tu password |

Copiar el `access_token` de la respuesta.

**2. Health check**

```
GET http://localhost:5000/healthz
```

No requiere auth. Respuesta esperada: `{"status": "up"}`.

**3. Readiness check**

```
GET http://localhost:5000/readyz
```

No requiere auth. Respuesta esperada: `{"status": "ready", "middleware": {"keycloak": "up", "kafka": "up"}}`.

**4. Analizar imagen**

```
POST http://localhost:5000/analizar
```

Headers:

| Key | Value |
|-----|-------|
| `Authorization` | `Bearer <token_del_paso_1>` |
| `Content-Type` | `application/json` |

Body (`raw JSON`):

```json
{"image": "camisa_azul.jpg"}
```

Respuesta esperada:

```json
{
  "status": "success",
  "analysis": {
    "category": "Camiseta",
    "style": "Casual",
    "confidence": 0.95
  }
}
```

El evento queda publicado en Kafka (topic `vusion.analysis.request`), visible en Kafka UI: `http://localhost:8080`.

## Frontend

```bash
ng serve -o
```
