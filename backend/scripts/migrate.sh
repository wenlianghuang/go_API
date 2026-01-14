#!/bin/bash
set -e

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-password}"
DB_NAME="${DB_NAME:-iot_db}"

# 添加调试输出
echo "Connecting to: postgres://${DB_USER}:***@${DB_HOST}:${DB_PORT}/${DB_NAME}"

DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

COMMAND=${1:-up}

case $COMMAND in
  up)
    echo "Running migrations..."
    migrate -path ./migrations -database "$DATABASE_URL" up
    ;;
  down)
    echo "Rolling back last migration..."
    migrate -path ./migrations -database "$DATABASE_URL" down 1
    ;;
  force)
    VERSION=$2
    echo "Forcing version to $VERSION..."
    migrate -path ./migrations -database "$DATABASE_URL" force $VERSION
    ;;
  version)
    echo "=== Current migration version (local migrate tool) ==="
    migrate -path ./migrations -database "$DATABASE_URL" version
    echo ""
    echo "=== Docker database schema_migrations ==="
    docker-compose exec postgres psql -U postgres -d iot_db -c "SELECT * FROM schema_migrations;" 2>/dev/null || echo "Cannot connect to Docker database"
    ;;
  create)
    NAME=$2
    echo "Creating new migration: $NAME"
    migrate create -ext sql -dir ./migrations -seq $NAME
    ;;
  *)
    echo "Usage: $0 {up|down|force VERSION|version|create NAME}"
    exit 1
    ;;
esac
