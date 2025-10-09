package services

import (
	"fmt"
	"hunt-chat-api/pkg/models"
	"math"
	"time"
)

// DemandForecastService 需要予測サービス
type DemandForecastService struct {
	weatherService *WeatherService
}

// NewDemandForecastService 新しい需要予測サービスを作成
func NewDemandForecastService(weatherService *WeatherService) *DemandForecastService {
	return &DemandForecastService{
		weatherService: weatherService,
	}
}

// loadSalesData 模擬的な販売実績データを読み込む
func (dfs *DemandForecastService) loadSalesData(days int) []models.SalesRecord {
	// 本来はデータベースやCSVから読み込むが、ここでは模擬データを生成
	var records []models.SalesRecord
	baseSales := 100
	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -days+i)
		sales := baseSales + (i%10)*10 // 通常の変動
		// 異常な売上を意図的に挿入
		if i == days-5 { // 5日前に猛暑日で売上急増
			sales = 300
		}
		if i == days-10 { // 10日前に雨で売上減
			sales = 50
		}

		records = append(records, models.SalesRecord{
			Date:          date.Format("2006-01-02"),
			ProductID:     "P001",
			ProductName:   "ミネラルウォーター",
			SalesQuantity: sales,
			SalesAmount:   float64(sales) * 120.0,
			Region:        "240000",
		})
	}
	return records
}

// DetectAnomalies 販売データと気象データから異常を検知する
func (dfs *DemandForecastService) DetectAnomalies(regionCode string, days int) ([]models.Anomaly, error) {
	// 1. 販売実績データを読み込む
	salesRecords := dfs.loadSalesData(days)

	// 2. 対応する期間の気象データを取得
	weatherData, err := dfs.weatherService.GetHistoricalWeatherDataByRange(regionCode, days)
	if err != nil {
		return nil, fmt.Errorf("気象データの取得に失敗: %w", err)
	}

	// 気象データを日付で検索できるようにマップに変換
	weatherMap := make(map[string]HistoricalWeatherData)
	for _, w := range weatherData {
		weatherMap[w.Date] = w
	}

	// 3. 異常検知ロジック
	var anomalies []models.Anomaly
	avgSales := 0
	for _, s := range salesRecords {
		avgSales += s.SalesQuantity
	}
	avgSales /= len(salesRecords)

	for _, sale := range salesRecords {
		weather, ok := weatherMap[sale.Date]
		if !ok {
			continue // 対応する気象データがない場合はスキップ
		}

		// 検知ルール1: 売上が平均の1.5倍以上で、かつ気温が30度以上
		if float64(sale.SalesQuantity) > float64(avgSales)*1.5 && weather.Temperature >= 30.0 {
			anomaly := models.Anomaly{
				Date:        sale.Date,
				ProductID:   sale.ProductID,
				Description: fmt.Sprintf("猛暑日（%.1f℃）に売上が平均（%d個）を大幅に上回りました（%d個）。", weather.Temperature, avgSales, sale.SalesQuantity),
				ImpactScore: (float64(sale.SalesQuantity) / float64(avgSales)) * (weather.Temperature / 30.0),
				Trigger:     "weather_sales_high",
				Weather:     weather.Weather,
				Temperature: weather.Temperature,
			}
			anomalies = append(anomalies, anomaly)
		}

		// 検知ルール2: 売上が平均の半分以下で、かつ降水量が10mm以上
		if float64(sale.SalesQuantity) < float64(avgSales)*0.5 && weather.Precipitation >= 10.0 {
			anomaly := models.Anomaly{
				Date:        sale.Date,
				ProductID:   sale.ProductID,
				Description: fmt.Sprintf("大雨（%.1fmm）の日に売上が平均（%d個）を大幅に下回りました（%d個）。", weather.Precipitation, avgSales, sale.SalesQuantity),
				ImpactScore: (float64(avgSales) / float64(sale.SalesQuantity)) * (weather.Precipitation / 10.0),
				Trigger:     "weather_sales_low",
				Weather:     weather.Weather,
				Temperature: weather.Temperature,
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies, nil
}

// DemandForecastRequest 需要予測リクエスト構造体
type DemandForecastRequest struct {
	RegionCode      string               `json:"region_code"`
	ProductCategory string               `json:"product_category"`
	ForecastDays    int                  `json:"forecast_days"`
	HistoricalDays  int                  `json:"historical_days"`
	TacitKnowledge  []TacitKnowledgeItem `json:"tacit_knowledge"`
	SeasonalFactors SeasonalFactors      `json:"seasonal_factors"`
	ExternalFactors ExternalFactors      `json:"external_factors"`
}

// TacitKnowledgeItem 暗黙知項目
type TacitKnowledgeItem struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`
	Condition   string  `json:"condition"`
}

// SeasonalFactors 季節要因
type SeasonalFactors struct {
	SummerDemandIncrease float64 `json:"summer_demand_increase"`
	WinterDemandIncrease float64 `json:"winter_demand_increase"`
	RainySeasonImpact    float64 `json:"rainy_season_impact"`
	HolidayImpact        float64 `json:"holiday_impact"`
}

// ExternalFactors 外部要因
type ExternalFactors struct {
	EconomicIndex         float64 `json:"economic_index"`
	CompetitorActivity    float64 `json:"competitor_activity"`
	MarketingCampaign     float64 `json:"marketing_campaign"`
	SupplyChainDisruption float64 `json:"supply_chain_disruption"`
}

// DemandForecastResponse 需要予測結果
type DemandForecastResponse struct {
	RegionCode      string               `json:"region_code"`
	RegionName      string               `json:"region_name"`
	ProductCategory string               `json:"product_category"`
	ForecastPeriod  string               `json:"forecast_period"`
	Forecasts       []DemandForecastItem `json:"forecasts"`
	Statistics      DemandStatistics     `json:"statistics"`
	Confidence      ConfidenceMetrics    `json:"confidence"`
	Explanations    []ExplanationItem    `json:"explanations"`
	GeneratedAt     string               `json:"generated_at"`
}

// DemandForecastItem 需要予測項目
type DemandForecastItem struct {
	Date            string              `json:"date"`
	PredictedDemand float64             `json:"predicted_demand"`
	ConfidenceLevel float64             `json:"confidence_level"`
	WeatherImpact   float64             `json:"weather_impact"`
	SeasonalImpact  float64             `json:"seasonal_impact"`
	TacitImpact     float64             `json:"tacit_impact"`
	ExternalImpact  float64             `json:"external_impact"`
	WeatherData     DailyWeatherSummary `json:"weather_data"`
	Factors         []InfluencingFactor `json:"factors"`
}

// DemandStatistics 需要統計
type DemandStatistics struct {
	AverageDemand float64 `json:"average_demand"`
	MaxDemand     float64 `json:"max_demand"`
	MinDemand     float64 `json:"min_demand"`
	StandardDev   float64 `json:"standard_deviation"`
	TotalDemand   float64 `json:"total_demand"`
	GrowthRate    float64 `json:"growth_rate"`
	Volatility    float64 `json:"volatility"`
}

// ConfidenceMetrics 信頼度メトリクス
type ConfidenceMetrics struct {
	OverallConfidence  float64 `json:"overall_confidence"`
	WeatherConfidence  float64 `json:"weather_confidence"`
	SeasonalConfidence float64 `json:"seasonal_confidence"`
	TacitConfidence    float64 `json:"tacit_confidence"`
	ModelAccuracy      float64 `json:"model_accuracy"`
}

// ExplanationItem 説明項目
type ExplanationItem struct {
	Factor      string  `json:"factor"`
	Impact      float64 `json:"impact"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"`
}

// InfluencingFactor 影響要因
type InfluencingFactor struct {
	Name   string  `json:"name"`
	Impact float64 `json:"impact"`
	Weight float64 `json:"weight"`
}

// PredictDemand 需要予測を実行
func (dfs *DemandForecastService) PredictDemand(request DemandForecastRequest) (*DemandForecastResponse, error) {
	// 1. 過去の気象データを取得
	historicalData, err := dfs.weatherService.GetHistoricalWeatherDataByRange(request.RegionCode, request.HistoricalDays)
	if err != nil {
		return nil, fmt.Errorf("過去データ取得エラー: %w", err)
	}

	// 2. 予報データを取得
	forecastData, err := dfs.weatherService.GetForecastData(request.RegionCode)
	if err != nil {
		return nil, fmt.Errorf("予報データ取得エラー: %w", err)
	}

	// 3. 需要予測を計算
	forecasts, err := dfs.calculateDemandForecasts(request, historicalData, forecastData)
	if err != nil {
		return nil, fmt.Errorf("需要予測計算エラー: %w", err)
	}

	// 4. 統計とメトリクスを計算
	statistics := dfs.calculateStatistics(forecasts)
	confidence := dfs.calculateConfidence(request, historicalData, forecasts)
	explanations := dfs.generateExplanations(request, forecasts)

	response := &DemandForecastResponse{
		RegionCode:      request.RegionCode,
		RegionName:      dfs.weatherService.getRegionName(request.RegionCode),
		ProductCategory: request.ProductCategory,
		ForecastPeriod:  fmt.Sprintf("%d日間", request.ForecastDays),
		Forecasts:       forecasts,
		Statistics:      statistics,
		Confidence:      confidence,
		Explanations:    explanations,
		GeneratedAt:     time.Now().Format("2006-01-02 15:04:05"),
	}

	return response, nil
}

// calculateDemandForecasts 需要予測を計算
func (dfs *DemandForecastService) calculateDemandForecasts(
	request DemandForecastRequest,
	historicalData []HistoricalWeatherData,
	forecastData []JMAForecastData,
) ([]DemandForecastItem, error) {
	var forecasts []DemandForecastItem

	// 基準需要を計算（過去データから推定）
	baseDemand := dfs.calculateBaseDemand(request.ProductCategory, historicalData)

	// 予測日数分のデータを生成
	for i := 0; i < request.ForecastDays; i++ {
		forecastDate := time.Now().AddDate(0, 0, i+1)

		// 気象影響を計算
		weatherImpact := dfs.calculateWeatherImpact(request.ProductCategory, forecastDate, forecastData)

		// 季節影響を計算
		seasonalImpact := dfs.calculateSeasonalImpact(request.ProductCategory, forecastDate, request.SeasonalFactors)

		// 暗黙知影響を計算
		tacitImpact := dfs.calculateTacitKnowledgeImpact(request.TacitKnowledge, forecastDate)

		// 外部要因影響を計算
		externalImpact := dfs.calculateExternalFactorImpact(request.ExternalFactors, forecastDate)

		// 総合需要を計算
		totalDemand := baseDemand * (1 + weatherImpact + seasonalImpact + tacitImpact + externalImpact)

		// 信頼度を計算
		confidence := dfs.calculateItemConfidence(weatherImpact, seasonalImpact, tacitImpact, externalImpact)

		// 気象データを取得（簡略化）
		weatherData := DailyWeatherSummary{
			Date:    forecastDate.Format("2006-01-02"),
			AvgTemp: 25.0 + float64(i)*0.5, // 簡略化
			Weather: "晴れ",
		}

		// 影響要因を構築
		factors := []InfluencingFactor{
			{Name: "気象", Impact: weatherImpact, Weight: 0.3},
			{Name: "季節", Impact: seasonalImpact, Weight: 0.25},
			{Name: "暗黙知", Impact: tacitImpact, Weight: 0.25},
			{Name: "外部要因", Impact: externalImpact, Weight: 0.2},
		}

		forecast := DemandForecastItem{
			Date:            forecastDate.Format("2006-01-02"),
			PredictedDemand: totalDemand,
			ConfidenceLevel: confidence,
			WeatherImpact:   weatherImpact,
			SeasonalImpact:  seasonalImpact,
			TacitImpact:     tacitImpact,
			ExternalImpact:  externalImpact,
			WeatherData:     weatherData,
			Factors:         factors,
		}

		forecasts = append(forecasts, forecast)
	}

	return forecasts, nil
}

// calculateBaseDemand 基準需要を計算
func (dfs *DemandForecastService) calculateBaseDemand(productCategory string, historicalData []HistoricalWeatherData) float64 {
	// 製品カテゴリに基づく基準需要（簡略化）
	baseDemands := map[string]float64{
		"飲料":     1000.0,
		"アイス":    800.0,
		"冷房器具":   600.0,
		"暖房器具":   400.0,
		"傘":      200.0,
		"レインコート": 150.0,
		"その他":    500.0,
	}

	baseDemand, exists := baseDemands[productCategory]
	if !exists {
		baseDemand = 500.0 // デフォルト値
	}

	return baseDemand
}

// calculateWeatherImpact 気象影響を計算
func (dfs *DemandForecastService) calculateWeatherImpact(productCategory string, date time.Time, forecastData []JMAForecastData) float64 {
	// 製品カテゴリごとの気象影響係数（簡略化）
	weatherImpacts := map[string]map[string]float64{
		"飲料": {
			"temperature_high": 0.8,  // 高温時の需要増加
			"temperature_low":  -0.3, // 低温時の需要減少
			"sunny":            0.2,  // 晴天時の需要増加
			"rainy":            -0.1, // 雨天時の需要減少
		},
		"アイス": {
			"temperature_high": 1.2,
			"temperature_low":  -0.8,
			"sunny":            0.5,
			"rainy":            -0.3,
		},
		"傘": {
			"rainy": 2.0,  // 雨天時の大幅需要増加
			"sunny": -0.5, // 晴天時の需要減少
		},
	}

	categoryImpacts, exists := weatherImpacts[productCategory]
	if !exists {
		return 0.0 // 影響なし
	}

	// 簡略化した気象影響計算
	impact := 0.0

	// 気温影響（仮想的な気温データ）
	temperature := 25.0 + float64(date.Day()%10) // 簡略化
	if temperature > 30.0 {
		if tempImpact, exists := categoryImpacts["temperature_high"]; exists {
			impact += tempImpact
		}
	} else if temperature < 15.0 {
		if tempImpact, exists := categoryImpacts["temperature_low"]; exists {
			impact += tempImpact
		}
	}

	// 天気影響（簡略化）
	if sunnyImpact, exists := categoryImpacts["sunny"]; exists {
		impact += sunnyImpact * 0.7 // 晴天確率を70%と仮定
	}

	return impact
}

// calculateSeasonalImpact 季節影響を計算
func (dfs *DemandForecastService) calculateSeasonalImpact(productCategory string, date time.Time, factors SeasonalFactors) float64 {
	month := date.Month()
	impact := 0.0

	// 夏季影響（6-8月）
	if month >= 6 && month <= 8 {
		if productCategory == "飲料" || productCategory == "アイス" || productCategory == "冷房器具" {
			impact += factors.SummerDemandIncrease
		}
	}

	// 冬季影響（12-2月）
	if month == 12 || month <= 2 {
		if productCategory == "暖房器具" {
			impact += factors.WinterDemandIncrease
		}
	}

	// 梅雨季影響（6-7月）
	if month == 6 || month == 7 {
		if productCategory == "傘" || productCategory == "レインコート" {
			impact += factors.RainySeasonImpact
		}
	}

	return impact
}

// calculateTacitKnowledgeImpact 暗黙知影響を計算
func (dfs *DemandForecastService) calculateTacitKnowledgeImpact(tacitKnowledge []TacitKnowledgeItem, date time.Time) float64 {
	impact := 0.0

	for _, item := range tacitKnowledge {
		// 条件チェック（簡略化）
		conditionMet := dfs.checkTacitCondition(item.Condition, date)
		if conditionMet {
			impact += item.Weight
		}
	}

	return impact
}

// checkTacitCondition 暗黙知条件をチェック
func (dfs *DemandForecastService) checkTacitCondition(condition string, date time.Time) bool {
	// 簡略化した条件チェック
	switch condition {
	case "weekend":
		return date.Weekday() == time.Saturday || date.Weekday() == time.Sunday
	case "holiday":
		return dfs.isHoliday(date)
	case "hot_day":
		return date.Month() >= 6 && date.Month() <= 8
	case "cold_day":
		return date.Month() == 12 || date.Month() <= 2
	default:
		return false
	}
}

// isHoliday 祝日判定（簡略化）
func (dfs *DemandForecastService) isHoliday(date time.Time) bool {
	// 簡略化：土日を祝日として扱う
	return date.Weekday() == time.Saturday || date.Weekday() == time.Sunday
}

// calculateExternalFactorImpact 外部要因影響を計算
func (dfs *DemandForecastService) calculateExternalFactorImpact(factors ExternalFactors, date time.Time) float64 {
	impact := 0.0

	// 経済指数影響
	impact += factors.EconomicIndex * 0.1

	// 競合活動影響
	impact += factors.CompetitorActivity * 0.05

	// マーケティングキャンペーン影響
	impact += factors.MarketingCampaign * 0.15

	// サプライチェーン混乱影響
	impact += factors.SupplyChainDisruption * 0.2

	return impact
}

// calculateItemConfidence 項目信頼度を計算
func (dfs *DemandForecastService) calculateItemConfidence(weatherImpact, seasonalImpact, tacitImpact, externalImpact float64) float64 {
	// 各影響の絶対値の合計に基づく信頼度計算
	totalImpact := math.Abs(weatherImpact) + math.Abs(seasonalImpact) + math.Abs(tacitImpact) + math.Abs(externalImpact)

	// 影響が小さいほど信頼度が高い（逆相関）
	confidence := 1.0 - (totalImpact / 4.0)

	// 信頼度を0.5-0.95の範囲に制限
	if confidence < 0.5 {
		confidence = 0.5
	} else if confidence > 0.95 {
		confidence = 0.95
	}

	return confidence
}

// calculateStatistics 統計を計算
func (dfs *DemandForecastService) calculateStatistics(forecasts []DemandForecastItem) DemandStatistics {
	if len(forecasts) == 0 {
		return DemandStatistics{}
	}

	var totalDemand, maxDemand, minDemand float64
	var demands []float64

	for i, forecast := range forecasts {
		demand := forecast.PredictedDemand
		totalDemand += demand
		demands = append(demands, demand)

		if i == 0 {
			maxDemand = demand
			minDemand = demand
		} else {
			if demand > maxDemand {
				maxDemand = demand
			}
			if demand < minDemand {
				minDemand = demand
			}
		}
	}

	avgDemand := totalDemand / float64(len(forecasts))

	// 標準偏差を計算
	var variance float64
	for _, demand := range demands {
		variance += math.Pow(demand-avgDemand, 2)
	}
	variance /= float64(len(demands))
	standardDev := math.Sqrt(variance)

	// 成長率を計算（簡略化）
	growthRate := 0.0
	if len(forecasts) > 1 {
		firstDemand := forecasts[0].PredictedDemand
		lastDemand := forecasts[len(forecasts)-1].PredictedDemand
		growthRate = (lastDemand - firstDemand) / firstDemand
	}

	// ボラティリティを計算
	volatility := standardDev / avgDemand

	return DemandStatistics{
		AverageDemand: avgDemand,
		MaxDemand:     maxDemand,
		MinDemand:     minDemand,
		StandardDev:   standardDev,
		TotalDemand:   totalDemand,
		GrowthRate:    growthRate,
		Volatility:    volatility,
	}
}

// calculateConfidence 信頼度を計算
func (dfs *DemandForecastService) calculateConfidence(request DemandForecastRequest, historicalData []HistoricalWeatherData, forecasts []DemandForecastItem) ConfidenceMetrics {
	// 全体信頼度を計算
	var totalConfidence float64
	for _, forecast := range forecasts {
		totalConfidence += forecast.ConfidenceLevel
	}
	overallConfidence := totalConfidence / float64(len(forecasts))

	return ConfidenceMetrics{
		OverallConfidence:  overallConfidence,
		WeatherConfidence:  0.8,  // 気象データの信頼度
		SeasonalConfidence: 0.9,  // 季節パターンの信頼度
		TacitConfidence:    0.7,  // 暗黙知の信頼度
		ModelAccuracy:      0.85, // モデル精度
	}
}

// generateExplanations 説明を生成
func (dfs *DemandForecastService) generateExplanations(request DemandForecastRequest, forecasts []DemandForecastItem) []ExplanationItem {
	var explanations []ExplanationItem

	// 主要な影響要因を分析
	avgWeatherImpact := 0.0
	avgSeasonalImpact := 0.0
	avgTacitImpact := 0.0
	avgExternalImpact := 0.0

	for _, forecast := range forecasts {
		avgWeatherImpact += forecast.WeatherImpact
		avgSeasonalImpact += forecast.SeasonalImpact
		avgTacitImpact += forecast.TacitImpact
		avgExternalImpact += forecast.ExternalImpact
	}

	count := float64(len(forecasts))
	avgWeatherImpact /= count
	avgSeasonalImpact /= count
	avgTacitImpact /= count
	avgExternalImpact /= count

	// 説明項目を作成
	explanations = append(explanations, ExplanationItem{
		Factor:      "気象影響",
		Impact:      avgWeatherImpact,
		Description: fmt.Sprintf("気象条件が需要に%.1f%%の影響を与えています", avgWeatherImpact*100),
		Confidence:  0.8,
	})

	explanations = append(explanations, ExplanationItem{
		Factor:      "季節影響",
		Impact:      avgSeasonalImpact,
		Description: fmt.Sprintf("季節要因が需要に%.1f%%の影響を与えています", avgSeasonalImpact*100),
		Confidence:  0.9,
	})

	explanations = append(explanations, ExplanationItem{
		Factor:      "暗黙知",
		Impact:      avgTacitImpact,
		Description: fmt.Sprintf("専門知識が需要に%.1f%%の影響を与えています", avgTacitImpact*100),
		Confidence:  0.7,
	})

	explanations = append(explanations, ExplanationItem{
		Factor:      "外部要因",
		Impact:      avgExternalImpact,
		Description: fmt.Sprintf("外部要因が需要に%.1f%%の影響を与えています", avgExternalImpact*100),
		Confidence:  0.6,
	})

	return explanations
}
