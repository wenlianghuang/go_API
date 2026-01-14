#!/bin/bash
set -e

COMMAND=${1:-up}

case $COMMAND in
  up)
    echo "Running migrations in Docker..."
    docker-compose exec api migrate -path /migrations -database "postgres://postgres:password@postgres:5432/iot_db?sslmode=disable" up
    ;;
  down)
    echo "Rolling back last migration in Docker..."
    docker-compose exec api migrate -path /migrations -database "postgres://postgres:password@postgres:5432/iot_db?sslmode=disable" down 1
    ;;
  force)
    VERSION=$2
    if [ -z "$VERSION" ]; then
      echo "Error: VERSION is required for force command"
      echo "Usage: $0 force VERSION"
      exit 1
    fi
    echo "Forcing version to $VERSION in Docker..."
    docker-compose exec api migrate -path /migrations -database "postgres://postgres:password@postgres:5432/iot_db?sslmode=disable" force $VERSION
    ;;
  version)
    echo "=== Migration version in Docker ==="
    docker-compose exec api migrate -path /migrations -database "postgres://postgres:password@postgres:5432/iot_db?sslmode=disable" version
    echo ""
    echo "=== Schema migrations table ==="
    docker-compose exec postgres psql -U postgres -d iot_db -c "SELECT * FROM schema_migrations;"
    ;;
  *)
    echo "Usage: $0 {up|down|force VERSION|version}"
    exit 1
    ;;
esac
