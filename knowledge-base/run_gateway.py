import os
os.environ["NEO4J_URI"] = "bolt://127.0.0.1:7687"
os.environ["NEO4J_USER"] = "neo4j"
os.environ["NEO4J_PASSWORD"] = "VerixNeo4j2026"
os.environ["NEO4J_DATABASE"] = "neo4j"
os.environ["KB_ADMIN_TOKEN"] = "test-admin-token"

import uvicorn
uvicorn.run("gateway.main:app", host="127.0.0.1", port=8090)
