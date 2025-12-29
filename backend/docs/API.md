# API 文檔

本文件詳細說明 IoT Device Management API 的所有端點和使用方式。

## 📡 API 端點

### 公開端點（無需認證）

| 方法 | 路徑 | 說明 |
|------|------|------|
| GET | `/` | API 歡迎訊息 |
| POST | `/users` | 創建使用者 |
| GET | `/ws` | WebSocket 連線端點（用於實時數據推播） |

### 私有端點（需要認證）

所有私有端點都需要在 Header 中提供：
```
Authorization: Bearer secret-token-123
```

#### 使用者相關

| 方法 | 路徑 | 說明 |
|------|------|------|
| GET | `/users` | 取得所有使用者列表 |
| GET | `/users/{id}` | 取得單一使用者 |
| GET | `/me` | 取得當前登入者資訊 |

#### 設備相關

| 方法 | 路徑 | 說明 |
|------|------|------|
| POST | `/devices` | 創建設備 |
| GET | `/devices` | 取得所有設備列表 |
| GET | `/devices/{id}` | 取得單一設備（包含遙測數據） |
| PUT | `/devices/{id}` | 完整更新設備（所有字段必須提供） |
| PATCH | `/devices/{id}` | 部分更新設備（只需提供要更新的字段） |
| DELETE | `/devices/{id}` | 刪除設備及其所有遙測數據 |

#### 遙測數據相關

| 方法 | 路徑 | 說明 |
|------|------|------|
| POST | `/telemetries` | 創建遙測數據（會自動推播給所有 WebSocket 客戶端） |
| GET | `/telemetries/{id}` | 取得單一遙測數據 |
| PATCH | `/telemetries/{id}` | 部分更新遙測數據（會自動推播給所有 WebSocket 客戶端） |

## 📝 API 請求範例

### 創建設備

```bash
POST /devices
Authorization: Bearer secret-token-123
Content-Type: application/json

{
  "name": "Temperature Sensor 1",
  "type": "Sensor",
  "mac_address": "00:11:22:33:44:55",
  "is_active": true
}
```

**回應範例：**

```json
{
  "id": 1,
  "name": "Temperature Sensor 1",
  "type": "Sensor",
  "mac_address": "00:11:22:33:44:55",
  "is_active": true,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### 部分更新設備（PATCH）

```bash
PATCH /devices/1
Authorization: Bearer secret-token-123
Content-Type: application/json

{
  "name": "Updated Sensor Name"
}
```

**說明：**
- PATCH 只需要提供要更新的字段
- 其他字段保持不變

### 完整更新設備（PUT）

```bash
PUT /devices/1
Authorization: Bearer secret-token-123
Content-Type: application/json

{
  "name": "Temperature Sensor 1",
  "type": "Sensor",
  "mac_address": "00:11:22:33:44:55",
  "is_active": false
}
```

**說明：**
- PUT 需要提供所有必填字段
- 未提供的字段可能會被設為預設值或清空

### 創建遙測數據

```bash
POST /telemetries
Authorization: Bearer secret-token-123
Content-Type: application/json

{
  "device_id": 1,
  "data_type": "Temperature",
  "value": 25.5,
  "recorded_at": "2024-01-15T10:30:00Z"
}
```

**說明：**
- `recorded_at` 為可選字段，如果不提供則使用當前時間
- 創建後會自動推播給所有訂閱的 WebSocket 客戶端

**回應範例：**

```json
{
  "id": 1,
  "device_id": 1,
  "data_type": "Temperature",
  "value": 25.5,
  "recorded_at": "2024-01-15T10:30:00Z",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### 取得設備列表

```bash
GET /devices
Authorization: Bearer secret-token-123
```

**回應範例：**

```json
[
  {
    "id": 1,
    "name": "Temperature Sensor 1",
    "type": "Sensor",
    "mac_address": "00:11:22:33:44:55",
    "is_active": true,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
]
```

### 取得單一設備（包含遙測數據）

```bash
GET /devices/1
Authorization: Bearer secret-token-123
```

**回應範例：**

```json
{
  "id": 1,
  "name": "Temperature Sensor 1",
  "type": "Sensor",
  "mac_address": "00:11:22:33:44:55",
  "is_active": true,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z",
  "telemetries": [
    {
      "id": 1,
      "data_type": "Temperature",
      "value": 25.5,
      "recorded_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### 刪除設備

```bash
DELETE /devices/1
Authorization: Bearer secret-token-123
```

**說明：**
- 刪除設備會同時刪除該設備的所有遙測數據
- 此操作不可恢復

**回應範例：**

```json
{
  "status": "deleted"
}
```

## 🔐 認證說明

目前系統使用簡單的 Bearer Token 認證：

```
Authorization: Bearer secret-token-123
```

**注意事項：**
- 生產環境應使用 JWT 或其他更安全的認證機制
- Token 目前為硬編碼，僅供開發和測試使用

## 📊 錯誤回應格式

當請求失敗時，API 會返回以下格式的錯誤訊息：

```json
{
  "error": "錯誤訊息描述"
}
```

**常見 HTTP 狀態碼：**

- `200 OK` - 請求成功
- `201 Created` - 資源創建成功
- `400 Bad Request` - 請求參數錯誤
- `401 Unauthorized` - 未授權（缺少或無效的 Token）
- `404 Not Found` - 資源不存在
- `500 Internal Server Error` - 伺服器內部錯誤

## 🔗 相關文檔

- [WebSocket 實時推播](../docs/WEBSOCKET.md)
- [架構設計](../docs/ARCHITECTURE.md)
- [故障排除](../docs/TROUBLESHOOTING.md)

