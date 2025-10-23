package services

import (
	"fmt"
	"math"
	"sort"
	"time"

	"hunt-chat-api/pkg/models"
)

// AnalyzeWeeklySales 週次単位での販売分析（粒度指定可能）
func (s *StatisticsService) AnalyzeWeeklySales(productID, productName string, salesData []models.SalesDataPoint, startDate, endDate time.Time, granularity string) (*models.WeeklyAnalysisResponse, error) {
	if len(salesData) == 0 {
		return nil, fmt.Errorf("販売データが空です")
	}

	// デフォルトは週次
	if granularity == "" {
		granularity = "weekly"
	}

	var weeklySummaries []models.WeeklySummary

	switch granularity {
	case "daily":
		// 日次データ（集約なし）
		weeklySummaries = s.groupByDay(salesData)
	case "monthly":
		// 月次データ
		weeklySummaries = s.groupByMonth(salesData, startDate)
	default: // "weekly"
		// データを週単位でグループ化
		weeklyGroups := s.groupByWeek(salesData, startDate)

		// 週ごとのサマリーを生成
		weeklySummaries = make([]models.WeeklySummary, 0)
		var prevWeekSales float64 = 0

		for weekNum := 0; weekNum < len(weeklyGroups); weekNum++ {
			weekData, exists := weeklyGroups[weekNum]
			if !exists {
				continue
			}
			summary := s.calculateWeeklySummary(weekNum, weekData, prevWeekSales)
			weeklySummaries = append(weeklySummaries, summary)
			prevWeekSales = summary.TotalSales
		}
	}

	// 全体統計を計算
	overallStats := s.calculateWeeklyOverallStats(weeklySummaries)

	// トレンド分析
	trends := s.analyzeWeeklyTrends(weeklySummaries)

	// 推奨事項を生成
	recommendations := s.generateWeeklyRecommendations(weeklySummaries, overallStats, trends)

	return &models.WeeklyAnalysisResponse{
		ProductID:       productID,
		ProductName:     productName,
		AnalysisPeriod:  fmt.Sprintf("%s ~ %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		TotalWeeks:      len(weeklySummaries),
		WeeklySummary:   weeklySummaries,
		OverallStats:    overallStats,
		Trends:          trends,
		Recommendations: recommendations,
		Granularity:     granularity,
	}, nil
}

// groupByWeek データを週単位でグループ化（月曜始まり）
func (s *StatisticsService) groupByWeek(data []models.SalesDataPoint, startDate time.Time) map[int][]models.SalesDataPoint {
	weeklyGroups := make(map[int][]models.SalesDataPoint)

	for _, point := range data {
		date, err := time.Parse("2006-01-02", point.Date)
		if err != nil {
			continue
		}

		// 開始日からの週数を計算（月曜始まり）
		weekNum := s.getWeekNumber(date, startDate)
		weeklyGroups[weekNum] = append(weeklyGroups[weekNum], point)
	}

	return weeklyGroups
}

// groupByDay データを日次でサマリー化（集約なし）
func (s *StatisticsService) groupByDay(data []models.SalesDataPoint) []models.WeeklySummary {
	summaries := make([]models.WeeklySummary, 0, len(data))
	var prevSales float64 = 0

	for i, point := range data {
		_, err := time.Parse("2006-01-02", point.Date)
		if err != nil {
			continue
		}

		var changeRate float64
		if prevSales > 0 {
			changeRate = ((point.Sales - prevSales) / prevSales) * 100
		}

		summaries = append(summaries, models.WeeklySummary{
			WeekNumber:     i + 1,
			WeekStart:      point.Date,
			WeekEnd:        point.Date,
			TotalSales:     point.Sales,
			AverageSales:   point.Sales,
			MinSales:       point.Sales,
			MaxSales:       point.Sales,
			BusinessDays:   1,
			WeekOverWeek:   changeRate,
			StdDev:         0,
			AvgTemperature: point.Temperature,
		})

		prevSales = point.Sales
	}

	return summaries
}

// groupByMonth データを月次で集約
func (s *StatisticsService) groupByMonth(data []models.SalesDataPoint, startDate time.Time) []models.WeeklySummary {
	monthlyGroups := make(map[string][]models.SalesDataPoint)

	// 月ごとにグループ化
	for _, point := range data {
		date, err := time.Parse("2006-01-02", point.Date)
		if err != nil {
			continue
		}
		monthKey := date.Format("2006-01")
		monthlyGroups[monthKey] = append(monthlyGroups[monthKey], point)
	}

	// ソート用にキーを取得
	monthKeys := make([]string, 0, len(monthlyGroups))
	for key := range monthlyGroups {
		monthKeys = append(monthKeys, key)
	}
	sort.Strings(monthKeys)

	// サマリーを生成
	summaries := make([]models.WeeklySummary, 0, len(monthKeys))
	var prevMonthSales float64 = 0

	for i, monthKey := range monthKeys {
		monthData := monthlyGroups[monthKey]
		if len(monthData) == 0 {
			continue
		}

		// 月の開始・終了日を取得
		firstDate, _ := time.Parse("2006-01-02", monthData[0].Date)
		lastDate, _ := time.Parse("2006-01-02", monthData[len(monthData)-1].Date)

		// 合計・平均・最小・最大を計算
		var total, avgTemp, min, max, sumSquaredDiff float64
		min = math.MaxFloat64
		max = -math.MaxFloat64

		for _, point := range monthData {
			total += point.Sales
			avgTemp += point.Temperature
			if point.Sales < min {
				min = point.Sales
			}
			if point.Sales > max {
				max = point.Sales
			}
		}

		businessDays := len(monthData)
		average := total / float64(businessDays)
		avgTemp = avgTemp / float64(businessDays)

		// 前月比を計算
		var monthOverMonth float64
		if prevMonthSales > 0 {
			monthOverMonth = ((total - prevMonthSales) / prevMonthSales) * 100
		}

		// 標準偏差を計算
		for _, point := range monthData {
			diff := point.Sales - average
			sumSquaredDiff += diff * diff
		}
		stdDev := math.Sqrt(sumSquaredDiff / float64(businessDays))

		summaries = append(summaries, models.WeeklySummary{
			WeekNumber:     i + 1,
			WeekStart:      firstDate.Format("2006-01-02"),
			WeekEnd:        lastDate.Format("2006-01-02"),
			TotalSales:     total,
			AverageSales:   average,
			MinSales:       min,
			MaxSales:       max,
			BusinessDays:   businessDays,
			WeekOverWeek:   monthOverMonth,
			StdDev:         stdDev,
			AvgTemperature: avgTemp,
		})

		prevMonthSales = total
	}

	return summaries
}

// getWeekNumber 開始日からの週番号を計算（月曜始まり）
func (s *StatisticsService) getWeekNumber(date, startDate time.Time) int {
	// 月曜日に調整
	startMonday := s.adjustToMonday(startDate)
	dateMonday := s.adjustToMonday(date)

	daysDiff := dateMonday.Sub(startMonday).Hours() / 24
	weekNum := int(daysDiff) / 7

	if weekNum < 0 {
		weekNum = 0
	}

	return weekNum
}

// adjustToMonday 日付をその週の月曜日に調整
func (s *StatisticsService) adjustToMonday(date time.Time) time.Time {
	weekday := int(date.Weekday())
	if weekday == 0 { // 日曜日
		weekday = 7
	}
	daysToMonday := weekday - 1
	return date.AddDate(0, 0, -daysToMonday)
}

// calculateWeeklySummary 週ごとのサマリーを計算
func (s *StatisticsService) calculateWeeklySummary(weekNum int, weekData []models.SalesDataPoint, prevWeekSales float64) models.WeeklySummary {
	if len(weekData) == 0 {
		return models.WeeklySummary{WeekNumber: weekNum}
	}

	// 週の開始日・終了日を取得
	firstDate, _ := time.Parse("2006-01-02", weekData[0].Date)
	lastDate, _ := time.Parse("2006-01-02", weekData[len(weekData)-1].Date)

	// 合計・平均・最小・最大を計算
	var total, avgTemp float64
	min := math.MaxFloat64
	max := -math.MaxFloat64

	for _, point := range weekData {
		total += point.Sales
		avgTemp += point.Temperature
		if point.Sales < min {
			min = point.Sales
		}
		if point.Sales > max {
			max = point.Sales
		}
	}

	businessDays := len(weekData)
	average := total / float64(businessDays)
	avgTemp = avgTemp / float64(businessDays)

	// 前週比を計算
	var weekOverWeek float64
	if prevWeekSales > 0 {
		weekOverWeek = ((total - prevWeekSales) / prevWeekSales) * 100
	}

	// 標準偏差を計算
	var sumSquaredDiff float64
	for _, point := range weekData {
		diff := point.Sales - average
		sumSquaredDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSquaredDiff / float64(businessDays))

	return models.WeeklySummary{
		WeekNumber:     weekNum + 1, // 1始まりに
		WeekStart:      firstDate.Format("2006-01-02"),
		WeekEnd:        lastDate.Format("2006-01-02"),
		TotalSales:     total,
		AverageSales:   average,
		MinSales:       min,
		MaxSales:       max,
		BusinessDays:   businessDays,
		WeekOverWeek:   weekOverWeek,
		StdDev:         stdDev,
		AvgTemperature: avgTemp,
	}
}

// calculateWeeklyOverallStats 全体統計を計算
func (s *StatisticsService) calculateWeeklyOverallStats(summaries []models.WeeklySummary) models.WeeklyOverallStats {
	if len(summaries) == 0 {
		return models.WeeklyOverallStats{}
	}

	// 週次売上を集計
	weeklySales := make([]float64, len(summaries))
	var total float64
	var bestWeek, worstWeek int
	var bestSales, worstSales float64 = -1, math.MaxFloat64

	for i, summary := range summaries {
		weeklySales[i] = summary.TotalSales
		total += summary.TotalSales

		if summary.TotalSales > bestSales {
			bestSales = summary.TotalSales
			bestWeek = summary.WeekNumber
		}
		if summary.TotalSales < worstSales {
			worstSales = summary.TotalSales
			worstWeek = summary.WeekNumber
		}
	}

	avgWeeklySales := total / float64(len(summaries))

	// 中央値を計算
	sortedSales := make([]float64, len(weeklySales))
	copy(sortedSales, weeklySales)
	sort.Float64s(sortedSales)

	var median float64
	mid := len(sortedSales) / 2
	if len(sortedSales)%2 == 0 {
		median = (sortedSales[mid-1] + sortedSales[mid]) / 2
	} else {
		median = sortedSales[mid]
	}

	// 標準偏差を計算
	var sumSquaredDiff float64
	for _, sales := range weeklySales {
		diff := sales - avgWeeklySales
		sumSquaredDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSquaredDiff / float64(len(weeklySales)))

	// 成長率を計算（最初の週 vs 最後の週）
	var growthRate float64
	if len(summaries) >= 2 && summaries[0].TotalSales > 0 {
		firstWeek := summaries[0].TotalSales
		lastWeek := summaries[len(summaries)-1].TotalSales
		growthRate = ((lastWeek - firstWeek) / firstWeek) * 100
	}

	// 変動係数（ボラティリティ）
	var volatility float64
	if avgWeeklySales > 0 {
		volatility = stdDev / avgWeeklySales
	}

	return models.WeeklyOverallStats{
		AverageWeeklySales: avgWeeklySales,
		MedianWeeklySales:  median,
		StdDevWeeklySales:  stdDev,
		BestWeek:           bestWeek,
		WorstWeek:          worstWeek,
		GrowthRate:         growthRate,
		Volatility:         volatility,
	}
}

// analyzeWeeklyTrends 週次トレンドを分析
func (s *StatisticsService) analyzeWeeklyTrends(summaries []models.WeeklySummary) models.WeeklyTrends {
	if len(summaries) < 2 {
		return models.WeeklyTrends{Direction: "データ不足"}
	}

	// 前週比の平均を計算
	var totalGrowth float64
	var positiveWeeks, negativeWeeks int
	var peakWeek, lowWeek int
	var peakSales, lowSales float64 = -1, math.MaxFloat64

	for i, summary := range summaries {
		if i > 0 { // 最初の週はスキップ
			totalGrowth += summary.WeekOverWeek
			if summary.WeekOverWeek > 0 {
				positiveWeeks++
			} else if summary.WeekOverWeek < 0 {
				negativeWeeks++
			}
		}

		if summary.TotalSales > peakSales {
			peakSales = summary.TotalSales
			peakWeek = summary.WeekNumber
		}
		if summary.TotalSales < lowSales {
			lowSales = summary.TotalSales
			lowWeek = summary.WeekNumber
		}
	}

	avgGrowth := totalGrowth / float64(len(summaries)-1)

	// トレンド方向を判定
	var direction string
	var strength float64

	if avgGrowth > 2 {
		direction = "上昇"
		strength = math.Min(avgGrowth/10, 1.0)
	} else if avgGrowth < -2 {
		direction = "下降"
		strength = math.Min(math.Abs(avgGrowth)/10, 1.0)
	} else {
		direction = "横ばい"
		strength = 1.0 - math.Min(math.Abs(avgGrowth)/2, 1.0)
	}

	// 季節性の検出（簡易版）
	var seasonality string
	if len(summaries) >= 4 {
		// 前半と後半で比較
		midPoint := len(summaries) / 2
		var firstHalfAvg, secondHalfAvg float64

		for i := 0; i < midPoint; i++ {
			firstHalfAvg += summaries[i].TotalSales
		}
		firstHalfAvg /= float64(midPoint)

		for i := midPoint; i < len(summaries); i++ {
			secondHalfAvg += summaries[i].TotalSales
		}
		secondHalfAvg /= float64(len(summaries) - midPoint)

		diff := ((secondHalfAvg - firstHalfAvg) / firstHalfAvg) * 100
		if diff > 15 {
			seasonality = "後半期に需要増加傾向"
		} else if diff < -15 {
			seasonality = "前半期に需要集中傾向"
		} else {
			seasonality = "明確な季節パターンなし"
		}
	}

	return models.WeeklyTrends{
		Direction:     direction,
		Strength:      strength,
		Seasonality:   seasonality,
		PeakWeek:      peakWeek,
		LowWeek:       lowWeek,
		AverageGrowth: avgGrowth,
	}
}

// generateWeeklyRecommendations 週次分析に基づく推奨事項を生成
func (s *StatisticsService) generateWeeklyRecommendations(summaries []models.WeeklySummary, stats models.WeeklyOverallStats, trends models.WeeklyTrends) []string {
	var recommendations []string

	// トレンドに基づく推奨
	switch trends.Direction {
	case "上昇":
		recommendations = append(recommendations,
			fmt.Sprintf("📈 上昇トレンド（平均+%.1f%%/週）：需要増加に備えて生産能力の確保を推奨", trends.AverageGrowth))
	case "下降":
		recommendations = append(recommendations,
			fmt.Sprintf("📉 下降トレンド（平均%.1f%%/週）：在庫最適化とマーケティング強化を検討", trends.AverageGrowth))
	case "横ばい":
		recommendations = append(recommendations,
			"📊 安定した需要パターン：現状の生産計画を維持することを推奨")
	}

	// ボラティリティに基づく推奨
	if stats.Volatility > 0.3 {
		recommendations = append(recommendations,
			fmt.Sprintf("⚠️ 需要変動が大きいです（変動係数: %.2f）：安全在庫の確保を推奨", stats.Volatility))
	} else if stats.Volatility < 0.15 {
		recommendations = append(recommendations,
			"✅ 需要が安定しています：JIT生産方式の適用を検討可能")
	}

	// ベスト・ワースト週に基づく推奨
	if stats.BestWeek > 0 && stats.WorstWeek > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("📅 第%d週が最高、第%d週が最低需要：パターン分析で生産計画を最適化", stats.BestWeek, stats.WorstWeek))
	}

	// 成長率に基づく推奨
	if stats.GrowthRate > 20 {
		recommendations = append(recommendations,
			fmt.Sprintf("🚀 期間全体で%.1f%%成長：需要急増に対応した供給体制の強化が必要", stats.GrowthRate))
	} else if stats.GrowthRate < -20 {
		recommendations = append(recommendations,
			fmt.Sprintf("📊 期間全体で%.1f%%減少：需要回復施策の立案を推奨", stats.GrowthRate))
	}

	// 季節性に基づく推奨
	if trends.Seasonality != "明確な季節パターンなし" {
		recommendations = append(recommendations,
			fmt.Sprintf("🌤️ %s：季節要因を考慮した在庫管理を実施", trends.Seasonality))
	}

	return recommendations
}
