package handlers

import (
	"fmt"
	"math"
	"strings"
	"time"

	"hunt-chat-api/pkg/models"

	"github.com/qdrant/go-client/qdrant"
)

// findIndex finds the index of the first candidate in a slice
func findIndex(slice []string, candidates ...string) int {
	for _, candidate := range candidates {
		for i, item := range slice {
			if strings.EqualFold(item, candidate) {
				return i
			}
		}
	}
	return -1
}

// AnalysisProgress 分析の進捗情報
type AnalysisProgress struct {
	Step       string `json:"step"`        // 処理ステップ名
	Progress   int    `json:"progress"`    // 進捗率 (0-100)
	Message    string `json:"message"`     // 表示メッセージ
	ElapsedMs  int64  `json:"elapsed_ms"`  // 経過時間（ミリ秒）
	TotalSteps int    `json:"total_steps"` // 総ステップ数
	StepIndex  int    `json:"step_index"`  // 現在のステップインデックス
}

// getProductName 製品IDから製品名を取得（簡易版）
func (ah *AIHandler) getProductName(productID string) string {
	productNames := map[string]string{
		"P001": "製品A",
		"P002": "製品B",
		"P003": "製品C",
		"P004": "製品D",
		"P005": "製品E",
	}

	if name, exists := productNames[productID]; exists {
		return name
	}
	return "不明な製品"
}

// generateSampleHistoricalData サンプルの履歴データを生成（テスト用）
func (ah *AIHandler) generateSampleHistoricalData(_ string, days int) []models.SalesDataPoint {
	data := make([]models.SalesDataPoint, days)
	baseDate := time.Now().AddDate(0, 0, -days)
	baseSales := 100.0

	for i := 0; i < days; i++ {
		date := baseDate.AddDate(0, 0, i)
		dayOfWeek := []string{"日", "月", "火", "水", "木", "金", "土"}[date.Weekday()]

		// 曜日効果
		weekdayMultiplier := 1.0
		switch date.Weekday() {
		case time.Saturday, time.Sunday:
			weekdayMultiplier = 1.3 // 週末は30%増
		case time.Friday:
			weekdayMultiplier = 1.15 // 金曜は15%増
		}

		// 季節効果
		seasonalMultiplier := 1.0
		month := date.Month()
		if month >= 6 && month <= 8 {
			seasonalMultiplier = 1.2 // 夏は20%増
		} else if month == 12 || month <= 2 {
			seasonalMultiplier = 0.9 // 冬は10%減
		}

		// トレンド効果（徐々に増加）
		trendEffect := 1.0 + (float64(i) / float64(days) * 0.1)

		// ランダムノイズ
		randomNoise := 0.9 + (0.2 * float64(i%10) / 10.0)

		sales := baseSales * weekdayMultiplier * seasonalMultiplier * trendEffect * randomNoise

		data[i] = models.SalesDataPoint{
			Date:        date.Format("2006-01-02"),
			DayOfWeek:   dayOfWeek,
			Sales:       sales,
			Temperature: 15.0 + float64(month)*1.5 + float64(i%10-5)*0.5,
		}
	}

	return data
}

// generatePatternDescription パターンの説明を生成
func (ah *AIHandler) generatePatternDescription(tag string, avgImpact float64, count int) string {
	impactStr := "影響"
	if avgImpact > 0 {
		impactStr = fmt.Sprintf("平均+%.1f%%の需要増加", avgImpact)
	} else if avgImpact < 0 {
		impactStr = fmt.Sprintf("平均%.1f%%の需要減少", math.Abs(avgImpact))
	}

	return fmt.Sprintf("%sが発生した際、%sの傾向があります（%d件の実績から学習）", tag, impactStr, count)
}

// ヘルパー関数: Payloadから文字列を取得
func getStringFromPayload(payload map[string]*qdrant.Value, key string) string {
	if val, ok := payload[key]; ok && val != nil {
		if strVal := val.GetStringValue(); strVal != "" {
			return strVal
		}
	}
	return ""
}

// ヘルパー関数: Payloadから数値を取得
func getFloatFromPayload(payload map[string]*qdrant.Value, key string) float64 {
	if val, ok := payload[key]; ok && val != nil {
		if doubleVal := val.GetDoubleValue(); doubleVal != 0 {
			return doubleVal
		}
		if intVal := val.GetIntegerValue(); intVal != 0 {
			return float64(intVal)
		}
	}
	return 0
}
