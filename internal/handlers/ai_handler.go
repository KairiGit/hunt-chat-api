package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"hunt-chat-api/internal/services"

	"github.com/gin-gonic/gin"
)

// AIHandler AI統合ハンドラー
type AIHandler struct {
	azureOpenAIService    *services.AzureOpenAIService
	weatherService        *services.WeatherService
	demandForecastService *services.DemandForecastService
}

// NewAIHandler 新しいAI統合ハンドラーを作成
func NewAIHandler(azureOpenAIService *services.AzureOpenAIService, weatherService *services.WeatherService, demandForecastService *services.DemandForecastService) *AIHandler {
	return &AIHandler{
		azureOpenAIService:    azureOpenAIService,
		weatherService:        weatherService,
		demandForecastService: demandForecastService,
	}
}

// AnalyzeWeatherDataRequest 気象データ分析リクエスト
type AnalyzeWeatherDataRequest struct {
	RegionCode string `json:"region_code"`
	Days       int    `json:"days"`
}

// AnalyzeWeatherDataResponse 気象データ分析レスポンス
type AnalyzeWeatherDataResponse struct {
	RegionCode string `json:"region_code"`
	Period     string `json:"period"`
	Analysis   string `json:"analysis"`
	Insights   string `json:"insights"`
}

// AnalyzeWeatherData 気象データをAIで分析
func (ah *AIHandler) AnalyzeWeatherData(c *gin.Context) {
	var req AnalyzeWeatherDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "リクエストの形式が正しくありません",
		})
		return
	}

	// デフォルト値を設定
	if req.RegionCode == "" {
		req.RegionCode = "240000" // 三重県
	}
	if req.Days == 0 {
		req.Days = 30
	}

	// 気象データを取得
	weatherSummary, err := ah.weatherService.GetSuzukaWeatherSummary(req.Days, "daily")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "気象データの取得に失敗しました",
		})
		return
	}

	// 気象データをJSON文字列に変換
	weatherDataJSON, err := json.Marshal(weatherSummary)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "気象データの変換に失敗しました",
		})
		return
	}

	// Azure OpenAI で分析
	analysis, err := ah.azureOpenAIService.AnalyzeWeatherData(string(weatherDataJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "AI分析に失敗しました: " + err.Error(),
		})
		return
	}

	response := AnalyzeWeatherDataResponse{
		RegionCode: req.RegionCode,
		Period:     "過去" + strconv.Itoa(req.Days) + "日間",
		Analysis:   analysis,
		Insights:   analysis,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// GenerateDemandInsightsRequest 需要洞察生成リクエスト
type GenerateDemandInsightsRequest struct {
	RegionCode      string `json:"region_code"`
	Days            int    `json:"days"`
	ProductCategory string `json:"product_category"`
}

// GenerateDemandInsightsResponse 需要洞察生成レスポンス
type GenerateDemandInsightsResponse struct {
	RegionCode      string   `json:"region_code"`
	Period          string   `json:"period"`
	ProductCategory string   `json:"product_category"`
	Insights        string   `json:"insights"`
	Recommendations []string `json:"recommendations"`
}

// GenerateDemandInsights AI を使用した需要予測洞察の生成
func (ah *AIHandler) GenerateDemandInsights(c *gin.Context) {
	var req GenerateDemandInsightsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "リクエストの形式が正しくありません",
		})
		return
	}

	// デフォルト値を設定
	if req.RegionCode == "" {
		req.RegionCode = "240000" // 三重県
	}
	if req.Days == 0 {
		req.Days = 30
	}
	if req.ProductCategory == "" {
		req.ProductCategory = "一般製造業"
	}

	// 気象データを取得
	weatherSummary, err := ah.weatherService.GetSuzukaWeatherSummary(req.Days, "daily")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "気象データの取得に失敗しました",
		})
		return
	}

	// 過去データを取得
	historicalData, err := ah.weatherService.GetHistoricalWeatherDataByRange(req.RegionCode, req.Days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "過去データの取得に失敗しました",
		})
		return
	}

	// データをJSON文字列に変換
	weatherDataJSON, _ := json.Marshal(weatherSummary)
	historicalDataJSON, _ := json.Marshal(historicalData)

	// Azure OpenAI で需要洞察を生成
	insights, err := ah.azureOpenAIService.GenerateDemandInsights(string(weatherDataJSON), string(historicalDataJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "AI洞察生成に失敗しました: " + err.Error(),
		})
		return
	}

	response := GenerateDemandInsightsResponse{
		RegionCode:      req.RegionCode,
		Period:          "過去" + strconv.Itoa(req.Days) + "日間",
		ProductCategory: req.ProductCategory,
		Insights:        insights,
		Recommendations: []string{
			"気象データを定期的に監視し、需要変動に備えてください",
			"季節性パターンを考慮した在庫管理を実施してください",
			"予測精度向上のため、過去データの蓄積を継続してください",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// PredictDemandWithAIRequest AI需要予測リクエスト
type PredictDemandWithAIRequest struct {
	RegionCode      string `json:"region_code"`
	Days            int    `json:"days"`
	ProductCategory string `json:"product_category"`
}

// PredictDemandWithAIResponse AI需要予測レスポンス
type PredictDemandWithAIResponse struct {
	RegionCode      string   `json:"region_code"`
	Period          string   `json:"period"`
	ProductCategory string   `json:"product_category"`
	Prediction      string   `json:"prediction"`
	Confidence      float64  `json:"confidence"`
	Factors         []string `json:"factors"`
}

// PredictDemandWithAI AI を使用した需要予測
func (ah *AIHandler) PredictDemandWithAI(c *gin.Context) {
	var req PredictDemandWithAIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "リクエストの形式が正しくありません",
		})
		return
	}

	// デフォルト値を設定
	if req.RegionCode == "" {
		req.RegionCode = "240000" // 三重県
	}
	if req.Days == 0 {
		req.Days = 30
	}
	if req.ProductCategory == "" {
		req.ProductCategory = "一般製造業"
	}

	// 気象データを取得
	weatherSummary, err := ah.weatherService.GetSuzukaWeatherSummary(req.Days, "daily")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "気象データの取得に失敗しました",
		})
		return
	}

	// 過去データを取得
	historicalData, err := ah.weatherService.GetHistoricalWeatherDataByRange(req.RegionCode, req.Days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "過去データの取得に失敗しました",
		})
		return
	}

	// データをJSON文字列に変換
	weatherDataJSON, _ := json.Marshal(weatherSummary)
	historicalDataJSON, _ := json.Marshal(historicalData)

	// Azure OpenAI で需要予測
	prediction, err := ah.azureOpenAIService.PredictDemandWithAI(string(weatherDataJSON), string(historicalDataJSON), req.ProductCategory)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "AI需要予測に失敗しました: " + err.Error(),
		})
		return
	}

	response := PredictDemandWithAIResponse{
		RegionCode:      req.RegionCode,
		Period:          "過去" + strconv.Itoa(req.Days) + "日間",
		ProductCategory: req.ProductCategory,
		Prediction:      prediction,
		Confidence:      0.75, // 固定値（実際の実装では動的に計算）
		Factors: []string{
			"気象条件（気温、湿度、降水量）",
			"季節性パターン",
			"過去の需要トレンド",
			"地域特性",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// ExplainForecastRequest 予測説明リクエスト
type ExplainForecastRequest struct {
	ForecastData string `json:"forecast_data"`
	Factors      string `json:"factors"`
}

// ExplainForecastResponse 予測説明レスポンス
type ExplainForecastResponse struct {
	Explanation string   `json:"explanation"`
	KeyFactors  []string `json:"key_factors"`
}

// ExplainForecast 予測結果の説明可能性を提供
func (ah *AIHandler) ExplainForecast(c *gin.Context) {
	var req ExplainForecastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "リクエストの形式が正しくありません",
		})
		return
	}

	// Azure OpenAI で予測説明を生成
	explanation, err := ah.azureOpenAIService.ExplainForecast(req.ForecastData, req.Factors)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "予測説明の生成に失敗しました: " + err.Error(),
		})
		return
	}

	response := ExplainForecastResponse{
		Explanation: explanation,
		KeyFactors: []string{
			"気象パターンの影響",
			"季節性要因",
			"地域特性",
			"過去データとの相関",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// GetAICapabilities AI機能の一覧を取得
func (ah *AIHandler) GetAICapabilities(c *gin.Context) {
	capabilities := map[string]interface{}{
		"weather_analysis": map[string]interface{}{
			"description": "気象データの包括的な分析",
			"endpoint":    "/api/v1/ai/analyze-weather",
			"method":      "POST",
		},
		"demand_insights": map[string]interface{}{
			"description": "需要予測の洞察生成",
			"endpoint":    "/api/v1/ai/demand-insights",
			"method":      "POST",
		},
		"demand_prediction": map[string]interface{}{
			"description": "AI を使用した需要予測",
			"endpoint":    "/api/v1/ai/predict-demand",
			"method":      "POST",
		},
		"forecast_explanation": map[string]interface{}{
			"description": "予測結果の説明可能性",
			"endpoint":    "/api/v1/ai/explain-forecast",
			"method":      "POST",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"capabilities": capabilities,
		"ai_service":   "Azure OpenAI",
	})
}

// GenerateAnomalyQuestion 異常検知結果から質問を生成する
func (ah *AIHandler) GenerateAnomalyQuestion(c *gin.Context) {
	regionCode := c.Query("region_code")
	if regionCode == "" {
		regionCode = "240000" // デフォルト：三重県
	}

	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	// 1. 異常を検知
	anomalies, err := ah.demandForecastService.DetectAnomalies(regionCode, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "異常検知の実行に失敗しました: " + err.Error(),
		})
		return
	}

	if len(anomalies) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"message":  "特筆すべき異常は見つかりませんでした。",
			"question": "",
		})
		return
	}

	// 2. 最初の異常から質問を生成
	// 簡単のため、最初の異常を使用する
	targetAnomaly := anomalies[0]
	question, err := ah.azureOpenAIService.GenerateQuestionFromAnomaly(targetAnomaly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "AIからの質問生成に失敗しました: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"message":        "異常を検知し、質問を生成しました。",
		"question":       question,
		"source_anomaly": targetAnomaly,
	})
}
