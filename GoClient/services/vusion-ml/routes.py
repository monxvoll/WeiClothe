from flask import Blueprint, request, jsonify

ai_bp = Blueprint('ai_routes',__name__)

@ai_bp.route('/analizar', methods=['POST'])
def analizar():
  
    data = request.get_json() #Deserealizate
    image_name = data.get('image', '').lower() #Extraction

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

    return jsonify(resultado) #Serialization

@ai_bp.route('/health', methods=['GET'])
def health():
    return jsonify({"status": "Funcionando"})