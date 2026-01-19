package api

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse 错误响应结构（保持向后兼容）
// 新的错误处理使用 errors.HandleError，这个结构保留用于向后兼容
type ErrorResponse struct {
	Error string `json:"error"`
}

// WriteJSON 負責將資料轉成 JSON 並寫入 Response
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteError 負責回傳統一格式的錯誤訊息（保留用于向后兼容）
// 新代码应该使用 errors.HandleError
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ErrorResponse{Error: message})
}
