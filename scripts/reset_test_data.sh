#!/bin/bash

# Reset test data for all services

echo "🧹 Resetting test data..."

# Stop server
echo "Stopping server..."
pkill -9 -f aiwriter-server

# Clear PostgreSQL
echo "Clearing PostgreSQL data..."
docker exec -i ai-writer-postgres psql -U ai_writer_user -d ai_writer <<EOF
TRUNCATE TABLE users, agents, agent_groups, knowledge_bases, ai_provider_configs, documents, chunks CASCADE;
EOF

# Clear Redis
echo "Clearing Redis data..."
docker exec -i ai-writer-redis redis-cli FLUSHALL

# Clear MinIO
echo "Clearing MinIO data..."
docker exec -i ai-writer-minio mc rm --recursive --force minio/ai-writer/ || true
docker exec -i ai-writer-minio mc mb minio/ai-writer || true

# Clear Milvus (drop all collections)
echo "Clearing Milvus data..."
docker exec -i ai-writer-milvus-standalone bash -c "
export PYTHONPATH=/opt/milvus/lib:/opt/milvus/lib/python3.11/site-packages
python3 <<'PYTHON'
from pymilvus import connections, utility
try:
    connections.connect(host='localhost', port='19530')
    collections = utility.list_collections()
    for col in collections:
        utility.drop_collection(col)
        print(f'Dropped collection: {col}')
    print('All collections dropped')
except Exception as e:
    print(f'Error: {e}')
finally:
    connections.disconnect('default')
PYTHON
" 2>/dev/null || echo "Milvus clear skipped (optional)"

echo "✅ Test data reset complete!"
