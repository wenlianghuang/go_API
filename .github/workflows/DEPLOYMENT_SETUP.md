# GitHub Actions 部署配置指南

本文档说明如何配置 GitHub Actions 以支持数据库迁移和部署。

## 📋 工作流说明

`deploy.yaml` 包含三个作业：

1. **push-to-ghcr**：构建并推送 Docker 镜像到 GitHub Container Registry
2. **run-migrations**：执行数据库迁移（在部署前）
3. **deploy**：部署应用到生产环境

## 🔐 需要配置的 GitHub Secrets

在 GitHub 仓库中，进入 **Settings → Secrets and variables → Actions**，添加以下 Secret：

### 必需配置

#### `PRODUCTION_DATABASE_URL`

生产数据库的连接字符串。

**格式**：
```
postgres://username:password@host:port/database?sslmode=require
```

**示例**：
```
postgres://postgres:your-secure-password@db.example.com:5432/iot_db?sslmode=require
```

**安全建议**：
- ✅ 使用 `sslmode=require` 或 `sslmode=verify-full` 确保加密连接
- ✅ 使用强密码
- ✅ 定期轮换密码
- ✅ 限制数据库防火墙只允许部署服务器的 IP

### 可选配置（如果使用分离的数据库参数）

如果不想使用完整的连接字符串，可以分别配置：

- `PRODUCTION_DB_HOST`：数据库主机地址
- `PRODUCTION_DB_PORT`：数据库端口（默认：5432）
- `PRODUCTION_DB_USER`：数据库用户名
- `PRODUCTION_DB_PASSWORD`：数据库密码
- `PRODUCTION_DB_NAME`：数据库名称

如果使用这些分离的参数，需要修改 `deploy.yaml` 中的 `DATABASE_URL` 构建方式。

## 🚀 工作流执行流程

```
Push to main branch
    ↓
1. Build Docker Image
    ↓
2. Push to GHCR
    ↓
3. Run Database Migrations
    ├─ Check current version
    ├─ Execute migrations
    └─ Verify final version
    ↓
4. Deploy Application
    └─ (添加您的部署脚本)
```

## 📝 迁移步骤说明

### 1. 检查当前版本
```bash
migrate -path ./migrations -database "$DATABASE_URL" version
```
- 显示当前数据库的迁移版本
- 如果数据库是新的，可能显示错误（这是正常的）

### 2. 执行迁移
```bash
migrate -path ./migrations -database "$DATABASE_URL" up
```
- 执行所有待处理的迁移
- 自动更新 `schema_migrations` 表

### 3. 验证版本
```bash
migrate -path ./migrations -database "$DATABASE_URL" version
```
- 确认迁移成功执行
- 显示最终的迁移版本号

## ⚠️ 重要注意事项

### 迁移安全

1. **备份数据库**：在生产环境执行迁移前，建议先备份数据库
2. **测试迁移**：在测试环境先验证迁移脚本
3. **回滚计划**：准备回滚方案（使用 `.down.sql` 文件）
4. **监控**：迁移后检查应用是否正常运行

### 错误处理

如果迁移失败：
1. 检查 `schema_migrations` 表的状态（可能显示 `dirty`）
2. 使用 `migrate force` 命令修复状态（谨慎使用）
3. 检查数据库连接和权限
4. 查看迁移日志

### 权限要求

数据库用户需要以下权限：
- `CREATE TABLE`：创建表
- `ALTER TABLE`：修改表结构
- `CREATE INDEX`：创建索引
- `DROP TABLE`：删除表（用于回滚）
- 对 `schema_migrations` 表的读写权限

## 🔧 自定义部署步骤

在 `deploy.yaml` 的 `deploy` 作业中，添加您的实际部署脚本：

### 示例 1：SSH 部署
```yaml
- name: Deploy via SSH
  uses: appleboy/ssh-action@master
  with:
    host: ${{ secrets.SSH_HOST }}
    username: ${{ secrets.SSH_USER }}
    key: ${{ secrets.SSH_PRIVATE_KEY }}
    script: |
      docker pull ghcr.io/${{ github.repository }}/my-api:${{ github.sha }}
      docker-compose up -d
```

### 示例 2：Kubernetes 部署
```yaml
- name: Deploy to Kubernetes
  uses: azure/k8s-deploy@v4
  with:
    manifests: |
      k8s/deployment.yaml
    images: |
      ghcr.io/${{ github.repository }}/my-api:${{ github.sha }}
```

### 示例 3：Docker Compose 部署
```yaml
- name: Deploy with Docker Compose
  run: |
    ssh user@server << 'EOF'
      cd /path/to/app
      docker-compose pull
      docker-compose up -d
    EOF
```

## 📊 监控和验证

部署后，建议验证：

1. **应用健康检查**：
   ```bash
   curl https://your-api.com/health
   ```

2. **数据库连接**：
   ```bash
   psql "$DATABASE_URL" -c "SELECT version();"
   ```

3. **迁移版本**：
   ```bash
   migrate -path ./migrations -database "$DATABASE_URL" version
   ```

## 🆘 故障排除

### 问题：迁移步骤失败

**可能原因**：
- 数据库连接字符串错误
- 数据库不可访问
- 权限不足
- 迁移文件有语法错误

**解决方案**：
1. 检查 `PRODUCTION_DATABASE_URL` Secret 是否正确
2. 验证数据库网络连接
3. 检查数据库用户权限
4. 在本地测试迁移脚本

### 问题：迁移显示 "dirty" 状态

**解决方案**：
```bash
# 在本地或通过 SSH 连接到服务器
migrate -path ./migrations -database "$DATABASE_URL" force <version>
```

### 问题：部署后应用无法启动

**检查清单**：
- [ ] 数据库迁移是否成功
- [ ] 环境变量是否正确配置
- [ ] 应用日志是否有错误
- [ ] 数据库连接是否正常

## 📚 相关文档

- [数据库迁移指南](../backend/docs/MIGRATION.md)
- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [golang-migrate 文档](https://github.com/golang-migrate/migrate)
