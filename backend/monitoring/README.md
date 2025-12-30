# 監控系統說明

本目錄包含 Prometheus 和 Grafana 的配置檔案，用於監控 IoT API 服務。

## 📊 架構概覽

```
┌─────────────┐      ┌──────────────┐      ┌─────────────┐
│   API       │      │  Prometheus  │      │   Grafana   │
│  :9090      │─────▶│   :9091      │◀────│   :3000     │
│  /metrics   │      │  (抓取&存儲)  │      │  (視覺化)   │
└─────────────┘      └──────────────┘      └─────────────┘
```

## 🚀 快速開始

### 1. 啟動所有服務（包含監控）

```bash
docker-compose up -d
```

### 2. 訪問服務

- **API**: http://localhost:8080
- **Metrics 端點**: http://localhost:9090/metrics
- **Prometheus**: http://localhost:9091
- **Grafana**: http://localhost:3000
  - 預設帳號: `admin`
  - 預設密碼: `admin`

### 3. 查看監控儀表板

1. 登入 Grafana (http://localhost:3000)
2. 進入 **Dashboards** → **Browse**
3. 選擇 **IoT API 監控儀表板**

## 📁 目錄結構

```
monitoring/
├── README.md                    # 本文件
├── prometheus/
│   └── prometheus.yml           # Prometheus 配置
└── grafana/
    └── provisioning/
        ├── datasources/
        │   └── prometheus.yml    # Grafana 數據源配置
        └── dashboards/
            ├── dashboard.yml     # Dashboard provider 配置
            └── iot-api-dashboard.json  # IoT API 監控儀表板
```

## 📈 監控指標

### HTTP 指標

- `http_requests_total`: HTTP 請求總數（按 method, path, status 分類）
- `http_request_duration_seconds`: HTTP 請求持續時間（histogram）

### WebSocket 指標

- `websocket_active_connections`: 當前活躍的 WebSocket 連線數

## 🎨 Grafana Dashboard 面板

### 1. HTTP 請求速率 (按方法)
顯示不同 HTTP 方法（GET, POST, PUT, DELETE 等）的請求速率

### 2. HTTP 請求持續時間 P95 (按路徑)
顯示各 API 路徑的 95 百分位請求持續時間

### 3. WebSocket 活躍連線數
即時顯示當前活躍的 WebSocket 連線數量

### 4. HTTP 狀態碼分佈
以圓餅圖顯示不同 HTTP 狀態碼的請求分佈

### 5. 請求速率 (按路徑和方法)
表格形式顯示各 API 端點的請求速率

## ⚙️ 配置說明

### Prometheus 配置

`prometheus/prometheus.yml` 定義了：
- 抓取間隔：15 秒
- 目標服務：`api:9090/metrics`
- 數據保留：30 天

### Grafana 配置

- **數據源**: 自動配置 Prometheus 數據源
- **Dashboard**: 自動載入 IoT API 監控儀表板
- **刷新間隔**: 10 秒

## 🔧 自訂配置

### 修改 Prometheus 抓取間隔

編輯 `monitoring/prometheus/prometheus.yml`:

```yaml
global:
  scrape_interval: 10s  # 改為 10 秒
```

### 修改 Grafana 密碼

編輯 `docker-compose.yml` 中的環境變數：

```yaml
grafana:
  environment:
    - GF_SECURITY_ADMIN_PASSWORD=your_new_password
```

### 添加新的監控指標

1. 在 `api/metrics.go` 中添加新的指標
2. 在 Grafana dashboard 中添加新的面板
3. 重啟服務：`docker-compose restart grafana`

## 🐛 故障排除

### Prometheus 無法抓取 metrics

1. 檢查 API 服務是否運行：`docker-compose ps`
2. 檢查 metrics 端點是否可訪問：`curl http://localhost:9090/metrics`
3. 檢查 Prometheus 配置：`docker-compose logs prometheus`

### Grafana 無法連接 Prometheus

1. 確認 Prometheus 服務運行：`docker-compose ps prometheus`
2. 檢查網絡連接：確認兩個服務都在 `monitoring` 網絡中
3. 查看 Grafana 日誌：`docker-compose logs grafana`

### Dashboard 沒有數據

1. 確認 Prometheus 正在抓取數據：訪問 http://localhost:9091/targets
2. 確認指標名稱正確：在 Prometheus 查詢界面測試查詢
3. 檢查時間範圍：確保選擇的時間範圍內有數據

## 📚 相關資源

- [Prometheus 官方文檔](https://prometheus.io/docs/)
- [Grafana 官方文檔](https://grafana.com/docs/)
- [PromQL 查詢語言](https://prometheus.io/docs/prometheus/latest/querying/basics/)

