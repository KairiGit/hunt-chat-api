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
	economicService    *EconomicService
	azureOpenAIService *AzureOpenAIService
}

// NewStatisticsService 新しい統計分析サービスを作成
func NewStatisticsService(weatherService *WeatherService, economicService *EconomicService, azureOpenAIService *AzureOpenAIService) *StatisticsService {
	return &StatisticsService{
		weatherService:     weatherService,
		economicService:    economicService,
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

// CalculateLaggedCorrelations x(t) vs y(t+lag) for lag in [-maxLagDays, +maxLagDays].
// Returns a sorted list by absolute correlation desc.
func (s *StatisticsService) CalculateLaggedCorrelations(xDates []string, xVals []float64, yDates []string, yVals []float64, maxLagDays int) ([]models.CorrelationResult, error) {
	if len(xDates) != len(xVals) || len(yDates) != len(yVals) {
		return nil, fmt.Errorf("データ系列の長さが一致しません")
	}
	if len(xVals) < 5 || len(yVals) < 5 {
		return nil, fmt.Errorf("データ点が不足しています（最低5点）")
	}
	// Build map for fast lookup
	xMap := make(map[string]float64, len(xDates))
	for i, d := range xDates {
		xMap[d] = xVals[i]
	}
	yMap := make(map[string]float64, len(yDates))
	for i, d := range yDates {
		yMap[d] = yVals[i]
	}

	// Helper to shift dates by lag days
	shift := func(date string, lag int) (string, bool) {
		t, err := time.Parse("2006-01-02", date)
		if err != nil {
			return "", false
		}
		return t.AddDate(0, 0, lag).Format("2006-01-02"), true
	}

	var results []models.CorrelationResult
	for lag := -maxLagDays; lag <= maxLagDays; lag++ {
		var xs, ys []float64
		// align on x dates
		for _, d := range xDates {
			if d == "" {
				continue
			}
			shifted, ok := shift(d, lag)
			if !ok {
				continue
			}
			xv, okx := xMap[d]
			yv, oky := yMap[shifted]
			if okx && oky {
				xs = append(xs, xv)
				ys = append(ys, yv)
			}
		}
		if len(xs) >= 5 && len(xs) == len(ys) {
			r, err := s.CalculateCorrelation(xs, ys)
			if err == nil {
				p := s.CalculatePValue(r, len(xs))
				label := "lag=0"
				if lag > 0 {
					label = fmt.Sprintf("yがxに対して+%d日遅れ", lag)
				}
				if lag < 0 {
					label = fmt.Sprintf("yがxに対して%d日先行", -lag)
				}
				results = append(results, models.CorrelationResult{
					Factor:          label,
					CorrelationCoef: r,
					PValue:          p,
					SampleSize:      len(xs),
					Interpretation:  s.InterpretCorrelation(r, p),
				})
			}
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return math.Abs(results[i].CorrelationCoef) > math.Abs(results[j].CorrelationCoef)
	})
	return results, nil
}

// CalculatePValue 相関係数のp値を近似計算（簡易版）
func (s *StatisticsService) CalculatePValue(r float64, n int) float64 {
	if n < 3 {
		return 1.0
	}
	t := r * math.Sqrt(float64(n-2)) / math.Sqrt(1-r*r)
	df := float64(n - 2)
	// Use Student's t CDF for two-tailed p-value
	p := 2 * (1 - studentTCDF(math.Abs(t), df))
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	return p
}

// CalculateLaggedCorrelationsWindowed runs lag scan over sliding windows across time.
// windows of size windowDays, step stepDays, returns best lag per window including p and BH-corrected p over lags.
func (s *StatisticsService) CalculateLaggedCorrelationsWindowed(xDates []string, xVals []float64, yDates []string, yVals []float64, maxLagDays int, windowDays int, stepDays int) ([]map[string]interface{}, error) {
	if len(xDates) != len(xVals) || len(yDates) != len(yVals) {
		return nil, fmt.Errorf("データ系列の長さが一致しません")
	}
	if windowDays < 7 {
		return nil, fmt.Errorf("windowDaysは7以上を指定してください")
	}
	// map for lookup
	xMap := map[string]float64{}
	for i, d := range xDates {
		xMap[d] = xVals[i]
	}
	yMap := map[string]float64{}
	for i, d := range yDates {
		yMap[d] = yVals[i]
	}

	// window loop over x timeline
	var results []map[string]interface{}
	// convert xDates to times
	var times []time.Time
	for _, d := range xDates {
		if t, err := time.Parse("2006-01-02", d); err == nil {
			times = append(times, t)
		}
	}
	if len(times) == 0 {
		return nil, fmt.Errorf("xDatesの形式が不正です")
	}
	// ensure sorted
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	startIdx := 0
	for {
		winStart := times[startIdx]
		winEnd := winStart.AddDate(0, 0, windowDays-1)
		// collect window dates
		var wxDates []string
		for _, t := range times {
			if t.Before(winStart) || t.After(winEnd) {
				continue
			}
			wxDates = append(wxDates, t.Format("2006-01-02"))
		}
		if len(wxDates) < 5 {
			break
		}
		// sweep lags
		type lr struct {
			lag int
			r   float64
			p   float64
			n   int
		}
		var all []lr
		for lag := -maxLagDays; lag <= maxLagDays; lag++ {
			var xs, ys []float64
			for _, d := range wxDates {
				t, _ := time.Parse("2006-01-02", d)
				sd := t.AddDate(0, 0, lag).Format("2006-01-02")
				xv, okx := xMap[d]
				yv, oky := yMap[sd]
				if okx && oky {
					xs = append(xs, xv)
					ys = append(ys, yv)
				}
			}
			if len(xs) >= 5 && len(xs) == len(ys) {
				r, err := s.CalculateCorrelation(xs, ys)
				if err == nil {
					p := s.CalculatePValue(r, len(xs))
					all = append(all, lr{lag: lag, r: r, p: p, n: len(xs)})
				}
			}
		}
		if len(all) > 0 {
			// BH correction across lags in the window
			pvals := make([]float64, len(all))
			for i, a := range all {
				pvals[i] = a.p
			}
			padj := s.AdjustPValuesBH(pvals)
			// pick best by |r|
			bestIdx := 0
			for i := 1; i < len(all); i++ {
				if math.Abs(all[i].r) > math.Abs(all[bestIdx].r) {
					bestIdx = i
				}
			}
			res := map[string]interface{}{
				"window_start": winStart.Format("2006-01-02"),
				"window_end":   winEnd.Format("2006-01-02"),
				"best_lag":     all[bestIdx].lag,
				"r":            all[bestIdx].r,
				"p":            all[bestIdx].p,
				"p_adj":        padj[bestIdx],
				"n":            all[bestIdx].n,
			}
			results = append(results, res)
		}
		// advance window
		winNext := winStart.AddDate(0, 0, stepDays)
		// find index closest to winNext
		nextIdx := -1
		for i, t := range times {
			if !t.Before(winNext) {
				nextIdx = i
				break
			}
		}
		if nextIdx == -1 || nextIdx == startIdx {
			break
		}
		startIdx = nextIdx
	}
	return results, nil
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

// PerformLinearRegression 線形回帰分析を実行
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

// SimpleGrangerCausality performs a simplified Granger causality test: does x help predict y?
// Returns F-stat and p-value.
func (s *StatisticsService) SimpleGrangerCausality(y []float64, x []float64, order int) (float64, float64, error) {
	T := len(y)
	if T != len(x) || T < 2*order+10 {
		return 0, 1, fmt.Errorf("データ長不足またはorder過大")
	}
	// Restricted model: AR(order) for y
	// Build design matrix for y: y[t] = c + a1*y[t-1] + ... + aP*y[t-P] + e
	// Full model: y[t] = c + a1*y[t-1]+...+aP*y[t-P] + b1*x[t-1]+...+bP*x[t-P] + e
	// We'll fit both and compare RSS.

	// For simplicity, we'll use a direct approach with normal equations, not intercept for AR (de-mean first).
	// De-mean
	my := 0.0
	mx := 0.0
	for i := 0; i < T; i++ {
		my += y[i]
		mx += x[i]
	}
	my /= float64(T)
	mx /= float64(T)
	Y := make([]float64, T)
	X := make([]float64, T)
	for i := 0; i < T; i++ {
		Y[i] = y[i] - my
		X[i] = x[i] - mx
	}
	// Build restricted model (only y lags)
	// rows: t=order..T-1
	n := T - order
	Xr := make([][]float64, n)
	for i := 0; i < n; i++ {
		Xr[i] = make([]float64, order)
		for lag := 0; lag < order; lag++ {
			Xr[i][lag] = Y[order-1-lag+i]
		}
	}
	YRestr := Y[order:T]
	// Build full model (y lags + x lags)
	Xf := make([][]float64, n)
	for i := 0; i < n; i++ {
		Xf[i] = make([]float64, 2*order)
		for lag := 0; lag < order; lag++ {
			Xf[i][lag] = Y[order-1-lag+i]
		}
		for lag := 0; lag < order; lag++ {
			Xf[i][order+lag] = X[order-1-lag+i]
		}
	}
	rssR, err := olsRSS(YRestr, Xr)
	if err != nil {
		return 0, 1, err
	}
	rssF, err := olsRSS(YRestr, Xf)
	if err != nil {
		return 0, 1, err
	}
	// F-test
	df1 := float64(order)
	df2 := float64(T - 2*order)
	if df2 <= 0 {
		return 0, 1, fmt.Errorf("自由度不足")
	}
	F := ((rssR - rssF) / df1) / (rssF / df2)
	// Approximate p-value using F distribution tail via incomplete beta (relation with Beta)
	p := fDistSurvival(F, df1, df2)
	return F, p, nil
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

	// 相関分析（天気データ）
	weatherCorrelations, err := s.AnalyzeSalesWeatherCorrelation(salesData, regionCode)
	if err != nil {
		weatherCorrelations = []models.CorrelationResult{} // エラーでも空配列で継続
	}

	// 相関分析（経済データ）- 遅れ相関も含む
	economicCorrelations, err := s.AnalyzeSalesEconomicCorrelation(salesData, []string{"NIKKEI", "USDJPY", "WTI"}, 30)
	if err != nil {
		log.Printf("⚠️ 経済データ相関分析エラー: %v", err)
		economicCorrelations = []models.CorrelationResult{}
	}

	// 天気と経済の相関結果を結合
	correlations := append(weatherCorrelations, economicCorrelations...)

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

	// 相関分析からのレコメンデーション（天気データ）
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
			// 経済データの相関
			if strings.Contains(corr.Factor, "NIKKEI") {
				if corr.CorrelationCoef > 0 {
					recommendations = append(recommendations, fmt.Sprintf("日経平均との正の相関が検出されました（相関係数: %.2f）。株価動向を需要予測に活用できる可能性があります。", corr.CorrelationCoef))
				} else {
					recommendations = append(recommendations, fmt.Sprintf("日経平均との負の相関が検出されました（相関係数: %.2f）。景気後退期に需要が増加する製品特性が示唆されます。", corr.CorrelationCoef))
				}
			}
			if strings.Contains(corr.Factor, "USDJPY") {
				recommendations = append(recommendations, fmt.Sprintf("為替レート（USD/JPY）との相関が検出されました（相関係数: %.2f）。輸入原材料や外国人観光客需要の影響を考慮してください。", corr.CorrelationCoef))
			}
			if strings.Contains(corr.Factor, "WTI") {
				recommendations = append(recommendations, fmt.Sprintf("原油価格との相関が検出されました（相関係数: %.2f）。輸送コストや消費者心理への影響を監視してください。", corr.CorrelationCoef))
			}
		}
	}

	// 遅れ相関の検出
	for _, corr := range correlations {
		if strings.Contains(corr.Factor, "遅れ") || strings.Contains(corr.Factor, "先行") {
			if math.Abs(corr.CorrelationCoef) > 0.4 && corr.PValue < 0.05 {
				recommendations = append(recommendations, fmt.Sprintf("⏱️ タイムラグが検出されました: %s（相関係数: %.2f）。先行指標として活用できます。", corr.Factor, corr.CorrelationCoef))
			}
		}
	}

	// 回帰分析からのレコメンデーション
	if regression != nil && regression.RSquared > 0.3 {
		recommendations = append(recommendations, fmt.Sprintf("回帰モデルの精度は%.1f%%です。気象データを使った需要予測が有効です。", regression.RSquared*100))
	}

	// 相関が見つからなかった場合
	if len(correlations) == 0 {
		recommendations = append(recommendations, "⚠️ 販売データの日付と外部データがマッチしませんでした。日付形式を確認してください（YYYY-MM-DD形式を推奨）。")
		recommendations = append(recommendations, "現在のデータは模擬データです。実際のデータ期間との整合性を確認してください。")
	}

	// デフォルトのレコメンデーション
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "さらなるデータ蓄積により、より精度の高い分析が可能になります。")
		recommendations = append(recommendations, "季節性や曜日効果も考慮した多変量解析を検討してください。")
	}

	return recommendations
}
