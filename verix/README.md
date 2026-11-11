# VERIX Knowledge Base — запуск локально
cd knowledge-base
python -m venv .venv
.venv\Scripts\activate
pip install -r requirements.txt

# Переменные окружения
cp .env.example .env
# Установите NEO4J_PASSWORD в .env

# Запуск шлюза
set NEO4J_URI=bolt://localhost:7687
set NEO4J_USER=neo4j
set NEO4J_PASSWORD=<пароль>
set KB_ADMIN_TOKEN=<токен>
uvicorn gateway.main:app --reload --port 8090
