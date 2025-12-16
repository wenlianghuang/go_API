# --- 第一階段：Builder (編譯階段) ---
# 使用官方 Go 映像檔作為基礎，注意版本要對應您的開發環境
# 由於您的 go.mod 原本寫 1.25.1 (無效版本)，這裡我們用穩定的 1.23
#FROM golang:1.23-alpine AS builder
FROM golang:1.25-alpine AS builder
# 設定工作目錄
WORKDIR /app

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

# 安裝基本憑證 (如果您的 API 需要呼叫外部 HTTPS API，這是必須的)
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 從第一階段 (builder) 複製編譯好的執行檔過來
COPY --from=builder /app/backend-binary .

# 暴露應用程式運行的 Port (假設您的 server.go 是跑在 8080)
EXPOSE 8080

# 容器啟動時執行的指令
CMD ["./backend-binary"]