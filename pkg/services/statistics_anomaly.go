package services

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"hunt-chat-api/pkg/models"
)

// DetectAnomalies 異常検知を実行（デフォルトは週次）
func (s *StatisticsService) DetectAnomalies(sales []float64, dates []string, productID string, productName string) []models.AnomalyDetection {
	return s.DetectAnomaliesWithGranularity(sales, dates, productID, productName, "weekly")
}

// DetectAnomaliesWithGranularity 粒度を指定して異常検知を実行
func (s *StatisticsService) DetectAnomaliesWithGranularity(sales []float64, dates []string, productID string, productName string, granularity string) []models.AnomalyDetection {
	displayName := productName
	if displayName == "" {
		displayName = productID
	}

	// デフォルトは週次
	if granularity == "" {
		granularity = "weekly"
	}

	log.Printf("[異常検知@%s] 粒度: %s でデータを集約してから異常検知を実行します", displayName, granularity)

	// 日次データの場合のみ集約が必要（週次・月次の場合は既に集約済みと仮定）
	aggregatedSales := sales
	aggregatedDates := dates

	if granularity != "daily" && len(sales) > 0 {
		// データを週次または月次に集約
		aggregatedSales, aggregatedDates = s.aggregateDataForAnomalyDetection(sales, dates, granularity)
		log.Printf("[異常検知@%s] データを集約: %d件 → %d件", displayName, len(sales), len(aggregatedSales))
	}

	// 移動平均のウィンドウサイズを粒度に応じて調整
	var windowSize int
	var percentageThreshold float64

	switch granularity {
	case "daily":
		windowSize = 30           // 30日間の移動平均
		percentageThreshold = 0.5 // 50%の乖離
	case "weekly":
		windowSize = 4            // 4週間の移動平均
		percentageThreshold = 0.4 // 40%の乖離（週次は変動が大きいため緩和）
	case "monthly":
		windowSize = 3            // 3ヶ月の移動平均
		percentageThreshold = 0.3 // 30%の乖離（月次はさらに緩和）
	default:
		windowSize = 4
		percentageThreshold = 0.4
	}

	if len(aggregatedSales) < windowSize {
		log.Printf("[異常検知@%s] データが少なく、移動平均を計算できません（%d件 < %d件）", displayName, len(aggregatedSales), windowSize)
		return []models.AnomalyDetection{}
	}

	var anomalies []models.AnomalyDetection

	for i := windowSize; i < len(aggregatedSales); i++ {
		// ウィンドウ内のデータを取得
		window := aggregatedSales[i-windowSize : i]

		// 移動平均を計算
		mean := calculateMean(window)

		// 現在の値
		currentValue := aggregatedSales[i]

		// 移動平均からの乖離を計算
		deviation := currentValue - mean

		// 閾値を計算
		threshold := mean * percentageThreshold

		if mean > 0 && math.Abs(deviation) > threshold {
			anomalyType := "急増"
			if deviation < 0 {
				anomalyType = "急減"
			}

			// Zスコアは参考値として（ウィンドウ内の統計で計算）
			stdDev := calculateStandardDeviation(window)
			var zScore float64
			if stdDev > 0 {
				zScore = deviation / stdDev
			}

			anomalies = append(anomalies, models.AnomalyDetection{
				Date:          aggregatedDates[i],
				ProductID:     productID,
				ProductName:   productName,
				ActualValue:   currentValue,
				ExpectedValue: mean, // 期待値として移動平均を使用
				Deviation:     math.Abs(deviation),
				ZScore:        zScore,
				AnomalyType:   anomalyType,
				Severity:      s.calculateSeverity(math.Abs(zScore)),
			})
		}
	}

	log.Printf("[異常検知@%s] 移動平均法により %d 件の異常を検出しました", displayName, len(anomalies))

	return anomalies
}

// aggregateDataForAnomalyDetection 異常検知用にデータを集約
func (s *StatisticsService) aggregateDataForAnomalyDetection(sales []float64, dates []string, granularity string) ([]float64, []string) {
	if len(sales) != len(dates) {
		log.Printf("[警告] sales と dates の長さが一致しません: %d != %d", len(sales), len(dates))
		return sales, dates
	}

	// 期間キーごとにデータを集約
	periodMap := make(map[string][]float64)
	periodOrder := []string{} // 順序を保持

	for i, dateStr := range dates {
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Printf("[警告] 日付のパースに失敗: %s", dateStr)
			continue
		}

		var periodKey string
		switch granularity {
		case "weekly":
			// 月曜始まりの週番号
			year, week := t.ISOWeek()
			periodKey = fmt.Sprintf("%d-W%02d", year, week)
		case "monthly":
			periodKey = t.Format("2006-01")
		default:
			periodKey = dateStr // 日次の場合はそのまま
		}

		if _, exists := periodMap[periodKey]; !exists {
			periodOrder = append(periodOrder, periodKey)
		}
		periodMap[periodKey] = append(periodMap[periodKey], sales[i])
	}

	// 集約データを生成
	aggregatedSales := make([]float64, 0, len(periodOrder))
	aggregatedDates := make([]string, 0, len(periodOrder))

	for _, periodKey := range periodOrder {
		values := periodMap[periodKey]

		// 合計を計算
		var total float64
		for _, v := range values {
			total += v
		}

		aggregatedSales = append(aggregatedSales, total)
		aggregatedDates = append(aggregatedDates, periodKey)
	}

	return aggregatedSales, aggregatedDates
}

// calculateSeverity 異常の深刻度を計算
func (s *StatisticsService) calculateSeverity(absZScore float64) string {
	if absZScore > 4.0 {
		return "critical" // 極めて異常
	} else if absZScore > 3.5 {
		return "high" // 高度な異常
	} else if absZScore > 3.0 {
		return "medium" // 中程度の異常
	}
	return "low"
}

// formatDateForDisplay 日付を読みやすい形式にフォーマット
func (s *StatisticsService) formatDateForDisplay(date string) string {
	// 月次形式: YYYY-MM
	if len(date) == 7 && date[4] == '-' {
		t, err := time.Parse("2006-01", date)
		if err == nil {
			return t.Format("2006年1月")
		}
	}

	// 週次形式: YYYY-WWW
	if len(date) >= 7 && strings.Contains(date, "-W") {
		parts := strings.Split(date, "-W")
		if len(parts) == 2 {
			return fmt.Sprintf("%s年 第%s週", parts[0], parts[1])
		}
	}

	// 日次形式: YYYY-MM-DD
	if len(date) == 10 {
		t, err := time.Parse("2006-01-02", date)
		if err == nil {
			return t.Format("2006年1月2日")
		}
	}

	// パースできない場合はそのまま返す
	return date
}

// GenerateAIQuestion 異常値に基づいてAIが質問を生成
func (s *StatisticsService) GenerateAIQuestion(anomaly models.AnomalyDetection) (string, []string) {
	// 製品の表示名を決定
	displayName := anomaly.ProductName
	if displayName == "" {
		displayName = anomaly.ProductID
	}

	// 日付を読みやすい形式にフォーマット
	formattedDate := s.formatDateForDisplay(anomaly.Date)

	// AIサービスが利用可能な場合は、AIに質問と選択肢を生成させる
	if s.azureOpenAIService != nil {
		// AnomalyDetectionをAnomalyに変換
		anomalyForAI := models.Anomaly{
			Date:        formattedDate,
			ProductID:   displayName,
			Description: fmt.Sprintf("売上%s (実績: %.0f, 期待値: %.0f)", anomaly.AnomalyType, anomaly.ActualValue, anomaly.ExpectedValue),
		}

		result, err := s.azureOpenAIService.GenerateQuestionAndChoicesFromAnomaly(anomalyForAI)
		if err == nil && result != nil && result.Question != "" {
			return result.Question, result.Choices
		}
		log.Printf("⚠️ AIからの質問生成に失敗しました。フォールバックします。エラー: %v", err)
	}

	// フォールバック：テンプレートベースの質問と固定の選択肢
	var question string
	if anomaly.AnomalyType == "急増" {
		question = fmt.Sprintf(
			"📈 %s に「%s」の売上が通常より %.0f 増加しました（期待値: %.0f → 実績: %.0f）。この時期に特別なイベント、キャンペーン、または外的要因はありましたか？",
			formattedDate,
			displayName,
			anomaly.Deviation,
			anomaly.ExpectedValue,
			anomaly.ActualValue,
		)
	} else {
		question = fmt.Sprintf(
			"📉 %s に「%s」の売上が通常より %.0f 減少しました（期待値: %.0f → 実績: %.0f）。この時期に売上減少の原因となった要因（天候、競合、在庫切れなど）はありましたか？",
			formattedDate,
			displayName,
			anomaly.Deviation,
			anomaly.ExpectedValue,
			anomaly.ActualValue,
		)
	}

	defaultChoices := []string{
		"キャンペーン・販促活動",
		"天候の影響",
		"競合他社の動き",
		"特に思い当たる節はない",
		"その他（自由記述）",
	}

	return question, defaultChoices
}
