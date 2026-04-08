import os
from app_factory import create_app


app = create_app()

if __name__ == "__main__":
    print("Corriendo en http://127.0.0.1:5000")
    app.run(port=5000, debug=os.getenv("FLASK_DEBUG", "false").lower() == "true")