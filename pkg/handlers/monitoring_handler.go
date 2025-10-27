package handlers

import (
	"net/http"

	"hunt-chat-api/pkg/services"

	"github.com/gin-gonic/gin"
)

// MonitoringHandler はモニタリング関連の操作のハンドラです。
type MonitoringHandler struct {
	Service *services.MonitoringService
}

// NewMonitoringHandler は新しいMonitoringHandlerを生成します。
func NewMonitoringHandler(service *services.MonitoringService) *MonitoringHandler {
	return &MonitoringHandler{
		Service: service,
	}
}

// GetLogs は集計されたログデータを返します。
func (h *MonitoringHandler) GetLogs(c *gin.Context) {
	periodStr := c.DefaultQuery("period", "24h")
	var hours int

	switch periodStr {
	case "1h":
		hours = 1
	case "24h":
		hours = 24
	case "7d":
		hours = 24 * 7
	default:
		hours = 24
	}

	data := h.Service.GetDashboardData(hours)
	c.JSON(http.StatusOK, data)
}
