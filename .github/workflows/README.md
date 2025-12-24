# GitHub Actions CI/CD 工作流程說明

本目錄包含兩個 GitHub Actions 工作流程，用於自動化測試、構建和部署 Go API 應用程式。

## 📋 目錄

- [工作流程概覽](#工作流程概覽)
- [go-ci.yaml - 持續整合](#go-ciyaml---持續整合)
- [deploy.yaml - 自動部署](#deployyaml---自動部署)
- [常見問題與解決方案](#常見問題與解決方案)
- [使用說明](#使用說明)
- [本地測試 Docker 鏡像](#本地測試-docker-鏡像)

---

## 工作流程概覽

### 1. `go-ci.yaml` - 持續整合 (CI)
- **觸發條件**：推送到任何分支或對 `main` 分支發起 Pull Request
- **功能**：
  - 執行 Go 程式碼格式檢查
  - 執行靜態分析 (`go vet`)
  - 執行單元測試（含競態條件檢測）
  - 構建應用程式
  - 上傳構建成品

### 2. `deploy.yaml` - 自動部署 (CD)
- **觸發條件**：推送到 `main` 分支
- **功能**：
  - 構建 Docker 鏡像
  - 推送到 GitHub Container Registry (GHCR)
  - 建立兩個標籤：`latest` 和 commit SHA

---

## go-ci.yaml - 持續整合

### 工作流程步驟

1. **Checkout 程式碼**
   - 使用 `actions/checkout@v4` 取得程式碼

2. **設定 Go 環境**
   - 使用 Go 1.22 版本
   - 啟用依賴快取以加速構建

3. **修正 go.mod 版本**
   - 將 `go.mod` 中的 `go 1.25.1` 暫時修正為 `go 1.22`
   - 這是因為 CI 環境使用 Go 1.22，需要版本匹配

4. **下載依賴**
   - 執行 `go mod tidy` 下載並整理依賴

5. **執行檢查與測試**
   - 格式檢查：`gofmt`
   - 靜態分析：`go vet`
   - 單元測試：`go test -v -race -coverprofile=coverage.txt`

6. **構建應用程式**
   - 編譯 Go 應用程式為 `backend-binary`

7. **上傳構建成品**
   - 將構建好的二進位檔上傳為 Artifact

### 注意事項

- CI 環境使用 Go 1.22，因此會暫時修改 `go.mod` 中的版本號
- 此修改僅在 CI 環境中生效，不會影響本地開發環境

---

## deploy.yaml - 自動部署

### 工作流程步驟

1. **Checkout 程式碼**
   - 使用 `actions/checkout@v4` 取得程式碼

2. **設定小寫 Repository 名稱**
   - 將 repository 名稱轉換為小寫（Docker 標籤要求）
   - 例如：`go_API` → `go_api`

3. **登入 GitHub Container Registry**
   - 使用 `GITHUB_TOKEN` 自動登入 GHCR
   - 需要 `packages: write` 權限

4. **構建並推送 Docker 鏡像**
   - 構建目錄：`./backend`
   - Dockerfile 路徑：`./backend/Dockerfile`
   - 推送到 GHCR
   - 建立兩個標籤：
     - `ghcr.io/<username>/<repo>/my-api:latest`
     - `ghcr.io/<username>/<repo>/my-api:<commit-sha>`

### 權限設定

確保 GitHub repository 設定中已啟用：
- ✅ GitHub Packages
- ✅ Actions 權限（允許寫入 packages）

---

## 常見問題與解決方案

### ❌ 問題 1：路徑錯誤

**錯誤訊息**：
```
context: ./go_API/backend
```

**原因**：
- GitHub Actions checkout 後，程式碼位於工作目錄根目錄
- 不需要 `go_API` 前綴

**解決方案**：
```yaml
context: ./backend
file: ./backend/Dockerfile
```

---

### ❌ 問題 2：Repository 名稱大小寫錯誤

**錯誤訊息**：
```
ERROR: failed to build: invalid tag "ghcr.io/wenlianghuang/go_API/my-api:latest": 
repository name must be lowercase
```

**原因**：
- Docker 鏡像標籤要求所有字符必須是小寫
- `go_API` 包含大寫字母 `API`

**解決方案**：
在 workflow 中添加步驟，將 repository 名稱轉換為小寫：
```yaml
- name: Set lowercase repository name
  id: repo
  run: echo "name=$(echo '${{ github.repository }}' | tr '[:upper:]' '[:lower:]')" >> $GITHUB_OUTPUT
```

然後在標籤中使用：
```yaml
tags: |
  ghcr.io/${{ steps.repo.outputs.name }}/my-api:latest
  ghcr.io/${{ steps.repo.outputs.name }}/my-api:${{ github.sha }}
```

---

### ❌ 問題 3：Docker 網路不存在

**錯誤訊息**：
```
docker: Error response from daemon: failed to set up container networking: 
network go_api_backend_default not found
```

**原因**：
- 嘗試連接的 Docker 網路尚未建立
- 需要先啟動 docker-compose 服務來建立網路

**解決方案**：

1. **先啟動 PostgreSQL 容器**：
```bash
cd /Users/matthuang/Desktop/go_API/backend
docker-compose up -d postgres
```

2. **查找實際的網路名稱**：
```bash
# 方法 1：查看所有網路
docker network ls | grep backend

# 方法 2：查看 postgres 容器使用的網路
docker inspect $(docker-compose ps -q postgres) -f '{{range $net, $conf := .NetworkSettings.Networks}}{{$net}}{{end}}'
```

3. **使用正確的網路名稱運行容器**：
```bash
docker run -d \
  --name my-api-container \
  --platform linux/amd64 \
  -e DB_DSN="host=postgres user=postgres password=password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei" \
  -p 8080:8080 \
  --network backend_default \
  ghcr.io/wenlianghuang/go_api/my-api:latest
```

**注意**：網路名稱通常是 `<目錄名>_default`，例如 `backend_default`。

---

### ❌ 問題 4：平台不匹配警告

**警告訊息**：
```
WARNING: The requested image's platform (linux/amd64) does not match the 
detected host platform (linux/arm64/v8) and no specific platform was requested
```

**原因**：
- GitHub Actions 在 `ubuntu-latest` 上構建，預設是 `linux/amd64`
- 在 M1/M2 Mac (ARM64) 上運行時會出現平台不匹配

**解決方案**：

**選項 A：指定平台運行（推薦用於測試）**：
```bash
docker run -d \
  --name my-api-container \
  --platform linux/amd64 \
  # ... 其他參數
```

**選項 B：構建多平台鏡像（推薦用於生產）**：
修改 `deploy.yaml`，添加多平台構建：
```yaml
- name: Build and Push
  uses: docker/build-push-action@v5
  with:
    context: ./backend
    file: ./backend/Dockerfile
    push: true
    platforms: linux/amd64,linux/arm64
    tags: |
      ghcr.io/${{ steps.repo.outputs.name }}/my-api:latest
      ghcr.io/${{ steps.repo.outputs.name }}/my-api:${{ github.sha }}
```

---

## 使用說明

### 本地開發流程

1. **開發與測試**
   - 在本地進行開發
   - 推送到功能分支
   - CI 會自動執行測試

2. **合併到 main 分支**
   - 通過 Pull Request 合併到 `main`
   - CI 會再次執行測試確保通過

3. **自動部署**
   - 推送到 `main` 分支後，`deploy.yaml` 會自動觸發
   - 構建 Docker 鏡像並推送到 GHCR
   - 可以在 GitHub Packages 頁面查看鏡像

### 查看構建狀態

1. 前往 GitHub repository
2. 點擊 **Actions** 標籤
3. 查看工作流程執行狀態和日誌

### 查看 Docker 鏡像

1. 前往 GitHub repository
2. 點擊右側的 **Packages**
3. 找到 `my-api` 套件
4. 查看所有版本標籤

---

## 本地測試 Docker 鏡像

### 完整部署流程

```bash
# 1. 進入 backend 目錄
cd /Users/matthuang/Desktop/go_API/backend

# 2. 啟動 PostgreSQL（這會創建網路）
docker-compose up -d postgres

# 3. 等待 PostgreSQL 啟動
sleep 10

# 4. 導入本地數據（如果需要）
./import-local-db.sh

# 5. 拉取最新鏡像
docker pull ghcr.io/wenlianghuang/go_api/my-api:latest

# 6. 查找網路名稱
NETWORK_NAME=$(docker inspect $(docker-compose ps -q postgres) -f '{{range $net, $conf := .NetworkSettings.Networks}}{{$net}}{{end}}')
echo "網路名稱: $NETWORK_NAME"

# 7. 運行 API 容器
docker run -d \
  --name my-api-container \
  --platform linux/amd64 \
  -e DB_DSN="host=postgres user=postgres password=password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei" \
  -p 8080:8080 \
  --network $NETWORK_NAME \
  ghcr.io/wenlianghuang/go_api/my-api:latest
```

### 驗證部署

```bash
# 檢查容器狀態
docker ps

# 查看 API 日誌
docker logs my-api-container

# 測試 API
curl http://localhost:8080/health

# 查看 PostgreSQL 容器日誌
docker compose logs postgres
```

### 停止和清理

```bash
# 停止 API 容器
docker stop my-api-container
docker rm my-api-container

# 停止 PostgreSQL（保留數據）
docker compose stop postgres

# 停止並刪除所有容器（包括數據）
docker compose down -v
```

---

## 標籤說明

部署工作流程會建立兩個標籤：

1. **`latest`**：始終指向最新的構建
   - 格式：`ghcr.io/<username>/<repo>/my-api:latest`

2. **Commit SHA**：指向特定 commit 的構建
   - 格式：`ghcr.io/<username>/<repo>/my-api:<commit-sha>`
   - 範例：`ghcr.io/wenlianghuang/go_api/my-api:bdd843f591dfd493e67d792e117fbadf5f260eda`

**建議**：
- 生產環境使用 commit SHA 標籤（確保版本一致性）
- 開發/測試環境可以使用 `latest` 標籤

---

## 疑難排解

### CI 測試失敗

1. 檢查 Go 版本是否匹配
2. 確認 `go.mod` 中的依賴版本正確
3. 查看 Actions 日誌中的詳細錯誤訊息

### 部署失敗

1. 確認 GitHub Packages 權限已啟用
2. 檢查 `GITHUB_TOKEN` 是否有寫入權限
3. 確認 Dockerfile 路徑正確
4. 檢查構建日誌中的錯誤訊息

### 本地運行問題

1. **網路不存在**：先執行 `docker compose up -d postgres`
2. **平台不匹配**：添加 `--platform linux/amd64` 參數
3. **連接資料庫失敗**：確認 PostgreSQL 容器正在運行，且網路名稱正確

---

## 相關資源

- [GitHub Actions 文件](https://docs.github.com/en/actions)
- [Docker Build Push Action](https://github.com/docker/build-push-action)
- [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Docker Compose 文件](https://docs.docker.com/compose/)

---

## 更新記錄

- **2024-XX-XX**：初始版本
  - 添加 CI 工作流程
  - 添加部署工作流程
  - 解決 repository 名稱大小寫問題
  - 解決路徑問題
  - 添加平台支援說明

