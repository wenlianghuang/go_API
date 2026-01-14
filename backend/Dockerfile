# --- 第一階段：Builder (編譯階段) ---
# 使用官方 Go 映像檔作為基礎，注意版本要對應您的開發環境
# 由於您的 go.mod 原本寫 1.25.1 (無效版本)，這裡我們用穩定的 1.23
#FROM golang:1.23-alpine AS builder
FROM golang:1.25-alpine AS builder
# 設定工作目錄
WORKDIR /app

# 安裝 migrate CLI 和 git
RUN apk add --no-cache git
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 先複製 go.mod 和 go.sum，利用 Docker Cache 加速依賴下載
COPY go.mod go.sum ./

# 下載依賴
RUN go mod download

# 複製所有原始碼
COPY . .

# 編譯應用程式
# CGO_ENABLED=0 表示建立靜態連結的執行檔，不依賴系統 C 函式庫
# -o backend-binary 指定輸出檔名
RUN CGO_ENABLED=0 GOOS=linux go build -o backend-binary .

# --- 第二階段：Runner (執行階段) ---
# 使用極輕量的 alpine 映像檔 (只有幾 MB)
FROM alpine:latest

# 安裝基本憑證和 PostgreSQL 客戶端工具（用於等待資料庫就緒）
RUN apk --no-cache add ca-certificates tzdata postgresql-client

WORKDIR /root/

# 從第一階段 (builder) 複製必要的檔案
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /app/backend-binary .
COPY --from=builder /app/migrations /migrations
COPY --from=builder /app/scripts/entrypoint.sh /entrypoint.sh

# 設置 entrypoint 腳本為可執行
RUN chmod +x /entrypoint.sh

# 暴露應用程式運行的 Port
EXPOSE 8080 9090

# 容器啟動時執行 entrypoint 腳本（會先執行遷移再啟動應用程式）
ENTRYPOINT ["/entrypoint.sh"]