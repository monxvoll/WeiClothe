from flask import Flask
from flask_cors import CORS
from routes import ai_bp  # Routes import from the other file

app = Flask(__name__)
CORS(app) 

#Blueprint register
app.register_blueprint(ai_bp)

if __name__ == '__main__':
    print("Corriendo en http://127.0.0.1:5000")
    app.run(port=5000, debug=True)