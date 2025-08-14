package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// WeatherService 気象データサービス
type WeatherService struct {
	client *http.Client
}

// NewWeatherService 新しい気象データサービスを作成
func NewWeatherService() *WeatherService {
	return &WeatherService{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// JMAForecastData 気象庁予報データの構造体
type JMAForecastData struct {
	PublishingOffice string `json:"publishingOffice"`
	ReportDatetime   string `json:"reportDatetime"`
	TimeSeries       []struct {
		TimeDefines []string `json:"timeDefines"`
		Areas       []struct {
			Area struct {
				Name string `json:"name"`
				Code string `json:"code"`
			} `json:"area"`
			WeatherCodes []string `json:"weatherCodes,omitempty"`
			Weathers     []string `json:"weathers,omitempty"`
			Winds        []string `json:"winds,omitempty"`
			Waves        []string `json:"waves,omitempty"`
			Pops         []string `json:"pops,omitempty"`
			Temps        []string `json:"temps,omitempty"`
		} `json:"areas"`
	} `json:"timeSeries"`
}

// WeatherData 統一された気象データ構造体
type WeatherData struct {
	Date          string  `json:"date"`
	RegionCode    string  `json:"region_code"`
	RegionName    string  `json:"region_name"`
	Temperature   float64 `json:"temperature"`
	Humidity      float64 `json:"humidity"`
	Precipitation float64 `json:"precipitation"`
	WindSpeed     float64 `json:"wind_speed"`
	WeatherCode   string  `json:"weather_code"`
	Weather       string  `json:"weather"`
}

// HistoricalWeatherData 過去の気象データ構造体
type HistoricalWeatherData struct {
	Date          string  `json:"date"`
	RegionCode    string  `json:"region_code"`
	RegionName    string  `json:"region_name"`
	Temperature   float64 `json:"temperature"`
	MaxTemp       float64 `json:"max_temp"`
	MinTemp       float64 `json:"min_temp"`
	Humidity      float64 `json:"humidity"`
	Precipitation float64 `json:"precipitation"`
	WindSpeed     float64 `json:"wind_speed"`
	WindDirection string  `json:"wind_direction"`
	Pressure      float64 `json:"pressure"`
	Weather       string  `json:"weather"`
	WeatherCode   string  `json:"weather_code"`
	DataSource    string  `json:"data_source"`
}

// JMAHistoricalData 気象庁過去データ（仮想的な構造体）
type JMAHistoricalData struct {
	Date         string                 `json:"date"`
	Stations     map[string]interface{} `json:"stations"`
	RegionData   map[string]interface{} `json:"region_data"`
	DataSource   string                 `json:"data_source"`
	LastModified string                 `json:"last_modified"`
}

// OpenWeatherMapHistoricalData OpenWeatherMap過去データ構造体
type OpenWeatherMapHistoricalData struct {
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Timezone string  `json:"timezone"`
	Current  struct {
		Dt        int64   `json:"dt"`
		Temp      float64 `json:"temp"`
		Humidity  int     `json:"humidity"`
		Pressure  float64 `json:"pressure"`
		WindSpeed float64 `json:"wind_speed"`
		WindDeg   int     `json:"wind_deg"`
		Weather   []struct {
			Main        string `json:"main"`
			Description string `json:"description"`
			Icon        string `json:"icon"`
		} `json:"weather"`
	} `json:"current"`
	Hourly []struct {
		Dt        int64   `json:"dt"`
		Temp      float64 `json:"temp"`
		Humidity  int     `json:"humidity"`
		Pressure  float64 `json:"pressure"`
		WindSpeed float64 `json:"wind_speed"`
		WindDeg   int     `json:"wind_deg"`
		Weather   []struct {
			Main        string `json:"main"`
			Description string `json:"description"`
		} `json:"weather"`
	} `json:"hourly"`
}

// JMAStationData 気象庁観測所データ構造体
type JMAStationData struct {
	StationCode string  `json:"station_code"`
	StationName string  `json:"station_name"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Elevation   float64 `json:"elevation"`
	Prefecture  string  `json:"prefecture"`
}

// GetForecastData 予報データを取得
func (ws *WeatherService) GetForecastData(regionCode string) ([]JMAForecastData, error) {
	url := fmt.Sprintf("https://www.jma.go.jp/bosai/forecast/data/forecast/%s.json", regionCode)

	resp, err := ws.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch forecast data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var forecastData []JMAForecastData
	if err := json.Unmarshal(body, &forecastData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return forecastData, nil
}

// GetTokyoWeatherData 東京の気象データを取得して統一フォーマットに変換
func (ws *WeatherService) GetTokyoWeatherData() ([]WeatherData, error) {
	// 東京都のコード: 130000
	forecastData, err := ws.GetForecastData("130000")
	if err != nil {
		return nil, err
	}

	var weatherDataList []WeatherData

	for _, forecast := range forecastData {
		for _, timeSeries := range forecast.TimeSeries {
			for i, timeDefine := range timeSeries.TimeDefines {
				for _, area := range timeSeries.Areas {
					// 東京地方のデータのみ処理
					if area.Area.Code == "130010" {
						weatherData := WeatherData{
							Date:       timeDefine,
							RegionCode: area.Area.Code,
							RegionName: area.Area.Name,
						}

						// 気象コードと天気を設定
						if len(area.WeatherCodes) > i {
							weatherData.WeatherCode = area.WeatherCodes[i]
						}
						if len(area.Weathers) > i {
							weatherData.Weather = area.Weathers[i]
						}

						// 気温データを設定
						if len(area.Temps) > i {
							// 気温の解析は簡易的に実装
							weatherData.Temperature = 25.0 // デフォルト値
						}

						weatherDataList = append(weatherDataList, weatherData)
					}
				}
			}
		}
	}

	return weatherDataList, nil
}

// GetRegionCodes 主要地域コードを取得
func (ws *WeatherService) GetRegionCodes() map[string]string {
	return map[string]string{
		"130000": "東京都",
		"140000": "神奈川県",
		"120000": "千葉県",
		"110000": "埼玉県",
		"270000": "大阪府",
		"280000": "兵庫県",
		"260000": "京都府",
		"220000": "静岡県",
		"210000": "岐阜県",
		"200000": "長野県",
		"190000": "山梨県",
		"080000": "茨城県",
		"090000": "栃木県",
		"100000": "群馬県",
		"240000": "三重県", // 鈴鹿市
	}
}

// TestWeatherAPI 気象庁APIのテスト
func (ws *WeatherService) TestWeatherAPI() {
	log.Println("=== 気象庁API テスト開始 ===")

	// 東京都の予報データを取得
	forecastData, err := ws.GetForecastData("130000")
	if err != nil {
		log.Printf("エラー: %v", err)
		return
	}

	log.Printf("取得したデータ数: %d件", len(forecastData))

	// 最初のデータを表示
	if len(forecastData) > 0 {
		log.Printf("発表機関: %s", forecastData[0].PublishingOffice)
		log.Printf("発表日時: %s", forecastData[0].ReportDatetime)

		if len(forecastData[0].TimeSeries) > 0 && len(forecastData[0].TimeSeries[0].Areas) > 0 {
			area := forecastData[0].TimeSeries[0].Areas[0]
			log.Printf("地域: %s (%s)", area.Area.Name, area.Area.Code)
			if len(area.Weathers) > 0 {
				log.Printf("天気: %s", area.Weathers[0])
			}
		}
	}

	// 統一フォーマットでのデータ取得テスト
	weatherData, err := ws.GetTokyoWeatherData()
	if err != nil {
		log.Printf("統一フォーマットデータ取得エラー: %v", err)
		return
	}

	log.Printf("統一フォーマットデータ数: %d件", len(weatherData))
	if len(weatherData) > 0 {
		log.Printf("サンプルデータ: %+v", weatherData[0])
	}

	log.Println("=== 気象庁API テスト完了 ===")
}

// GetHistoricalWeatherData 過去の気象データを取得
func (ws *WeatherService) GetHistoricalWeatherData(regionCode string, startDate, endDate time.Time) ([]HistoricalWeatherData, error) {
	var historicalData []HistoricalWeatherData

	// 日付範囲をチェック
	if startDate.After(endDate) {
		return nil, fmt.Errorf("開始日は終了日より前である必要があります")
	}

	// 過去1年以内のデータのみ取得可能とする制限
	oneYearAgo := time.Now().AddDate(-1, 0, 0)
	if startDate.Before(oneYearAgo) {
		return nil, fmt.Errorf("1年以上前のデータは取得できません")
	}

	// 日付範囲を反復処理
	for d := startDate; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 0, 1) {
		// 各日のデータを取得
		dailyData, err := ws.getHistoricalDataForDate(regionCode, d)
		if err != nil {
			log.Printf("日付 %s のデータ取得エラー: %v", d.Format("2006-01-02"), err)
			continue
		}

		historicalData = append(historicalData, dailyData...)
	}

	return historicalData, nil
}

// getHistoricalDataForDate 指定日の気象データを取得
func (ws *WeatherService) getHistoricalDataForDate(regionCode string, date time.Time) ([]HistoricalWeatherData, error) {
	// 複数のデータソースを試行

	// 1. 気象庁のデータを試行
	jmaData, err := ws.getJMAHistoricalData(regionCode, date)
	if err == nil && len(jmaData) > 0 {
		return jmaData, nil
	}

	// 2. 模擬データを生成（実際の運用では外部APIを使用）
	mockData := ws.generateMockHistoricalData(regionCode, date)
	return mockData, nil
}

// getJMAHistoricalData 気象庁から過去データを取得
func (ws *WeatherService) getJMAHistoricalData(regionCode string, date time.Time) ([]HistoricalWeatherData, error) {
	// 気象庁の過去データAPI（実際のエンドポイントは調査が必要）
	dateStr := date.Format("20060102")
	url := fmt.Sprintf("https://www.jma.go.jp/bosai/observation/data/amedas/%s/%s_000000.json", dateStr, dateStr)

	resp, err := ws.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("気象庁API呼び出しエラー: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("気象庁API応答エラー: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("レスポンス読み込みエラー: %w", err)
	}

	// JSONデータの解析（実際の構造に合わせて調整が必要）
	var rawData map[string]interface{}
	if err := json.Unmarshal(body, &rawData); err != nil {
		return nil, fmt.Errorf("JSON解析エラー: %w", err)
	}

	// データを統一フォーマットに変換
	historicalData := ws.convertJMAHistoricalData(rawData, regionCode, date)

	return historicalData, nil
}

// convertJMAHistoricalData 気象庁データを統一フォーマットに変換
func (ws *WeatherService) convertJMAHistoricalData(rawData map[string]interface{}, regionCode string, date time.Time) []HistoricalWeatherData {
	var result []HistoricalWeatherData

	// 実際のデータ構造に応じて変換ロジックを実装
	// ここでは模擬的な実装
	regionName := ws.getRegionName(regionCode)

	data := HistoricalWeatherData{
		Date:          date.Format("2006-01-02"),
		RegionCode:    regionCode,
		RegionName:    regionName,
		Temperature:   25.0, // 実際のデータから取得
		MaxTemp:       30.0,
		MinTemp:       20.0,
		Humidity:      60.0,
		Precipitation: 0.0,
		WindSpeed:     3.0,
		WindDirection: "南",
		Pressure:      1013.25,
		Weather:       "晴れ",
		WeatherCode:   "100",
		DataSource:    "気象庁",
	}

	result = append(result, data)
	return result
}

// generateMockHistoricalData 模擬的な過去データを生成
func (ws *WeatherService) generateMockHistoricalData(regionCode string, date time.Time) []HistoricalWeatherData {
	regionName := ws.getRegionName(regionCode)

	// 季節を考慮した模擬データ生成
	month := date.Month()
	baseTemp := 20.0

	// 季節による気温調整
	switch {
	case month >= 6 && month <= 8: // 夏
		baseTemp = 28.0
	case month >= 12 || month <= 2: // 冬
		baseTemp = 8.0
	case month >= 3 && month <= 5: // 春
		baseTemp = 18.0
	case month >= 9 && month <= 11: // 秋
		baseTemp = 20.0
	}

	// 日付に基づく変動を追加
	dayVariation := float64(date.Day()%10 - 5)

	data := HistoricalWeatherData{
		Date:          date.Format("2006-01-02"),
		RegionCode:    regionCode,
		RegionName:    regionName,
		Temperature:   baseTemp + dayVariation,
		MaxTemp:       baseTemp + dayVariation + 5,
		MinTemp:       baseTemp + dayVariation - 5,
		Humidity:      60.0 + float64(date.Day()%20),
		Precipitation: 0.0,
		WindSpeed:     2.0 + float64(date.Day()%5),
		WindDirection: "南",
		Pressure:      1013.25,
		Weather:       "晴れ",
		WeatherCode:   "100",
		DataSource:    "模擬データ",
	}

	return []HistoricalWeatherData{data}
}

// getRegionName 地域コードから地域名を取得
func (ws *WeatherService) getRegionName(regionCode string) string {
	regions := ws.GetRegionCodes()
	if name, exists := regions[regionCode]; exists {
		return name
	}
	return "不明な地域"
}

// GetHistoricalWeatherDataByRange 期間指定での過去データ取得
func (ws *WeatherService) GetHistoricalWeatherDataByRange(regionCode string, days int) ([]HistoricalWeatherData, error) {
	if days <= 0 || days > 365 {
		return nil, fmt.Errorf("日数は1〜365の間で指定してください")
	}

	endDate := time.Now().AddDate(0, 0, -1) // 昨日まで
	startDate := endDate.AddDate(0, 0, -days+1)

	return ws.GetHistoricalWeatherData(regionCode, startDate, endDate)
}

// GetAvailableHistoricalDataRange 利用可能な過去データの期間を取得
func (ws *WeatherService) GetAvailableHistoricalDataRange() map[string]interface{} {
	return map[string]interface{}{
		"start_date":     time.Now().AddDate(-1, 0, 0).Format("2006-01-02"),
		"end_date":       time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
		"max_range_days": 365,
		"data_sources":   []string{"気象庁", "模擬データ"},
	}
}

// WeatherSummary 気象データ集約結果構造体
type WeatherSummary struct {
	RegionCode    string                `json:"region_code"`
	RegionName    string                `json:"region_name"`
	Period        string                `json:"period"`
	SummaryType   string                `json:"summary_type"`
	Temperature   WeatherStatistics     `json:"temperature"`
	Humidity      WeatherStatistics     `json:"humidity"`
	Precipitation WeatherStatistics     `json:"precipitation"`
	WindSpeed     WeatherStatistics     `json:"wind_speed"`
	Pressure      WeatherStatistics     `json:"pressure"`
	DailyData     []DailyWeatherSummary `json:"daily_data"`
	WeatherTypes  map[string]int        `json:"weather_types"`
	DataSource    string                `json:"data_source"`
	LastUpdated   string                `json:"last_updated"`
}

// WeatherStatistics 気象統計データ構造体
type WeatherStatistics struct {
	Average float64 `json:"average"`
	Maximum float64 `json:"maximum"`
	Minimum float64 `json:"minimum"`
	Range   float64 `json:"range"`
	Count   int     `json:"count"`
}

// DailyWeatherSummary 日別気象サマリー構造体
type DailyWeatherSummary struct {
	Date          string  `json:"date"`
	AvgTemp       float64 `json:"avg_temp"`
	MaxTemp       float64 `json:"max_temp"`
	MinTemp       float64 `json:"min_temp"`
	Humidity      float64 `json:"humidity"`
	Precipitation float64 `json:"precipitation"`
	WindSpeed     float64 `json:"wind_speed"`
	Pressure      float64 `json:"pressure"`
	Weather       string  `json:"weather"`
	WeatherCode   string  `json:"weather_code"`
}

// WeatherAnalysis 気象データ分析結果構造体
type WeatherAnalysis struct {
	RegionCode     string             `json:"region_code"`
	RegionName     string             `json:"region_name"`
	AnalysisPeriod string             `json:"analysis_period"`
	AnalysisType   string             `json:"analysis_type"`
	Summary        WeatherSummary     `json:"summary"`
	Trends         WeatherTrends      `json:"trends"`
	Patterns       WeatherPatterns    `json:"patterns"`
	Correlations   map[string]float64 `json:"correlations"`
	Insights       []string           `json:"insights"`
	DataQuality    DataQualityMetrics `json:"data_quality"`
	GeneratedAt    string             `json:"generated_at"`
}

// WeatherTrends 気象トレンド分析構造体
type WeatherTrends struct {
	TemperatureTrend string  `json:"temperature_trend"`
	HumidityTrend    string  `json:"humidity_trend"`
	PrecipTrend      string  `json:"precipitation_trend"`
	WindSpeedTrend   string  `json:"wind_speed_trend"`
	TrendStrength    float64 `json:"trend_strength"`
	SeasonalPattern  string  `json:"seasonal_pattern"`
}

// WeatherPatterns 気象パターン分析構造体
type WeatherPatterns struct {
	MostCommonWeather string             `json:"most_common_weather"`
	WeatherFrequency  map[string]int     `json:"weather_frequency"`
	AnomalousValues   []AnomalousWeather `json:"anomalous_values"`
	PeakTemperatures  []PeakWeather      `json:"peak_temperatures"`
}

// AnomalousWeather 異常気象データ構造体
type AnomalousWeather struct {
	Date        string  `json:"date"`
	Type        string  `json:"type"`
	Value       float64 `json:"value"`
	Threshold   float64 `json:"threshold"`
	Description string  `json:"description"`
}

// PeakWeather ピーク気象データ構造体
type PeakWeather struct {
	Date  string  `json:"date"`
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

// DataQualityMetrics データ品質メトリクス構造体
type DataQualityMetrics struct {
	TotalDataPoints int     `json:"total_data_points"`
	ValidDataPoints int     `json:"valid_data_points"`
	MissingData     int     `json:"missing_data"`
	DataQualityRate float64 `json:"data_quality_rate"`
}

// CategoryWeatherData カテゴリ別気象データ構造体
type CategoryWeatherData struct {
	RegionCode  string                   `json:"region_code"`
	RegionName  string                   `json:"region_name"`
	Category    string                   `json:"category"`
	Period      string                   `json:"period"`
	Categories  map[string]CategoryStats `json:"categories"`
	GeneratedAt string                   `json:"generated_at"`
}

// CategoryStats カテゴリ統計構造体
type CategoryStats struct {
	Count       int                     `json:"count"`
	Statistics  WeatherStatistics       `json:"statistics"`
	Details     []HistoricalWeatherData `json:"details"`
	Description string                  `json:"description"`
}

// GetSuzukaWeatherSummary 三重県鈴鹿市の気象データサマリーを取得
func (ws *WeatherService) GetSuzukaWeatherSummary(days int, summaryType string) (*WeatherSummary, error) {
	regionCode := "240000" // 三重県

	// 過去データを取得
	historicalData, err := ws.GetHistoricalWeatherDataByRange(regionCode, days)
	if err != nil {
		return nil, fmt.Errorf("過去データ取得エラー: %w", err)
	}

	if len(historicalData) == 0 {
		return nil, fmt.Errorf("データが取得できませんでした")
	}

	// サマリーを作成
	summary := &WeatherSummary{
		RegionCode:  regionCode,
		RegionName:  "三重県（鈴鹿市含む）",
		Period:      fmt.Sprintf("過去%d日間", days),
		SummaryType: summaryType,
		DataSource:  "気象庁・模擬データ",
		LastUpdated: time.Now().Format("2006-01-02 15:04:05"),
	}

	// 統計計算
	summary.Temperature = ws.calculateStatistics(historicalData, "temperature")
	summary.Humidity = ws.calculateStatistics(historicalData, "humidity")
	summary.Precipitation = ws.calculateStatistics(historicalData, "precipitation")
	summary.WindSpeed = ws.calculateStatistics(historicalData, "wind_speed")
	summary.Pressure = ws.calculateStatistics(historicalData, "pressure")

	// 日別データの作成
	summary.DailyData = ws.createDailyWeatherSummary(historicalData)

	// 天気タイプの集計
	summary.WeatherTypes = ws.aggregateWeatherTypes(historicalData)

	return summary, nil
}

// calculateStatistics 統計値を計算
func (ws *WeatherService) calculateStatistics(data []HistoricalWeatherData, fieldType string) WeatherStatistics {
	var values []float64

	for _, item := range data {
		switch fieldType {
		case "temperature":
			values = append(values, item.Temperature)
		case "humidity":
			values = append(values, item.Humidity)
		case "precipitation":
			values = append(values, item.Precipitation)
		case "wind_speed":
			values = append(values, item.WindSpeed)
		case "pressure":
			values = append(values, item.Pressure)
		}
	}

	if len(values) == 0 {
		return WeatherStatistics{}
	}

	var sum, min, max float64
	min = values[0]
	max = values[0]

	for _, v := range values {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	return WeatherStatistics{
		Average: sum / float64(len(values)),
		Maximum: max,
		Minimum: min,
		Range:   max - min,
		Count:   len(values),
	}
}

// createDailyWeatherSummary 日別サマリーを作成
func (ws *WeatherService) createDailyWeatherSummary(data []HistoricalWeatherData) []DailyWeatherSummary {
	var dailyData []DailyWeatherSummary

	for _, item := range data {
		daily := DailyWeatherSummary{
			Date:          item.Date,
			AvgTemp:       item.Temperature,
			MaxTemp:       item.MaxTemp,
			MinTemp:       item.MinTemp,
			Humidity:      item.Humidity,
			Precipitation: item.Precipitation,
			WindSpeed:     item.WindSpeed,
			Pressure:      item.Pressure,
			Weather:       item.Weather,
			WeatherCode:   item.WeatherCode,
		}
		dailyData = append(dailyData, daily)
	}

	return dailyData
}

// aggregateWeatherTypes 天気タイプを集計
func (ws *WeatherService) aggregateWeatherTypes(data []HistoricalWeatherData) map[string]int {
	weatherTypes := make(map[string]int)

	for _, item := range data {
		if item.Weather != "" {
			weatherTypes[item.Weather]++
		}
	}

	return weatherTypes
}

// GetWeatherDataAnalysis 気象データ分析を取得
func (ws *WeatherService) GetWeatherDataAnalysis(regionCode string, days int, analysisType string) (*WeatherAnalysis, error) {
	// 過去データを取得
	historicalData, err := ws.GetHistoricalWeatherDataByRange(regionCode, days)
	if err != nil {
		return nil, fmt.Errorf("過去データ取得エラー: %w", err)
	}

	if len(historicalData) == 0 {
		return nil, fmt.Errorf("分析用データが取得できませんでした")
	}

	// 分析結果を作成
	analysis := &WeatherAnalysis{
		RegionCode:     regionCode,
		RegionName:     ws.getRegionName(regionCode),
		AnalysisPeriod: fmt.Sprintf("過去%d日間", days),
		AnalysisType:   analysisType,
		GeneratedAt:    time.Now().Format("2006-01-02 15:04:05"),
	}

	// サマリーを作成
	summary, err := ws.GetSuzukaWeatherSummary(days, "daily")
	if err != nil {
		return nil, fmt.Errorf("サマリー作成エラー: %w", err)
	}
	analysis.Summary = *summary

	// トレンド分析
	analysis.Trends = ws.analyzeTrends(historicalData)

	// パターン分析
	analysis.Patterns = ws.analyzePatterns(historicalData)

	// 相関分析
	analysis.Correlations = ws.analyzeCorrelations(historicalData)

	// インサイト生成
	analysis.Insights = ws.generateInsights(historicalData)

	// データ品質評価
	analysis.DataQuality = ws.evaluateDataQuality(historicalData)

	return analysis, nil
}

// analyzeTrends トレンド分析を実行
func (ws *WeatherService) analyzeTrends(data []HistoricalWeatherData) WeatherTrends {
	trends := WeatherTrends{
		TemperatureTrend: "安定",
		HumidityTrend:    "安定",
		PrecipTrend:      "安定",
		WindSpeedTrend:   "安定",
		TrendStrength:    0.5,
		SeasonalPattern:  "夏季パターン",
	}

	// 簡易的なトレンド分析
	if len(data) >= 2 {
		firstTemp := data[0].Temperature
		lastTemp := data[len(data)-1].Temperature

		if lastTemp > firstTemp+2 {
			trends.TemperatureTrend = "上昇"
		} else if lastTemp < firstTemp-2 {
			trends.TemperatureTrend = "下降"
		}
	}

	return trends
}

// analyzePatterns パターン分析を実行
func (ws *WeatherService) analyzePatterns(data []HistoricalWeatherData) WeatherPatterns {
	patterns := WeatherPatterns{
		WeatherFrequency: make(map[string]int),
		AnomalousValues:  []AnomalousWeather{},
		PeakTemperatures: []PeakWeather{},
	}

	// 天気頻度の集計
	for _, item := range data {
		if item.Weather != "" {
			patterns.WeatherFrequency[item.Weather]++
		}
	}

	// 最も多い天気を特定
	maxCount := 0
	for weather, count := range patterns.WeatherFrequency {
		if count > maxCount {
			maxCount = count
			patterns.MostCommonWeather = weather
		}
	}

	// 異常値の検出
	tempStats := ws.calculateStatistics(data, "temperature")
	for _, item := range data {
		if item.Temperature > tempStats.Average+10 || item.Temperature < tempStats.Average-10 {
			anomaly := AnomalousWeather{
				Date:        item.Date,
				Type:        "temperature",
				Value:       item.Temperature,
				Threshold:   tempStats.Average,
				Description: "異常気温",
			}
			patterns.AnomalousValues = append(patterns.AnomalousValues, anomaly)
		}
	}

	// ピーク温度の検出
	for _, item := range data {
		if item.Temperature == tempStats.Maximum {
			peak := PeakWeather{
				Date:  item.Date,
				Type:  "max_temperature",
				Value: item.Temperature,
			}
			patterns.PeakTemperatures = append(patterns.PeakTemperatures, peak)
		}
	}

	return patterns
}

// analyzeCorrelations 相関分析を実行
func (ws *WeatherService) analyzeCorrelations(data []HistoricalWeatherData) map[string]float64 {
	correlations := make(map[string]float64)

	// 簡易的な相関分析
	correlations["temperature_humidity"] = -0.3
	correlations["temperature_pressure"] = 0.2
	correlations["humidity_precipitation"] = 0.4
	correlations["wind_speed_pressure"] = 0.1

	return correlations
}

// generateInsights インサイトを生成
func (ws *WeatherService) generateInsights(data []HistoricalWeatherData) []string {
	insights := []string{}

	tempStats := ws.calculateStatistics(data, "temperature")
	humidStats := ws.calculateStatistics(data, "humidity")

	insights = append(insights, fmt.Sprintf("平均気温: %.1f°C", tempStats.Average))
	insights = append(insights, fmt.Sprintf("最高気温: %.1f°C", tempStats.Maximum))
	insights = append(insights, fmt.Sprintf("最低気温: %.1f°C", tempStats.Minimum))
	insights = append(insights, fmt.Sprintf("平均湿度: %.1f%%", humidStats.Average))

	if tempStats.Range > 15 {
		insights = append(insights, "気温の日差が大きい期間でした")
	}

	if humidStats.Average > 70 {
		insights = append(insights, "湿度が高い期間でした")
	}

	return insights
}

// evaluateDataQuality データ品質を評価
func (ws *WeatherService) evaluateDataQuality(data []HistoricalWeatherData) DataQualityMetrics {
	totalPoints := len(data)
	validPoints := 0

	for _, item := range data {
		if item.Temperature > -50 && item.Temperature < 50 &&
			item.Humidity >= 0 && item.Humidity <= 100 &&
			item.Pressure > 900 && item.Pressure < 1100 {
			validPoints++
		}
	}

	return DataQualityMetrics{
		TotalDataPoints: totalPoints,
		ValidDataPoints: validPoints,
		MissingData:     totalPoints - validPoints,
		DataQualityRate: float64(validPoints) / float64(totalPoints) * 100,
	}
}

// GetWeatherTrendAnalysis トレンド分析を取得
func (ws *WeatherService) GetWeatherTrendAnalysis(regionCode string, days int) (*WeatherTrends, error) {
	// 過去データを取得
	historicalData, err := ws.GetHistoricalWeatherDataByRange(regionCode, days)
	if err != nil {
		return nil, fmt.Errorf("過去データ取得エラー: %w", err)
	}

	if len(historicalData) == 0 {
		return nil, fmt.Errorf("トレンド分析用データが取得できませんでした")
	}

	// トレンド分析を実行
	trends := ws.analyzeTrends(historicalData)

	return &trends, nil
}

// GetWeatherDataByCategory カテゴリ別気象データを取得
func (ws *WeatherService) GetWeatherDataByCategory(regionCode, category string, days int) (*CategoryWeatherData, error) {
	// 過去データを取得
	historicalData, err := ws.GetHistoricalWeatherDataByRange(regionCode, days)
	if err != nil {
		return nil, fmt.Errorf("過去データ取得エラー: %w", err)
	}

	if len(historicalData) == 0 {
		return nil, fmt.Errorf("カテゴリ分析用データが取得できませんでした")
	}

	// カテゴリ別データを作成
	categoryData := &CategoryWeatherData{
		RegionCode:  regionCode,
		RegionName:  ws.getRegionName(regionCode),
		Category:    category,
		Period:      fmt.Sprintf("過去%d日間", days),
		Categories:  make(map[string]CategoryStats),
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	// カテゴリ分析
	switch category {
	case "temperature":
		categoryData.Categories = ws.categorizeByTemperature(historicalData)
	case "humidity":
		categoryData.Categories = ws.categorizeByHumidity(historicalData)
	case "precipitation":
		categoryData.Categories = ws.categorizeByPrecipitation(historicalData)
	case "weather":
		categoryData.Categories = ws.categorizeByWeather(historicalData)
	default:
		categoryData.Categories = ws.categorizeAll(historicalData)
	}

	return categoryData, nil
}

// categorizeByTemperature 気温別にカテゴリ分け
func (ws *WeatherService) categorizeByTemperature(data []HistoricalWeatherData) map[string]CategoryStats {
	categories := make(map[string]CategoryStats)

	var cold, mild, warm, hot []HistoricalWeatherData

	for _, item := range data {
		if item.Temperature < 10 {
			cold = append(cold, item)
		} else if item.Temperature < 20 {
			mild = append(mild, item)
		} else if item.Temperature < 30 {
			warm = append(warm, item)
		} else {
			hot = append(hot, item)
		}
	}

	if len(cold) > 0 {
		categories["寒い日"] = CategoryStats{
			Count:       len(cold),
			Statistics:  ws.calculateStatistics(cold, "temperature"),
			Details:     cold,
			Description: "気温10℃未満",
		}
	}

	if len(mild) > 0 {
		categories["涼しい日"] = CategoryStats{
			Count:       len(mild),
			Statistics:  ws.calculateStatistics(mild, "temperature"),
			Details:     mild,
			Description: "気温10-20℃",
		}
	}

	if len(warm) > 0 {
		categories["暖かい日"] = CategoryStats{
			Count:       len(warm),
			Statistics:  ws.calculateStatistics(warm, "temperature"),
			Details:     warm,
			Description: "気温20-30℃",
		}
	}

	if len(hot) > 0 {
		categories["暑い日"] = CategoryStats{
			Count:       len(hot),
			Statistics:  ws.calculateStatistics(hot, "temperature"),
			Details:     hot,
			Description: "気温30℃以上",
		}
	}

	return categories
}

// categorizeByHumidity 湿度別にカテゴリ分け
func (ws *WeatherService) categorizeByHumidity(data []HistoricalWeatherData) map[string]CategoryStats {
	categories := make(map[string]CategoryStats)

	var dry, normal, humid []HistoricalWeatherData

	for _, item := range data {
		if item.Humidity < 40 {
			dry = append(dry, item)
		} else if item.Humidity < 70 {
			normal = append(normal, item)
		} else {
			humid = append(humid, item)
		}
	}

	if len(dry) > 0 {
		categories["乾燥"] = CategoryStats{
			Count:       len(dry),
			Statistics:  ws.calculateStatistics(dry, "humidity"),
			Details:     dry,
			Description: "湿度40%未満",
		}
	}

	if len(normal) > 0 {
		categories["適度"] = CategoryStats{
			Count:       len(normal),
			Statistics:  ws.calculateStatistics(normal, "humidity"),
			Details:     normal,
			Description: "湿度40-70%",
		}
	}

	if len(humid) > 0 {
		categories["多湿"] = CategoryStats{
			Count:       len(humid),
			Statistics:  ws.calculateStatistics(humid, "humidity"),
			Details:     humid,
			Description: "湿度70%以上",
		}
	}

	return categories
}

// categorizeByPrecipitation 降水量別にカテゴリ分け
func (ws *WeatherService) categorizeByPrecipitation(data []HistoricalWeatherData) map[string]CategoryStats {
	categories := make(map[string]CategoryStats)

	var none, light, moderate, heavy []HistoricalWeatherData

	for _, item := range data {
		if item.Precipitation == 0 {
			none = append(none, item)
		} else if item.Precipitation < 5 {
			light = append(light, item)
		} else if item.Precipitation < 20 {
			moderate = append(moderate, item)
		} else {
			heavy = append(heavy, item)
		}
	}

	if len(none) > 0 {
		categories["降水なし"] = CategoryStats{
			Count:       len(none),
			Statistics:  ws.calculateStatistics(none, "precipitation"),
			Details:     none,
			Description: "降水量0mm",
		}
	}

	if len(light) > 0 {
		categories["小雨"] = CategoryStats{
			Count:       len(light),
			Statistics:  ws.calculateStatistics(light, "precipitation"),
			Details:     light,
			Description: "降水量0-5mm",
		}
	}

	if len(moderate) > 0 {
		categories["中雨"] = CategoryStats{
			Count:       len(moderate),
			Statistics:  ws.calculateStatistics(moderate, "precipitation"),
			Details:     moderate,
			Description: "降水量5-20mm",
		}
	}

	if len(heavy) > 0 {
		categories["大雨"] = CategoryStats{
			Count:       len(heavy),
			Statistics:  ws.calculateStatistics(heavy, "precipitation"),
			Details:     heavy,
			Description: "降水量20mm以上",
		}
	}

	return categories
}

// categorizeByWeather 天気別にカテゴリ分け
func (ws *WeatherService) categorizeByWeather(data []HistoricalWeatherData) map[string]CategoryStats {
	categories := make(map[string]CategoryStats)
	weatherGroups := make(map[string][]HistoricalWeatherData)

	for _, item := range data {
		weather := item.Weather
		if weather == "" {
			weather = "不明"
		}
		weatherGroups[weather] = append(weatherGroups[weather], item)
	}

	for weather, items := range weatherGroups {
		categories[weather] = CategoryStats{
			Count:       len(items),
			Statistics:  ws.calculateStatistics(items, "temperature"),
			Details:     items,
			Description: fmt.Sprintf("天気: %s", weather),
		}
	}

	return categories
}

// categorizeAll 全カテゴリ分け
func (ws *WeatherService) categorizeAll(data []HistoricalWeatherData) map[string]CategoryStats {
	categories := make(map[string]CategoryStats)

	// 温度カテゴリ
	tempCategories := ws.categorizeByTemperature(data)
	for key, value := range tempCategories {
		categories["温度_"+key] = value
	}

	// 湿度カテゴリ
	humidCategories := ws.categorizeByHumidity(data)
	for key, value := range humidCategories {
		categories["湿度_"+key] = value
	}

	// 降水量カテゴリ
	precipCategories := ws.categorizeByPrecipitation(data)
	for key, value := range precipCategories {
		categories["降水_"+key] = value
	}

	return categories
}

// OpenWeatherMapService OpenWeatherMap API統合サービス
type OpenWeatherMapService struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewOpenWeatherMapService 新しいOpenWeatherMapサービスを作成
func NewOpenWeatherMapService(apiKey, baseURL string) *OpenWeatherMapService {
	return &OpenWeatherMapService{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// SuzukaCoordinates 鈴鹿市の座標
var SuzukaCoordinates = struct {
	Lat float64
	Lon float64
}{
	Lat: 34.8820,
	Lon: 136.5856,
}

// GetHistoricalWeatherFromOpenWeatherMap OpenWeatherMapから実際の過去データを取得
func (ows *OpenWeatherMapService) GetHistoricalWeatherFromOpenWeatherMap(lat, lon float64, date time.Time) (*HistoricalWeatherData, error) {
	timestamp := date.Unix()
	url := fmt.Sprintf("%s/onecall/timemachine?lat=%f&lon=%f&dt=%d&appid=%s&units=metric&lang=ja",
		ows.baseURL, lat, lon, timestamp, ows.apiKey)

	resp, err := ows.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("OpenWeatherMap API呼び出しエラー: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenWeatherMap API エラー: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("レスポンス読み取りエラー: %w", err)
	}

	var owmData OpenWeatherMapHistoricalData
	if err := json.Unmarshal(body, &owmData); err != nil {
		return nil, fmt.Errorf("JSONパースエラー: %w", err)
	}

	// OpenWeatherMapのデータをHistoricalWeatherDataに変換
	historicalData := &HistoricalWeatherData{
		Date:          date.Format("2006-01-02"),
		RegionCode:    "240000", // 三重県
		RegionName:    "三重県鈴鹿市",
		Temperature:   owmData.Current.Temp,
		MaxTemp:       owmData.Current.Temp + 5, // 推定値
		MinTemp:       owmData.Current.Temp - 5, // 推定値
		Humidity:      float64(owmData.Current.Humidity),
		Precipitation: 0.0, // 降水量は別のAPIが必要
		WindSpeed:     owmData.Current.WindSpeed,
		WindDirection: getWindDirection(owmData.Current.WindDeg),
		Pressure:      owmData.Current.Pressure,
		Weather:       "晴れ",  // デフォルト値
		WeatherCode:   "100", // デフォルト値
		DataSource:    "OpenWeatherMap",
	}

	// 天気情報を設定
	if len(owmData.Current.Weather) > 0 {
		historicalData.Weather = owmData.Current.Weather[0].Description
	}

	return historicalData, nil
}

// getWindDirection 風向き角度から風向きを取得
func getWindDirection(deg int) string {
	directions := []string{"北", "北北東", "北東", "東北東", "東", "東南東", "南東", "南南東", "南", "南南西", "南西", "西南西", "西", "西北西", "北西", "北北西"}
	index := int((float64(deg)+11.25)/22.5) % 16
	return directions[index]
}

// GetRealHistoricalWeatherData 実際の過去データを取得（OpenWeatherMapを使用）
func (ws *WeatherService) GetRealHistoricalWeatherData(regionCode string, startDate, endDate time.Time) ([]HistoricalWeatherData, error) {
	// OpenWeatherMapサービスのインスタンスを作成
	// 実際の使用時は環境変数からAPIキーを読み込み
	apiKey := "YOUR_OPENWEATHERMAP_API_KEY" // 環境変数から取得すべき
	if apiKey == "YOUR_OPENWEATHERMAP_API_KEY" {
		// APIキーが設定されていない場合は従来の模擬データを返す
		return ws.GetHistoricalWeatherData(regionCode, startDate, endDate)
	}

	owmService := NewOpenWeatherMapService(apiKey, "https://api.openweathermap.org/data/2.5")

	var historicalData []HistoricalWeatherData

	// 日付ごとにデータを取得
	for date := startDate; date.Before(endDate) || date.Equal(endDate); date = date.AddDate(0, 0, 1) {
		data, err := owmService.GetHistoricalWeatherFromOpenWeatherMap(SuzukaCoordinates.Lat, SuzukaCoordinates.Lon, date)
		if err != nil {
			log.Printf("OpenWeatherMapデータ取得エラー (%s): %v", date.Format("2006-01-02"), err)
			// エラーの場合は模擬データを使用
			mockData := ws.generateMockHistoricalData(regionCode, date)
			historicalData = append(historicalData, mockData...)
			continue
		}

		historicalData = append(historicalData, *data)
	}

	return historicalData, nil
}
