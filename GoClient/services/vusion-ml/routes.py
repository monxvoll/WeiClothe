from functools import wraps
from typing import Any, Callable

from flask import Blueprint, current_app, g, jsonify, request
from jwt.exceptions import InvalidTokenError

ai_bp = Blueprint('ai_routes',__name__)

def require_bearer_token(handler: Callable[..., Any]) -> Callable[..., Any]:
    @wraps(handler)
    def wrapper(*args: Any, **kwargs: Any) -> Any:
        auth_header = request.headers.get("Authorization", "")
        if not auth_header.startswith("Bearer "):
            return jsonify({"error": "missing bearer token"}), 401

        token = auth_header.split(" ", 1)[1].strip()
        if not token:
            return jsonify({"error": "empty bearer token"}), 401

        middleware = current_app.config["middleware"]
        try:
            claims = middleware.keycloak.validate_access_token(token)
        except InvalidTokenError:
            return jsonify({"error": "invalid or expired token"}), 401
        except Exception:
            return jsonify({"error": "auth service unavailable"}), 503

        g.token_claims = claims
        return handler(*args, **kwargs)

    return wrapper

@ai_bp.route('/analizar', methods=['POST'])
@require_bearer_token
def analizar():
    data = request.get_json(silent=True) or {} #Deserealizate
    image_name = data.get('image', '').lower() #Extraction
    if not image_name:
        return jsonify({"error": "field 'image' is required"}), 400

    print(f"Recibiendo: {image_name}")

    category = "Desconocido"
    estilo = "Casual"

    #Simulation

    if "camisa" in image_name:
        category = "Camiseta"
    elif "pantalon" in image_name :
        category = "Pantalón"
    elif "zapato" in image_name:
        category = "Calzado"
    elif "vestido" in image_name:
        category = "Vestido"
        estilo = "Elegante"
    
    #Dictionary
    resultado = {
        "status": "success",
        "analysis": {
            "category": category,
            "style": estilo,
            "confidence": 0.95
        }
    }

    middleware = current_app.config["middleware"]
    try:
        middleware.kafka.publish_analysis_request(
            {
                "image": image_name,
                "analysis": resultado["analysis"],
            },
            g.token_claims.get("sub"),
        )
    except Exception:
        return jsonify({"error": "event bus unavailable"}), 503

    return jsonify(resultado)