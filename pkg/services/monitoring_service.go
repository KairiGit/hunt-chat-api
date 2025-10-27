package services

import (
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// LogEntry は単一のリクエストログを表します。
type LogEntry struct {
	Timestamp    time.Time
	Path         string
	Method       string
	StatusCode   int
	ResponseTime time.Duration
}

// MonitoringService はAPIのモニタリング機能を提供します。
type MonitoringService struct {
	logs []LogEntry
	mu   sync.RWMutex
}

// NewMonitoringService は新しいMonitoringServiceを生成します。
func NewMonitoringService() *MonitoringService {
	return &MonitoringService{
		logs: make([]LogEntry, 0),
	}
}

// LogRequest はリクエストを記録します。
func (s *MonitoringService) LogRequest(entry LogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, entry)
}

// LoggingMiddleware はリクエスト情報を記録するGinミドルウェアです。
func (s *MonitoringService) LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 次のミドルウェア/ハンドラを実行
		c.Next()

		// 除外するパスプレフィックス
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/v1/admin") || strings.HasPrefix(path, "/api/v1/monitoring") {
			return
		}

		// リクエスト情報を記録
		entry := LogEntry{
			Timestamp:    start,
			Path:         path,
			Method:       c.Request.Method,
			StatusCode:   c.Writer.Status(),
			ResponseTime: time.Since(start),
		}
		s.LogRequest(entry)
	}
}

// DashboardData はダッシュボードに表示するための集計済みデータです。
type DashboardData struct {
	RequestsOverTime []map[string]interface{} `json:"requestsOverTime"`
	Endpoints        map[string]int             `json:"endpoints"`
	StatusCodes      []map[string]interface{} `json:"statusCodes"`
	AvgResponseTimes []map[string]interface{} `json:"avgResponseTimes"`
	RecentErrors     []LogEntry                 `json:"recentErrors"`
}

// GetDashboardData は指定された期間のログを集計してダッシュボード用データを返します。
func (s *MonitoringService) GetDashboardData(periodHours int) DashboardData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// JSTタイムゾーンを取得
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		// エラーハンドリング: JSTが取得できない場合はUTCをフォールバックとして使用
		jst = time.UTC
	}

	now := time.Now().In(jst)
	since := now.Add(-time.Duration(periodHours) * time.Hour)

	filteredLogs := make([]LogEntry, 0)
	for _, log := range s.logs {
		if log.Timestamp.After(since) {
			filteredLogs = append(filteredLogs, log)
		}
	}

	// requestsOverTime の集計
	requestsOverTimeSlice := make([]map[string]interface{}, periodHours)
	hourlyBuckets := make(map[string]int)

	// 時間のバケットを初期化し、スライスの順序を確定させる
	for i := 0; i < periodHours; i++ {
		// 過去から現在へ向かう順序で生成
		targetTime := now.Add(-time.Duration(periodHours-1-i) * time.Hour)
		hourKey := targetTime.Format("15:00") // 表示用のキー (JST)
		bucketKey := targetTime.Truncate(time.Hour).Format(time.RFC3339)
		hourlyBuckets[bucketKey] = 0
		requestsOverTimeSlice[i] = map[string]interface{}{"time": hourKey, "requests": 0}
	}

	// ログを集計
	for _, log := range filteredLogs {
		bucketKey := log.Timestamp.In(jst).Truncate(time.Hour).Format(time.RFC3339)
		hourlyBuckets[bucketKey]++
	}

	// 集計結果をスライスに反映
	for i := 0; i < periodHours; i++ {
		targetTime := now.Add(-time.Duration(periodHours-1-i) * time.Hour)
		bucketKey := targetTime.Truncate(time.Hour).Format(time.RFC3339)
		if count, ok := hourlyBuckets[bucketKey]; ok {
			requestsOverTimeSlice[i]["requests"] = count
		}
	}

	// endpoints の集計
	endpoints := make(map[string]int)
	for _, log := range filteredLogs {
		endpoints[log.Path]++
	}

	// statusCodes の集計
	statusCodes := make(map[string]int)
	statusCodes["2xx Success"] = 0
	statusCodes["4xx Client Error"] = 0
	statusCodes["5xx Server Error"] = 0
	for _, log := range filteredLogs {
		if log.StatusCode >= 200 && log.StatusCode < 300 {
			statusCodes["2xx Success"]++
		} else if log.StatusCode >= 400 && log.StatusCode < 500 {
			statusCodes["4xx Client Error"]++
		} else if log.StatusCode >= 500 {
			statusCodes["5xx Server Error"]++
		}
	}
	statusCodesSlice := make([]map[string]interface{}, 0)
	for name, value := range statusCodes {
		statusCodesSlice = append(statusCodesSlice, map[string]interface{}{"name": name, "value": value})
	}

	// avgResponseTimes の集計
	responseTimeSum := make(map[string]time.Duration)
	responseCount := make(map[string]int)
	for _, log := range filteredLogs {
		responseTimeSum[log.Path] += log.ResponseTime
		responseCount[log.Path]++
	}
	avgResponseTimesSlice := make([]map[string]interface{}, 0)
	for path, totalTime := range responseTimeSum {
		avg := totalTime.Milliseconds() / int64(responseCount[path])
		avgResponseTimesSlice = append(avgResponseTimesSlice, map[string]interface{}{"endpoint": path, "responseTime": avg})
	}

	// recentErrors の集計
	recentErrors := make([]LogEntry, 0)
	for i := len(filteredLogs) - 1; i >= 0; i-- {
		if filteredLogs[i].StatusCode >= 500 {
			// タイムスタンプをJSTに変換して格納
			errorLog := filteredLogs[i]
			// errorLog.Timestamp = errorLog.Timestamp.In(jst) // フロントエンドのtoLocaleString()に任せる
			recentErrors = append(recentErrors, errorLog)
			if len(recentErrors) >= 10 {
				break
			}
		}
	}

	return DashboardData{
		RequestsOverTime: requestsOverTimeSlice,
		Endpoints:        endpoints,
		StatusCodes:      statusCodesSlice,
		AvgResponseTimes: avgResponseTimesSlice,
		RecentErrors:     recentErrors,
	}
}
