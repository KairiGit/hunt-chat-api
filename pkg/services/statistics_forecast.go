package services

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"hunt-chat-api/pkg/models"
)

// PredictFutureSales 将来の売上を予測する
func (s *StatisticsService) PredictFutureSales(
	historicalSales []float64,
	historicalTemperatures []float64,
	futureTemperature float64,
	confidenceLevel float64,
) (models.SalesPrediction, error) {
	if len(historicalSales) != len(historicalTemperatures) {
		return models.SalesPrediction{}, fmt.Errorf("データ系列の長さが一致しません")
	}

	if len(historicalSales) < 10 {
		return models.SalesPrediction{}, fmt.Errorf("予測には最低10件のデータが必要です")
	}

	// 1. 回帰分析で予測値を計算
	regression, err := s.PerformLinearRegression(historicalTemperatures, historicalSales)
	if err != nil {
		return models.SalesPrediction{}, err
	}

	predictedValue := regression.Slope*futureTemperature + regression.Intercept

	// 2. 残差の標準偏差を計算（予測の不確実性）
	var residuals []float64
	for i := 0; i < len(historicalSales); i++ {
		predicted := regression.Slope*historicalTemperatures[i] + regression.Intercept
		residual := historicalSales[i] - predicted
		residuals = append(residuals, residual)
	}

	residualStdDev := calculateStandardDeviation(residuals)

	// 3. 信頼区間を計算（デフォルト95%）
	if confidenceLevel == 0 {
		confidenceLevel = 0.95
	}

	// z値（正規分布）: 90%=1.645, 95%=1.96, 99%=2.576
	var zScore float64
	switch confidenceLevel {
	case 0.90:
		zScore = 1.645
	case 0.95:
		zScore = 1.96
	case 0.99:
		zScore = 2.576
	default:
		zScore = 1.96 // デフォルト95%
	}

	margin := zScore * residualStdDev
	lowerBound := predictedValue - margin
	upperBound := predictedValue + margin

	// 4. 予測の信頼度を計算（R²値ベース）
	confidence := regression.RSquared

	// 5. 予測根拠を生成
	factors := []string{
		fmt.Sprintf("気温 %.1f°C に基づく回帰予測", futureTemperature),
		fmt.Sprintf("過去 %d 件のデータから学習", len(historicalSales)),
		fmt.Sprintf("決定係数 R² = %.3f", regression.RSquared),
	}

	if regression.RSquared > 0.5 {
		factors = append(factors, "気温と売上の相関が強いため、予測精度は高いです")
	} else if regression.RSquared > 0.3 {
		factors = append(factors, "気温と売上に相関がありますが、他の要因も考慮が必要です")
	} else {
		factors = append(factors, "気温以外の要因が売上に大きく影響している可能性があります")
	}

	return models.SalesPrediction{
		PredictedValue: predictedValue,
		ConfidenceInterval: models.ConfidenceInterval{
			Lower:      lowerBound,
			Upper:      upperBound,
			Confidence: confidenceLevel,
		},
		Confidence:         confidence,
		PredictionFactors:  factors,
		RegressionEquation: fmt.Sprintf("y = %.2fx + %.2f", regression.Slope, regression.Intercept),
	}, nil
}

// ForecastProductDemand 製品別の需要予測を実行
func (s *StatisticsService) ForecastProductDemand(
	productID string,
	productName string,
	historicalData []models.SalesDataPoint,
	period string,
	regionCode string,
) (models.ProductForecast, error) {
	if len(historicalData) < 14 {
		return models.ProductForecast{}, fmt.Errorf("予測には最低14日分のデータが必要です")
	}

	// 期間の日数を決定
	var forecastDays int
	switch period {
	case "week":
		forecastDays = 7
	case "2weeks":
		forecastDays = 14
	case "month":
		forecastDays = 30
	default:
		forecastDays = 7
	}

	// 統計情報を計算
	stats := s.calculateProductStatistics(historicalData)

	// 曜日効果を計算
	weekdayEffect := s.calculateWeekdayEffect(historicalData)

	// 気温との相関を計算
	var temperatures, sales []float64
	for _, point := range historicalData {
		if point.Temperature > 0 {
			temperatures = append(temperatures, point.Temperature)
			sales = append(sales, point.Sales)
		}
	}

	var regression *models.RegressionResult
	var err error
	if len(temperatures) >= 10 {
		regression, err = s.PerformLinearRegression(temperatures, sales)
		if err != nil {
			log.Printf("回帰分析エラー: %v", err)
		}
	}

	// 将来の予測日を生成
	lastDate, _ := time.Parse("2006-01-02", historicalData[len(historicalData)-1].Date)
	var dailyForecasts []models.DailyForecast
	var totalForecast float64

	for i := 1; i <= forecastDays; i++ {
		forecastDate := lastDate.AddDate(0, 0, i)
		dayOfWeek := s.getDayOfWeekJP(forecastDate.Weekday())

		// 基準値: 全体平均
		baseValue := stats.Mean

		// 曜日効果を適用
		if effect, ok := weekdayEffect[dayOfWeek]; ok {
			baseValue = baseValue * effect
		}

		// 気温効果を適用（回帰モデルがある場合）
		if regression != nil && regression.RSquared > 0.1 {
			// 簡易的に季節の平均気温を使用
			seasonalTemp := s.getSeasonalTemperature(forecastDate.Month())
			tempAdjustment := regression.Slope * (seasonalTemp - calculateMean(temperatures))
			baseValue += tempAdjustment
		}

		// トレンド効果（単純移動平均の傾き）
		trendAdjustment := s.calculateTrend(historicalData) * float64(i)
		baseValue += trendAdjustment

		dailyForecasts = append(dailyForecasts, models.DailyForecast{
			Date:           forecastDate.Format("2006-01-02"),
			DayOfWeek:      dayOfWeek,
			PredictedValue: math.Max(0, baseValue), // 負の値を避ける
			Temperature:    s.getSeasonalTemperature(forecastDate.Month()),
		})

		totalForecast += baseValue
	}

	// 信頼区間を計算
	stdDev := stats.StdDev
	zScore := 1.96 // 95% confidence
	marginTotal := zScore * stdDev * math.Sqrt(float64(forecastDays))

	confidence := 0.5 // デフォルト
	if regression != nil {
		confidence = regression.RSquared
	}

	// 期間の範囲を文字列化
	startForecast := dailyForecasts[0].Date
	endForecast := dailyForecasts[len(dailyForecasts)-1].Date
	forecastPeriod := fmt.Sprintf("%s 〜 %s", startForecast, endForecast)

	// 推奨事項を生成
	recommendations := s.generateForecastRecommendations(totalForecast, stats, period)

	// 季節性の判定
	seasonality := s.detectSeasonality(historicalData)

	return models.ProductForecast{
		ProductID:      productID,
		ProductName:    productName,
		ForecastPeriod: forecastPeriod,
		PredictedTotal: math.Max(0, totalForecast),
		DailyAverage:   math.Max(0, totalForecast/float64(forecastDays)),
		ConfidenceInterval: models.ConfidenceInterval{
			Lower:      math.Max(0, totalForecast-marginTotal),
			Upper:      totalForecast + marginTotal,
			Confidence: 0.95,
		},
		Confidence:      confidence,
		DailyBreakdown:  dailyForecasts,
		Factors:         s.buildFactorsList(regression, weekdayEffect, stats),
		Seasonality:     seasonality,
		Recommendations: recommendations,
	}, nil
}

// calculateProductStatistics 製品の統計情報を計算
func (s *StatisticsService) calculateProductStatistics(data []models.SalesDataPoint) models.ProductStatistics {
	var sales []float64
	weekdaySales := make(map[string][]float64)
	monthlySales := make(map[string][]float64)

	for _, point := range data {
		sales = append(sales, point.Sales)
		if point.DayOfWeek != "" {
			weekdaySales[point.DayOfWeek] = append(weekdaySales[point.DayOfWeek], point.Sales)
		}
		if t, err := time.Parse("2006-01-02", point.Date); err == nil {
			month := fmt.Sprintf("%d月", int(t.Month()))
			monthlySales[month] = append(monthlySales[month], point.Sales)
		}
	}

	mean := calculateMean(sales)
	stdDev := calculateStandardDeviation(sales)

	// 曜日別平均
	weekdayAvg := make(map[string]float64)
	for day, values := range weekdaySales {
		weekdayAvg[day] = calculateMean(values)
	}

	// 月別平均
	monthlyAvg := make(map[string]float64)
	for month, values := range monthlySales {
		monthlyAvg[month] = calculateMean(values)
	}

	// トレンド方向を判定
	trend := s.calculateTrend(data)
	var trendDirection string
	if trend > 0.5 {
		trendDirection = "増加"
	} else if trend < -0.5 {
		trendDirection = "減少"
	} else {
		trendDirection = "安定"
	}

	sortedSales := make([]float64, len(sales))
	copy(sortedSales, sales)
	sort.Float64s(sortedSales)

	median := sortedSales[len(sortedSales)/2]
	min := sortedSales[0]
	max := sortedSales[len(sortedSales)-1]

	return models.ProductStatistics{
		Mean:           mean,
		Median:         median,
		StdDev:         stdDev,
		Min:            min,
		Max:            max,
		WeekdayAverage: weekdayAvg,
		MonthlyAverage: monthlyAvg,
		TrendDirection: trendDirection,
	}
}

// calculateWeekdayEffect 曜日効果を計算（全体平均に対する比率）
func (s *StatisticsService) calculateWeekdayEffect(data []models.SalesDataPoint) map[string]float64 {
	weekdaySales := make(map[string][]float64)
	var allSales []float64

	for _, point := range data {
		allSales = append(allSales, point.Sales)
		if point.DayOfWeek != "" {
			weekdaySales[point.DayOfWeek] = append(weekdaySales[point.DayOfWeek], point.Sales)
		}
	}

	overallMean := calculateMean(allSales)
	effect := make(map[string]float64)

	for day, sales := range weekdaySales {
		dayMean := calculateMean(sales)
		effect[day] = dayMean / overallMean // 1.0が平均、>1.0が平均以上
	}

	return effect
}

// calculateTrend 単純なトレンドを計算（1日あたりの変化量）
func (s *StatisticsService) calculateTrend(data []models.SalesDataPoint) float64 {
	if len(data) < 2 {
		return 0
	}

	// 最初の1/3と最後の1/3の平均を比較
	n := len(data)
	firstThird := n / 3
	var earlySum, lateSum float64

	for i := 0; i < firstThird; i++ {
		earlySum += data[i].Sales
	}
	for i := n - firstThird; i < n; i++ {
		lateSum += data[i].Sales
	}

	earlyAvg := earlySum / float64(firstThird)
	lateAvg := lateSum / float64(firstThird)

	// 1日あたりの変化量
	return (lateAvg - earlyAvg) / float64(n-firstThird)
}

// getSeasonalTemperature 月ごとの平均気温を返す（簡易版）
func (s *StatisticsService) getSeasonalTemperature(month time.Month) float64 {
	temps := map[time.Month]float64{
		time.January:   5.0,
		time.February:  6.0,
		time.March:     10.0,
		time.April:     15.0,
		time.May:       20.0,
		time.June:      24.0,
		time.July:      28.0,
		time.August:    29.0,
		time.September: 25.0,
		time.October:   19.0,
		time.November:  13.0,
		time.December:  7.0,
	}
	return temps[month]
}

// getDayOfWeekJP 曜日を日本語で返す
func (s *StatisticsService) getDayOfWeekJP(weekday time.Weekday) string {
	days := []string{"日", "月", "火", "水", "木", "金", "土"}
	return days[int(weekday)]
}

// detectSeasonality 季節性を検出
func (s *StatisticsService) detectSeasonality(data []models.SalesDataPoint) string {
	if len(data) < 30 {
		return ""
	}

	monthlySales := make(map[int][]float64)
	for _, point := range data {
		if t, err := time.Parse("2006-01-02", point.Date); err == nil {
			month := int(t.Month())
			monthlySales[month] = append(monthlySales[month], point.Sales)
		}
	}

	// 夏季(6-8月)と冬季(12-2月)の平均を比較
	var summerSum, winterSum float64
	var summerCount, winterCount int

	for month, sales := range monthlySales {
		avg := calculateMean(sales)
		if month >= 6 && month <= 8 {
			summerSum += avg
			summerCount++
		} else if month == 12 || month <= 2 {
			winterSum += avg
			winterCount++
		}
	}

	if summerCount > 0 && winterCount > 0 {
		summerAvg := summerSum / float64(summerCount)
		winterAvg := winterSum / float64(winterCount)

		diff := (summerAvg - winterAvg) / winterAvg * 100
		if diff > 20 {
			return fmt.Sprintf("夏季需要が高い傾向（冬季比 +%.0f%%）", diff)
		} else if diff < -20 {
			return fmt.Sprintf("冬季需要が高い傾向（夏季比 +%.0f%%）", -diff)
		}
	}

	return "明確な季節性は検出されませんでした"
}

// generateForecastRecommendations 予測に基づく推奨事項を生成
func (s *StatisticsService) generateForecastRecommendations(forecast float64, stats models.ProductStatistics, period string) []string {
	var recommendations []string

	// 需要レベルに基づく推奨
	if forecast > stats.Mean*1.2 {
		recommendations = append(recommendations, fmt.Sprintf("予測需要が平均より高いです。十分な在庫を確保してください（予測: %.0f, 平均: %.0f）", forecast, stats.Mean))
	} else if forecast < stats.Mean*0.8 {
		recommendations = append(recommendations, "予測需要が平均より低いです。過剰在庫に注意してください")
	}

	// 曜日効果に基づく推奨
	if len(stats.WeekdayAverage) > 0 {
		var maxDay string
		var maxValue float64
		for day, avg := range stats.WeekdayAverage {
			if avg > maxValue {
				maxValue = avg
				maxDay = day
			}
		}
		if maxDay != "" {
			recommendations = append(recommendations, fmt.Sprintf("%s曜日の需要が最も高い傾向があります", maxDay))
		}
	}

	// トレンドに基づく推奨
	switch stats.TrendDirection {
	case "増加":
		recommendations = append(recommendations, "需要増加トレンドが見られます。供給体制の強化を検討してください")
	case "減少":
		recommendations = append(recommendations, "需要減少トレンドが見られます。マーケティング施策の見直しを推奨します")
	}

	return recommendations
}

// buildFactorsList 予測に使用した要因リストを生成
func (s *StatisticsService) buildFactorsList(regression *models.RegressionResult, weekdayEffect map[string]float64, stats models.ProductStatistics) []string {
	factors := []string{
		fmt.Sprintf("過去の販売実績（平均: %.0f個/日）", stats.Mean),
		fmt.Sprintf("トレンド方向: %s", stats.TrendDirection),
	}

	if len(weekdayEffect) > 0 {
		factors = append(factors, "曜日による需要変動を考慮")
	}

	if regression != nil && regression.RSquared > 0.1 {
		factors = append(factors, fmt.Sprintf("気温との相関（R² = %.2f）", regression.RSquared))
	}

	factors = append(factors, "季節性パターンを分析")

	return factors
}
