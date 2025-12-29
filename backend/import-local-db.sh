#!/bin/bash

# 从本地数据库导入数据到 Docker 容器的脚本
# 使用方法：./import-local-db.sh

set -e

echo "📥 从本地数据库导入数据到 Docker 容器..."

# 检查 Docker Compose 服务是否运行
if ! docker-compose ps postgres | grep -q "Up"; then
  echo "❌ PostgreSQL 容器未运行，请先执行: docker compose up -d"
  exit 1
fi

# 检查本地数据库是否存在
echo "检查本地数据库是否存在..."
if ! pg_dump -U postgres -h localhost -d iot_db > /dev/null 2>&1; then
  echo "⚠️  未发现本地数据库 iot_db，跳过导入"
  echo "提示：如果数据库名称不同，请修改脚本中的数据库名称"
  exit 0
fi

echo "✅ 发现本地数据库，正在导入..."
echo "⚠️  警告：这将清空 Docker 数据库中的所有现有数据！"
read -p "是否继续？(y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  echo "已取消操作"
  exit 0
fi

echo "正在清空 Docker 数据库..."
docker-compose exec -T postgres psql -U postgres -d iot_db <<EOF
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
GRANT ALL ON SCHEMA public TO postgres;
GRANT ALL ON SCHEMA public TO public;
EOF
echo "这可能需要一些时间，请稍候..."

# 从本地数据库导出并导入到容器
pg_dump -U postgres -h localhost -d iot_db | \
  docker-compose exec -T postgres psql -U postgres -d iot_db

if [ $? -eq 0 ]; then
  echo "✅ 数据导入完成！"
  echo ""
  echo "📊 验证导入的数据："
  docker-compose exec postgres psql -U postgres -d iot_db -c "SELECT COUNT(*) as device_count FROM devices;"
else
  echo "❌ 数据导入失败，请检查错误信息"
  exit 1
fi

