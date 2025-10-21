package services

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"hunt-chat-api/pkg/models"

	"github.com/google/uuid"
)

// StatisticsService 統計分析サービス
type StatisticsService struct {
	weatherService     *WeatherService
	azureOpenAIService *AzureOpenAIService
}

// NewStatisticsService 新しい統計分析サービスを作成
func NewStatisticsService(weatherService *WeatherService, azureOpenAIService *AzureOpenAIService) *StatisticsService {
	return &StatisticsService{
		weatherService:     weatherService,
		azureOpenAIService: azureOpenAIService,
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

		// 日付形式の診断ログを追加
		if len(salesData) > 0 {
			log.Printf("🔍 [診断] 販売データの日付例: '%s'", salesData[0].Date)
		}
		if len(weatherData) > 0 {
			log.Printf("🔍 [診断] 気象データの日付例: '%s'", weatherData[0].Date)
		}

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
		ReportID:        uuid.New().String(),
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
		Confidence:         confidence,
		PredictionFactors:  factors,
		RegressionEquation: fmt.Sprintf("y = %.2fx + %.2f", regression.Slope, regression.Intercept),
	}, nil
}

// DetectAnomalies 売上データから異常値を検出する（移動平均乖離率法）
// granularity: "daily", "weekly", "monthly" - データ集約粒度（デフォルト: "weekly"）
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
		mean := s.calculateMean(window)

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
			stdDev := s.calculateStandardDeviation(window)
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
// 例: "2022-04" → "2022年4月"
//     "2022-W10" → "2022年 第10週"
//     "2022-03-07" → "2022年3月7日"
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
	// 製品の表示名を決定（製品名があればそれを使用、なければID）
	displayName := anomaly.ProductName
	if displayName == "" {
		displayName = anomaly.ProductID
	}
	
	// 日付を読みやすい形式にフォーマット
	formattedDate := s.formatDateForDisplay(anomaly.Date)
	
	// AIサービスが利用可能な場合は、AIに質問と選択肢を生成させる
	if s.azureOpenAIService != nil {
		// AnomalyDetectionをAnomalyに変換（必要なフィールドのみ）
		anomalyForAI := models.Anomaly{
			Date:        formattedDate, // フォーマット済みの日付を使用
			ProductID:   displayName,    // 表示名を使用
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
			tempAdjustment := regression.Slope * (seasonalTemp - s.calculateMean(temperatures))
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

	mean := s.calculateMean(sales)
	stdDev := s.calculateStandardDeviation(sales)

	// 曜日別平均
	weekdayAvg := make(map[string]float64)
	for day, values := range weekdaySales {
		weekdayAvg[day] = s.calculateMean(values)
	}

	// 月別平均
	monthlyAvg := make(map[string]float64)
	for month, values := range monthlySales {
		monthlyAvg[month] = s.calculateMean(values)
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

	overallMean := s.calculateMean(allSales)
	effect := make(map[string]float64)

	for day, sales := range weekdaySales {
		dayMean := s.calculateMean(sales)
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
		avg := s.calculateMean(sales)
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
