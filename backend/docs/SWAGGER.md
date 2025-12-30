# Swagger API 文檔使用指南

本專案已整合 Swagger，可以自動生成和展示 API 文檔。

## 訪問 Swagger UI

啟動伺服器後，在瀏覽器中訪問：

```
http://localhost:8080/swagger/index.html
```

## 功能特性

1. **自動生成文檔**：所有 API 端點的文檔都會自動從程式碼註釋中生成
2. **互動式測試**：可以直接在 Swagger UI 中測試 API，無需使用 curl 或 Postman
3. **認證支援**：支援 Bearer Token 認證，可以在 Swagger UI 中直接輸入 Token

## 使用步驟

1. 啟動伺服器：
   ```bash
   go run main.go
   ```

2. 打開瀏覽器訪問 `http://localhost:8080/swagger/index.html`

3. 對於需要認證的 API：
   - 點擊右上角的 "Authorize" 按鈕
   - 輸入 Bearer Token（格式：`Bearer your_token_here`）
   - 點擊 "Authorize" 確認

4. 測試 API：
   - 展開任意 API 端點
   - 點擊 "Try it out"
   - 填寫請求參數
   - 點擊 "Execute" 發送請求
   - 查看響應結果

## 更新文檔

當你修改了 API 程式碼或註釋後，需要重新生成 Swagger 文檔：

### 方法 1：使用完整路徑（推薦）

```bash
~/go/bin/swag init
```

### 方法 2：將 swag 添加到 PATH

如果你希望直接使用 `swag` 命令，可以將 Go bin 目錄添加到 PATH：

**對於 zsh（macOS 預設）：**
```bash
echo 'export PATH=$PATH:~/go/bin' >> ~/.zshrc
source ~/.zshrc
```

**對於 bash：**
```bash
echo 'export PATH=$PATH:~/go/bin' >> ~/.bashrc
source ~/.bashrc
```

添加後就可以直接使用：
```bash
swag init
```

### 方法 3：使用便捷腳本（最簡單）

專案中已經提供了一個便捷腳本，可以直接使用：

```bash
./generate-swagger.sh
```

這個腳本會自動找到 swag 並生成文檔。

### 方法 4：在專案目錄中運行

確保你在 `backend` 目錄中運行命令：
```bash
cd backend
~/go/bin/swag init
```

## API 端點分類

- **users**: 使用者相關 API
- **devices**: 設備相關 API
- **telemetries**: 遙測數據相關 API

## 注意事項

- Swagger UI 路由是公開的，不需要認證即可訪問
- 文檔會自動反映程式碼中的註釋和結構體定義
- 所有請求和響應範例都會自動生成

