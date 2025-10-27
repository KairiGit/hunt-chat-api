package handlers

import (
	"net/http"
	"sync/atomic"

	config "hunt-chat-api/configs"

	"github.com/gin-gonic/gin"
)

// isMaintenanceMode はサーバーがメンテナンスモードかどうかを示します。
// atomic.Boolを使用して、スレッドセーフな読み書きを保証します。
var isMaintenanceMode atomic.Bool

// AdminHandler は管理者向け操作のハンドラです。
type AdminHandler struct {
	AdminUsername string
	AdminPassword string
}

// NewAdminHandler は新しいAdminHandlerを生成します。
func NewAdminHandler(cfg *config.Config) *AdminHandler {
	return &AdminHandler{
		AdminUsername: cfg.AdminUsername,
		AdminPassword: cfg.AdminPassword,
	}
}

// AdminCredentials は管理者認証のためのリクエストボディです。
type AdminCredentials struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// StartMaintenance はメンテナンスモードを開始します。
func (h *AdminHandler) StartMaintenance(c *gin.Context) {
	var input AdminCredentials
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
		return
	}

	if input.Username != h.AdminUsername || input.Password != h.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	isMaintenanceMode.Store(true)
	c.JSON(http.StatusOK, gin.H{"message": "Maintenance mode started"})
}

// StopMaintenance はメンテナンスモードを停止します。
func (h *AdminHandler) StopMaintenance(c *gin.Context) {
	var input AdminCredentials
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
		return
	}

	if input.Username != h.AdminUsername || input.Password != h.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	isMaintenanceMode.Store(false)
	c.JSON(http.StatusOK, gin.H{"message": "Maintenance mode stopped"})
}

// GetHealthStatus は現在のサーバーの状態を返します。
func (h *AdminHandler) GetHealthStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"isMaintenanceMode": isMaintenanceMode.Load()})
}

// HealthCheck は外部のヘルスチェッカー（例: ロードバランサー）からのリクエストに応答します。
func HealthCheck(c *gin.Context) {
	if isMaintenanceMode.Load() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "message": "Server is in maintenance mode"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
