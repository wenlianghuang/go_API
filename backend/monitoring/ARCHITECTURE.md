# 監控架構設計

## 🎯 設計目標

將原本只有 metrics 端點的簡單監控，升級為完整的視覺化監控系統，使用 Prometheus + Grafana 架構。

## 📊 架構圖

```
┌─────────────────────────────────────────────────────────────┐
│                    Docker Network (monitoring)               │
│                                                              │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────┐  │
│  │   API        │      │  Prometheus  │      │ Grafana  │  │
│  │  Service     │─────▶│   Server     │◀────│  Server  │  │
│  │              │      │              │      │          │  │
│  │ :8080 (API)  │      │ :9091        │      │ :3000    │  │
│  │ :9090 (metrics)│    │ (抓取&存儲)   │      │ (視覺化)  │  │
│  └──────────────┘      └──────────────┘      └──────────┘  │
│         │                      │                    │        │
│         └──────────────────────┴────────────────────┘        │
│                          (metrics flow)                       │
└─────────────────────────────────────────────────────────────┘
```

## 🔄 數據流程

1. **API 服務** (`api:9090/metrics`)
   - 暴露 Prometheus metrics 端點
   - 記錄 HTTP 請求、持續時間、WebSocket 連線等指標

2. **Prometheus** (`prometheus:9090`)
   - 每 15 秒抓取一次 API metrics
   - 存儲時間序列數據（保留 30 天）
   - 提供查詢 API

3. **Grafana** (`grafana:3000`)
   - 從 Prometheus 讀取數據
   - 提供視覺化儀表板
   - 自動配置數據源和 Dashboard

## 📁 文件結構

```
monitoring/
├── README.md                          # 使用說明
├── ARCHITECTURE.md                    # 本文件（架構說明）
├── prometheus/
│   └── prometheus.yml                 # Prometheus 配置
│       ├── 抓取目標：api:9090/metrics
│       ├── 抓取間隔：15s
│       └── 數據保留：30d
└── grafana/
    └── provisioning/
        ├── datasources/
        │   └── prometheus.yml         # 自動配置 Prometheus 數據源
        └── dashboards/
            ├── dashboard.yml          # Dashboard provider 配置
            └── iot-api-dashboard.json # IoT API 監控儀表板
                ├── HTTP 請求速率
                ├── HTTP 請求持續時間
                ├── WebSocket 連線數
                ├── HTTP 狀態碼分佈
                └── 請求速率表格
```

## 🎨 Grafana Dashboard 設計

### 面板布局

```
┌─────────────────────────┬─────────────────────────┐
│ HTTP 請求速率 (按方法)   │ HTTP 持續時間 P95      │
│                         │ (按路徑)               │
├─────────────────────────┼─────────────────────────┤
│ WebSocket 活躍連線數     │ HTTP 狀態碼分佈        │
│                         │ (圓餅圖)               │
├───────────────────────────────────────────────────┤
│        請求速率表格 (按路徑和方法)                  │
└───────────────────────────────────────────────────┘
```

### 監控指標

| 指標名稱 | 類型 | 說明 |
|---------|------|------|
| `http_requests_total` | Counter | HTTP 請求總數（標籤：method, path, status）|
| `http_request_duration_seconds` | Histogram | HTTP 請求持續時間（標籤：method, path）|
| `websocket_active_connections` | Gauge | WebSocket 活躍連線數 |

## 🔧 配置要點

### Prometheus 配置

- **抓取間隔**: 15 秒（平衡實時性和資源消耗）
- **數據保留**: 30 天（足夠進行趨勢分析）
- **目標配置**: 使用 Docker 服務名稱 `api:9090`

### Grafana 配置

- **自動配置**: 使用 Provisioning 自動配置數據源和 Dashboard
- **刷新間隔**: 10 秒（Dashboard 自動刷新）
- **認證**: 預設 admin/admin（生產環境需修改）

### Docker 網絡

- 所有服務加入 `monitoring` 網絡
- 使用服務名稱進行內部通信
- 避免端口衝突（Prometheus 使用 9091 對外）

## 🚀 啟動流程

1. **啟動所有服務**
   ```bash
   docker-compose up -d
   ```

2. **服務啟動順序**
   - PostgreSQL (健康檢查)
   - API 服務
   - Redis
   - Prometheus (等待 API metrics 可用)
   - Grafana (等待 Prometheus 可用)

3. **驗證服務**
   - API: http://localhost:8080
   - Metrics: http://localhost:9090/metrics
   - Prometheus: http://localhost:9091
   - Grafana: http://localhost:3000

## 📈 優勢

### 相比之前的改進

1. **視覺化**: 從原始 metrics 文字 → 美觀的圖表和儀表板
2. **歷史數據**: Prometheus 存儲 30 天歷史數據，可進行趨勢分析
3. **查詢能力**: 使用 PromQL 進行複雜查詢和分析
4. **告警準備**: 架構支持未來添加告警規則
5. **易於擴展**: 可輕鬆添加新的監控指標和 Dashboard

### 生產環境建議

1. **安全性**
   - 修改 Grafana 預設密碼
   - 使用 HTTPS
   - 限制 Prometheus 和 Grafana 的網絡訪問

2. **性能**
   - 根據實際負載調整抓取間隔
   - 配置適當的數據保留策略
   - 考慮使用 Prometheus 集群

3. **告警**
   - 配置 Alertmanager
   - 設置關鍵指標的告警規則
   - 集成通知渠道（Email, Slack 等）

## 🔗 相關文檔

- [使用說明](README.md)
- [Prometheus 官方文檔](https://prometheus.io/docs/)
- [Grafana 官方文檔](https://grafana.com/docs/)

