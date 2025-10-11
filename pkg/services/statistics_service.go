package services

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"hunt-chat-api/pkg/models"
)

// StatisticsService 統計分析サービス
type StatisticsService struct {
	weatherService *WeatherService
}

// NewStatisticsService 新しい統計分析サービスを作成
func NewStatisticsService(weatherService *WeatherService) *StatisticsService {
	return &StatisticsService{
		weatherService: weatherService,
	}
}

// CalculateCorrelation 2つのデータ系列のピアソン相関係数を計算
func (s *StatisticsService) CalculateCorrelation(x, y []float64) (float64, error) {
	if len(x) != len(y) || len(x) == 0 {
		return 0, fmt.Errorf("データ系列の長さが一致しないか、空です")
	}

	n := float64(len(x))
	var sumX, sumY, sumXY, sumX2, sumY2 float64

	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}

	numerator := n*sumXY - sumX*sumY
	denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))

	if denominator == 0 {
		return 0, fmt.Errorf("分母が0になりました（標準偏差が0）")
	}

	return numerator / denominator, nil
}

// CalculatePValue 相関係数のp値を近似計算（簡易版）
func (s *StatisticsService) CalculatePValue(r float64, n int) float64 {
	if n < 3 {
		return 1.0 // サンプル数が少なすぎる
	}

	// t統計量の計算
	t := r * math.Sqrt(float64(n-2)) / math.Sqrt(1-r*r)

	// 自由度 n-2 のt分布を使ってp値を近似
	// 簡易版: |t| > 2.0 で有意（p < 0.05程度）
	absT := math.Abs(t)
	if absT > 2.576 {
		return 0.01 // p < 0.01
	} else if absT > 1.96 {
		return 0.05 // p < 0.05
	} else {
		return 0.10 // p > 0.05 (not significant)
	}
}

// InterpretCorrelation 相関係数を人間が読める形で解釈
func (s *StatisticsService) InterpretCorrelation(r float64, pValue float64) string {
	absR := math.Abs(r)
	strength := ""

	if absR >= 0.7 {
		strength = "強い"
	} else if absR >= 0.4 {
		strength = "中程度の"
	} else if absR >= 0.2 {
		strength = "弱い"
	} else {
		strength = "ほぼ無い"
	}

	direction := "正の"
	if r < 0 {
		direction = "負の"
	}

	significance := ""
	if pValue < 0.05 {
		significance = "（統計的に有意）"
	} else {
		significance = "（統計的に有意ではない）"
	}

	return fmt.Sprintf("%s%s相関 %s", strength, direction, significance)
}

// calculateMean 平均値を計算
func (s *StatisticsService) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculateStandardDeviation 標準偏差を計算
func (s *StatisticsService) calculateStandardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := s.calculateMean(values)
	sumSquaredDiff := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	variance := sumSquaredDiff / float64(len(values))
	return math.Sqrt(variance)
}

// AnalyzeSalesWeatherCorrelation 販売データと気象データの相関を分析
func (s *StatisticsService) AnalyzeSalesWeatherCorrelation(
	salesData []models.WeatherSalesData,
	regionCode string,
) ([]models.CorrelationResult, error) {

	if len(salesData) == 0 {
		return nil, fmt.Errorf("販売データが空です")
	}

	// 販売データの日付範囲を特定
	var startDate, endDate time.Time
	for i, data := range salesData {
		t, err := time.Parse("2006-01-02", data.Date)
		if err != nil {
			continue
		}
		if i == 0 || t.Before(startDate) {
			startDate = t
		}
		if i == 0 || t.After(endDate) {
			endDate = t
		}
	}

	// 日付範囲が特定できない場合はデフォルト（過去90日）
	if startDate.IsZero() || endDate.IsZero() {
		endDate = time.Now()
		startDate = endDate.AddDate(0, 0, -90)
	}

	// 気象データを取得（販売データの期間に合わせる）
	weatherData, err := s.weatherService.GetHistoricalWeatherData(regionCode, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("気象データの取得に失敗: %w", err)
	}

	// 日付をキーにして気象データをマップ化
	weatherMap := make(map[string]struct {
		Temperature float64
		Humidity    float64
	})
	for _, w := range weatherData {
		weatherMap[w.Date] = struct {
			Temperature float64
			Humidity    float64
		}{
			Temperature: w.Temperature,
			Humidity:    w.Humidity,
		}
	}

	// 販売データと気象データをマージ
	var temperatures, humidities, sales []float64
	for _, sale := range salesData {
		if weather, ok := weatherMap[sale.Date]; ok {
			temperatures = append(temperatures, weather.Temperature)
			humidities = append(humidities, weather.Humidity)
			sales = append(sales, sale.Sales)
		}
	}

	if len(sales) < 3 {
		return nil, fmt.Errorf("マッチするデータが少なすぎます（最低3件必要）")
	}

	var results []models.CorrelationResult

	// 気温と売上の相関
	tempCorr, err := s.CalculateCorrelation(temperatures, sales)
	if err == nil {
		pValue := s.CalculatePValue(tempCorr, len(temperatures))
		results = append(results, models.CorrelationResult{
			Factor:          "temperature",
			CorrelationCoef: tempCorr,
			PValue:          pValue,
			SampleSize:      len(temperatures),
			Interpretation:  s.InterpretCorrelation(tempCorr, pValue),
		})
	}

	// 湿度と売上の相関
	humCorr, err := s.CalculateCorrelation(humidities, sales)
	if err == nil {
		pValue := s.CalculatePValue(humCorr, len(humidities))
		results = append(results, models.CorrelationResult{
			Factor:          "humidity",
			CorrelationCoef: humCorr,
			PValue:          pValue,
			SampleSize:      len(humidities),
			Interpretation:  s.InterpretCorrelation(humCorr, pValue),
		})
	}

	return results, nil
}

// PerformLinearRegression 単回帰分析を実行
func (s *StatisticsService) PerformLinearRegression(x, y []float64) (*models.RegressionResult, error) {
	if len(x) != len(y) || len(x) < 2 {
		return nil, fmt.Errorf("データ系列の長さが一致しないか、データ数が不足しています")
	}

	n := float64(len(x))
	var sumX, sumY, sumXY, sumX2 float64

	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}

	// 傾き（slope）の計算
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// 切片（intercept）の計算
	intercept := (sumY - slope*sumX) / n

	// R²（決定係数）の計算
	meanY := sumY / n
	var ssTotal, ssResidual float64
	for i := 0; i < len(x); i++ {
		predicted := slope*x[i] + intercept
		ssTotal += (y[i] - meanY) * (y[i] - meanY)
		ssResidual += (y[i] - predicted) * (y[i] - predicted)
	}
	rSquared := 1 - (ssResidual / ssTotal)

	// 予測値の計算（最後のx値を使用）
	lastX := x[len(x)-1]
	prediction := slope*lastX + intercept

	// 信頼度の計算（R²をベースに）
	confidence := rSquared

	description := fmt.Sprintf("回帰式: y = %.2fx + %.2f (R² = %.3f)", slope, intercept, rSquared)

	return &models.RegressionResult{
		Slope:       slope,
		Intercept:   intercept,
		RSquared:    rSquared,
		Prediction:  prediction,
		Confidence:  confidence,
		Description: description,
	}, nil
}

// GenerateStatisticalSummary 統計サマリーを生成
func (s *StatisticsService) GenerateStatisticalSummary(
	salesData []models.WeatherSalesData,
	regionCode string,
) (string, error) {

	if len(salesData) == 0 {
		return "", fmt.Errorf("販売データが空です")
	}

	// 基本統計量の計算
	var totalSales, minSales, maxSales float64
	minSales = math.MaxFloat64
	maxSales = -math.MaxFloat64

	salesByDate := make(map[string]float64)
	for _, data := range salesData {
		totalSales += data.Sales
		if data.Sales < minSales {
			minSales = data.Sales
		}
		if data.Sales > maxSales {
			maxSales = data.Sales
		}
		salesByDate[data.Date] = data.Sales
	}

	avgSales := totalSales / float64(len(salesData))

	// 標準偏差の計算
	var variance float64
	for _, data := range salesData {
		diff := data.Sales - avgSales
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(len(salesData)))

	// 中央値の計算
	sortedSales := make([]float64, len(salesData))
	for i, data := range salesData {
		sortedSales[i] = data.Sales
	}
	sort.Float64s(sortedSales)
	median := sortedSales[len(sortedSales)/2]

	summary := fmt.Sprintf(`統計サマリー:
- データ点数: %d
- 平均売上: %.2f
- 中央値: %.2f
- 標準偏差: %.2f
- 最小値: %.2f
- 最大値: %.2f
- 総売上: %.2f
`,
		len(salesData),
		avgSales,
		median,
		stdDev,
		minSales,
		maxSales,
		totalSales,
	)

	return summary, nil
}

// CreateAnalysisReport 総合的な分析レポートを作成
func (s *StatisticsService) CreateAnalysisReport(
	fileName string,
	salesData []models.WeatherSalesData,
	regionCode string,
	aiInsights string,
) (*models.AnalysisReport, error) {

	// 相関分析
	correlations, err := s.AnalyzeSalesWeatherCorrelation(salesData, regionCode)
	if err != nil {
		correlations = []models.CorrelationResult{} // エラーでも空配列で継続
	}

	// 統計サマリー生成
	summary, err := s.GenerateStatisticalSummary(salesData, regionCode)
	if err != nil {
		summary = "統計サマリーの生成に失敗しました"
	}

	// 回帰分析（気温と売上）
	var regression *models.RegressionResult
	var weatherMatches int
	var dateRange string

	if len(salesData) > 0 {
		// 販売データの日付範囲を特定
		var startDate, endDate time.Time
		for i, data := range salesData {
			t, err := time.Parse("2006-01-02", data.Date)
			if err != nil {
				continue
			}
			if i == 0 || t.Before(startDate) {
				startDate = t
			}
			if i == 0 || t.After(endDate) {
				endDate = t
			}
		}

		// 日付範囲が特定できない場合はデフォルト
		if startDate.IsZero() || endDate.IsZero() {
			endDate = time.Now()
			startDate = endDate.AddDate(0, 0, -90)
		}

		dateRange = fmt.Sprintf("%s 〜 %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

		// 気温データを抽出
		var temps, sales []float64
		weatherData, err := s.weatherService.GetHistoricalWeatherData(regionCode, startDate, endDate)
		if err != nil {
			log.Printf("⚠️ 気象データ取得エラー: %v", err)
		} else {
			log.Printf("✅ 気象データ取得成功: %d件 (期間: %s 〜 %s)", len(weatherData), startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
		}

		weatherMap := make(map[string]float64)
		for _, w := range weatherData {
			weatherMap[w.Date] = w.Temperature
		}

		log.Printf("📊 販売データ件数: %d, 気象データマップサイズ: %d", len(salesData), len(weatherMap))

		for _, sale := range salesData {
			if temp, ok := weatherMap[sale.Date]; ok {
				temps = append(temps, temp)
				sales = append(sales, sale.Sales)
				weatherMatches++
			}
		}

		log.Printf("🔗 マッチング結果: %d件 / %d件", weatherMatches, len(salesData))

		if len(temps) >= 2 {
			regression, _ = s.PerformLinearRegression(temps, sales)
		}
	}

	// レコメンデーション生成
	recommendations := s.generateRecommendations(correlations, regression)

	report := &models.AnalysisReport{
		ReportID:        fmt.Sprintf("RPT-%d", time.Now().Unix()),
		FileName:        fileName,
		AnalysisDate:    time.Now().Format(time.RFC3339),
		DataPoints:      len(salesData),
		DateRange:       dateRange,
		WeatherMatches:  weatherMatches,
		Summary:         summary,
		Correlations:    correlations,
		Regression:      regression,
		AIInsights:      aiInsights,
		Recommendations: recommendations,
	}

	return report, nil
}

// generateRecommendations 分析結果に基づいてレコメンデーションを生成
func (s *StatisticsService) generateRecommendations(
	correlations []models.CorrelationResult,
	regression *models.RegressionResult,
) []string {
	var recommendations []string

	// 相関分析からのレコメンデーション
	for _, corr := range correlations {
		if math.Abs(corr.CorrelationCoef) > 0.5 && corr.PValue < 0.05 {
			if corr.Factor == "temperature" {
				if corr.CorrelationCoef > 0 {
					recommendations = append(recommendations, "気温が高いほど売上が増加する傾向があります。夏季の在庫を強化することを推奨します。")
				} else {
					recommendations = append(recommendations, "気温が低いほど売上が増加する傾向があります。冬季の在庫を強化することを推奨します。")
				}
			}
			if corr.Factor == "humidity" {
				recommendations = append(recommendations, "湿度と売上に有意な相関が見られます。天気予報と連動した在庫管理を検討してください。")
			}
		}
	}

	// 回帰分析からのレコメンデーション
	if regression != nil && regression.RSquared > 0.3 {
		recommendations = append(recommendations, fmt.Sprintf("回帰モデルの精度は%.1f%%です。気象データを使った需要予測が有効です。", regression.RSquared*100))
	}

	// 相関が見つからなかった場合
	if len(correlations) == 0 {
		recommendations = append(recommendations, "⚠️ 販売データの日付と気象データがマッチしませんでした。日付形式を確認してください（YYYY-MM-DD形式を推奨）。")
		recommendations = append(recommendations, "現在の気象データは模擬データ（過去3年分）です。実際のデータ期間との整合性を確認してください。")
	}

	// デフォルトのレコメンデーション
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "さらなるデータ蓄積により、より精度の高い分析が可能になります。")
		recommendations = append(recommendations, "季節性や曜日効果も考慮した多変量解析を検討してください。")
	}

	return recommendations
}

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

	residualStdDev := s.calculateStandardDeviation(residuals)

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
		Confidence:       confidence,
		PredictionFactors: factors,
		RegressionEquation: fmt.Sprintf("y = %.2fx + %.2f", regression.Slope, regression.Intercept),
	}, nil
}

// DetectAnomalies 売上データから異常値を検出する（3σ法）
func (s *StatisticsService) DetectAnomalies(sales []float64, dates []string) []models.AnomalyDetection {
	if len(sales) != len(dates) || len(sales) < 10 {
		return nil
	}

	mean := s.calculateMean(sales)
	stdDev := s.calculateStandardDeviation(sales)

	var anomalies []models.AnomalyDetection
	threshold := 3.0 * stdDev // 3σ

	for i, value := range sales {
		deviation := math.Abs(value - mean)
		if deviation > threshold {
			anomalyType := "急増"
			if value < mean {
				anomalyType = "急減"
			}

			zScore := (value - mean) / stdDev

			anomalies = append(anomalies, models.AnomalyDetection{
				Date:          dates[i],
				ActualValue:   value,
				ExpectedValue: mean,
				Deviation:     deviation,
				ZScore:        zScore,
				AnomalyType:   anomalyType,
				Severity:      s.calculateSeverity(math.Abs(zScore)),
			})
		}
	}

	return anomalies
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

// GenerateAIQuestion 異常値に基づいてAIが質問を生成
func (s *StatisticsService) GenerateAIQuestion(anomaly models.AnomalyDetection) string {
	if anomaly.AnomalyType == "急増" {
		return fmt.Sprintf(
			"📈 %s に売上が通常より %.0f 増加しました（期待値: %.0f → 実績: %.0f）。\n"+
				"この日に特別なイベント、キャンペーン、または外的要因はありましたか？",
			anomaly.Date,
			anomaly.Deviation,
			anomaly.ExpectedValue,
			anomaly.ActualValue,
		)
	} else {
		return fmt.Sprintf(
			"📉 %s に売上が通常より %.0f 減少しました（期待値: %.0f → 実績: %.0f）。\n"+
				"この日に売上減少の原因となった要因（天候、競合、在庫切れなど）はありましたか？",
			anomaly.Date,
			anomaly.Deviation,
			anomaly.ExpectedValue,
			anomaly.ActualValue,
		)
	}
}
