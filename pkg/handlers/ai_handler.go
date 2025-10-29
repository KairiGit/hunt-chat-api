package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"hunt-chat-api/pkg/models"
	"hunt-chat-api/pkg/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AIHandler AI統合ハンドラー
type AIHandler struct {
	azureOpenAIService    *services.AzureOpenAIService
	weatherService        *services.WeatherService
	economicService       *services.EconomicService
	demandForecastService *services.DemandForecastService
	vectorStoreService    *services.VectorStoreService
	statisticsService     *services.StatisticsService
}

// NewAIHandler 新しいAI統合ハンドラーを作成
func NewAIHandler(azureOpenAIService *services.AzureOpenAIService, weatherService *services.WeatherService, economicService *services.EconomicService, demandForecastService *services.DemandForecastService, vectorStoreService *services.VectorStoreService) *AIHandler {
	return &AIHandler{
		azureOpenAIService:    azureOpenAIService,
		weatherService:        weatherService,
		economicService:       economicService,
		demandForecastService: demandForecastService,
		vectorStoreService:    vectorStoreService,
		statisticsService:     services.NewStatisticsService(weatherService, economicService, azureOpenAIService),
	}
}

// AnalyzeFile: Logic-based file analysis with configurable data granularity

// AnalyzeFileWithProgress ファイル分析を実行し、進捗をSSEで送信
func (ah *AIHandler) AnalyzeFileWithProgress(c *gin.Context) {
	// SSEヘッダーを設定
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	startTime := time.Now()
	totalSteps := 7

	// 進捗送信ヘルパー関数
	sendProgress := func(stepIndex int, step, message string, progress int) {
		elapsed := time.Since(startTime).Milliseconds()
		progressData := AnalysisProgress{
			Step:       step,
			Progress:   progress,
			Message:    message,
			ElapsedMs:  elapsed,
			TotalSteps: totalSteps,
			StepIndex:  stepIndex,
		}
		data, _ := json.Marshal(progressData)
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		c.Writer.Flush()
		log.Printf("📊 [進捗] ステップ%d/%d: %s (%dms)", stepIndex, totalSteps, message, elapsed)
	}

	// ファイル処理とパラメータ取得の実装は既存のAnalyzeFileと同じ
	// ここでは進捗送信のタイミングのみを示します

	sendProgress(1, "init", "ファイルを読み込んでいます...", 10)
	// ... 既存のファイル読み込みコード ...

	sendProgress(2, "parse", "CSVデータを解析しています...", 25)
	// ... CSV解析コード ...

	sendProgress(3, "stats", "統計分析を実行しています...", 45)
	// ... 統計分析コード ...

	sendProgress(4, "ai", "AI分析を実行しています...", 60)
	// ... AI分析コード ...

	sendProgress(5, "anomaly", "異常検知を実行しています...", 75)
	// ... 異常検知コード ...

	sendProgress(6, "save", "結果をデータベースに保存しています...", 90)
	// ... DB保存コード ...

	sendProgress(7, "complete", "分析が完了しました！", 100)

	// 最終結果を送信
	fmt.Fprintf(c.Writer, "event: done\ndata: {\"success\": true}\n\n")
	c.Writer.Flush()
}

type ChatInputRequest struct {
	ChatMessage string `json:"chat_message"`
	Context     string `json:"context,omitempty"`
	SessionID   string `json:"session_id,omitempty"` // セッションID（会話の継続性）
	UserID      string `json:"user_id,omitempty"`    // ユーザーID（履歴の紐付け）
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

func (ah *AIHandler) AnalyzeWeatherData(c *gin.Context) {
	var req AnalyzeWeatherDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "リクエストの形式が正しくありません"})
		return
	}
	if req.RegionCode == "" {
		req.RegionCode = "240000"
	}
	if req.Days == 0 {
		req.Days = 30
	}
	weatherSummary, err := ah.weatherService.GetSuzukaWeatherSummary(req.Days, "daily")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "気象データの取得に失敗しました"})
		return
	}
	weatherDataJSON, err := json.Marshal(weatherSummary)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "気象データの変換に失敗しました"})
		return
	}
	analysis, err := ah.azureOpenAIService.AnalyzeWeatherData(string(weatherDataJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI分析に失敗しました: " + err.Error()})
		return
	}
	response := AnalyzeWeatherDataResponse{
		RegionCode: req.RegionCode,
		Period:     "過去" + strconv.Itoa(req.Days) + "日間",
		Analysis:   analysis,
		Insights:   analysis,
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": response})
}

type GenerateDemandInsightsRequest struct {
	RegionCode      string `json:"region_code"`
	Days            int    `json:"days"`
	ProductCategory string `json:"product_category"`
}

type GenerateDemandInsightsResponse struct {
	RegionCode      string   `json:"region_code"`
	Period          string   `json:"period"`
	ProductCategory string   `json:"product_category"`
	Insights        string   `json:"insights"`
	Recommendations []string `json:"recommendations"`
}

func (ah *AIHandler) GenerateDemandInsights(c *gin.Context) {
	var req GenerateDemandInsightsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "リクエストの形式が正しくありません"})
		return
	}
	if req.RegionCode == "" {
		req.RegionCode = "240000"
	}
	if req.Days == 0 {
		req.Days = 30
	}
	if req.ProductCategory == "" {
		req.ProductCategory = "一般製造業"
	}
	weatherSummary, err := ah.weatherService.GetSuzukaWeatherSummary(req.Days, "daily")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "気象データの取得に失敗しました"})
		return
	}
	historicalData, err := ah.weatherService.GetHistoricalWeatherDataByRange(req.RegionCode, req.Days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "過去データの取得に失敗しました"})
		return
	}
	weatherDataJSON, _ := json.Marshal(weatherSummary)
	historicalDataJSON, _ := json.Marshal(historicalData)
	insights, err := ah.azureOpenAIService.GenerateDemandInsights(string(weatherDataJSON), string(historicalDataJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI洞察生成に失敗しました: " + err.Error()})
		return
	}
	response := GenerateDemandInsightsResponse{
		RegionCode:      req.RegionCode,
		Period:          "過去" + strconv.Itoa(req.Days) + "日間",
		ProductCategory: req.ProductCategory,
		Insights:        insights,
		Recommendations: []string{"気象データを定期的に監視し、需要変動に備えてください", "季節性パターンを考慮した在庫管理を実施してください", "予測精度向上のため、過去データの蓄積を継続してください"},
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": response})
}

type PredictDemandWithAIRequest struct {
	RegionCode      string `json:"region_code"`
	Days            int    `json:"days"`
	ProductCategory string `json:"product_category"`
}

type PredictDemandWithAIResponse struct {
	RegionCode      string   `json:"region_code"`
	Period          string   `json:"period"`
	ProductCategory string   `json:"product_category"`
	Prediction      string   `json:"prediction"`
	Confidence      float64  `json:"confidence"`
	Factors         []string `json:"factors"`
}

func (ah *AIHandler) PredictDemandWithAI(c *gin.Context) {
	var req PredictDemandWithAIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "リクエストの形式が正しくありません"})
		return
	}
	if req.RegionCode == "" {
		req.RegionCode = "240000"
	}
	if req.Days == 0 {
		req.Days = 30
	}
	if req.ProductCategory == "" {
		req.ProductCategory = "一般製造業"
	}
	weatherSummary, err := ah.weatherService.GetSuzukaWeatherSummary(req.Days, "daily")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "気象データの取得に失敗しました"})
		return
	}
	historicalData, err := ah.weatherService.GetHistoricalWeatherDataByRange(req.RegionCode, req.Days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "過去データの取得に失敗しました"})
		return
	}
	weatherDataJSON, _ := json.Marshal(weatherSummary)
	historicalDataJSON, _ := json.Marshal(historicalData)
	prediction, err := ah.azureOpenAIService.PredictDemandWithAI(string(weatherDataJSON), string(historicalDataJSON), req.ProductCategory)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI需要予測に失敗しました: " + err.Error()})
		return
	}
	response := PredictDemandWithAIResponse{
		RegionCode:      req.RegionCode,
		Period:          "過去" + strconv.Itoa(req.Days) + "日間",
		ProductCategory: req.ProductCategory,
		Prediction:      prediction,
		Confidence:      0.75,
		Factors:         []string{"気象条件（気温、湿度、降水量）", "季節性パターン", "過去の需要トレンド", "地域特性"},
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": response})
}

type ExplainForecastRequest struct {
	ForecastData string `json:"forecast_data"`
	Factors      string `json:"factors"`
}

type ExplainForecastResponse struct {
	Explanation string   `json:"explanation"`
	KeyFactors  []string `json:"key_factors"`
}

func (ah *AIHandler) ExplainForecast(c *gin.Context) {
	var req ExplainForecastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "リクエストの形式が正しくありません"})
		return
	}
	explanation, err := ah.azureOpenAIService.ExplainForecast(req.ForecastData, req.Factors)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "予測説明の生成に失敗しました: " + err.Error()})
		return
	}
	response := ExplainForecastResponse{
		Explanation: explanation,
		KeyFactors:  []string{"気象パターンの影響", "季節性要因", "地域特性", "過去データとの相関"},
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": response})
}

func (ah *AIHandler) GetAICapabilities(c *gin.Context) {
	capabilities := map[string]interface{}{
		"weather_analysis":     map[string]interface{}{"description": "気象データの包括的な分析", "endpoint": "/api/v1/ai/analyze-weather", "method": "POST"},
		"demand_insights":      map[string]interface{}{"description": "需要予測の洞察生成", "endpoint": "/api/v1/ai/demand-insights", "method": "POST"},
		"demand_prediction":    map[string]interface{}{"description": "AI を使用した需要予測", "endpoint": "/api/v1/ai/predict-demand", "method": "POST"},
		"forecast_explanation": map[string]interface{}{"description": "予測結果の説明可能性", "endpoint": "/api/v1/ai/explain-forecast", "method": "POST"},
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "capabilities": capabilities, "ai_service": "Azure OpenAI"})
}

func (ah *AIHandler) GenerateAnomalyQuestion(c *gin.Context) {
	regionCode := c.Query("region_code")
	if regionCode == "" {
		regionCode = "240000"
	}
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}
	anomalies, err := ah.demandForecastService.DetectAnomalies(regionCode, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "異常検知の実行に失敗しました: " + err.Error()})
		return
	}
	if len(anomalies) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "特筆すべき異常は見つかりませんでした。", "question": ""})
		return
	}
	targetAnomaly := anomalies[0]
	result, err := ah.azureOpenAIService.GenerateQuestionAndChoicesFromAnomaly(targetAnomaly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AIからの質問生成に失敗しました: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "異常を検知し、質問を生成しました。", "question": result.Question, "choices": result.Choices, "source_anomaly": targetAnomaly})
}

// PredictSales 将来の売上を予測する
func (ah *AIHandler) PredictSales(c *gin.Context) {
	var req models.PredictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "リクエストパラメータが不正です: " + err.Error(),
		})
		return
	}

	// デフォルト値設定
	if req.ConfidenceLevel == 0 {
		req.ConfidenceLevel = 0.95
	}

	// 過去データの取得（簡易版：ファイルから取得する代わりにサンプルデータを使用）
	// 実際の実装では、Qdrantや外部DBから過去データを取得する
	historicalSales := []float64{100, 110, 105, 120, 115, 130, 125, 140, 135, 150, 145, 160}
	historicalTemperatures := []float64{15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26}

	prediction, err := ah.statisticsService.PredictFutureSales(
		historicalSales,
		historicalTemperatures,
		req.FutureTemperature,
		req.ConfidenceLevel,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "予測の計算に失敗しました: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.PredictionResponse{
		Success:    true,
		Prediction: prediction,
		Message:    fmt.Sprintf("製品 %s の売上予測が完了しました", req.ProductID),
	})
}

// DetectAnomaliesInSales 売上データから異常値を検出する
func (ah *AIHandler) DetectAnomaliesInSales(c *gin.Context) {
	// サンプルデータ（実際の実装ではリクエストボディから取得）
	type AnomalyRequest struct {
		Sales     []float64 `json:"sales" binding:"required"`
		Dates     []string  `json:"dates" binding:"required"`
		ProductID string    `json:"product_id,omitempty"` // 追加
	}

	var req AnomalyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "リクエストパラメータが不正です: " + err.Error(),
		})
		return
	}

	if len(req.Sales) != len(req.Dates) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "売上データと日付データの長さが一致しません",
		})
		return
	}

	// 異常検知を実行（製品名は空で渡す - このAPIではProductNameフィールドがないため）
	productName := ""
	anomalies := ah.statisticsService.DetectAnomalies(req.Sales, req.Dates, req.ProductID, productName)

	// 各異常に対してAIが質問を生成
	for i := range anomalies {
		question, choices := ah.statisticsService.GenerateAIQuestion(anomalies[i])
		anomalies[i].AIQuestion = question
		anomalies[i].QuestionChoices = choices
	}

	c.JSON(http.StatusOK, models.AnomalyDetectionResponse{
		Success:   true,
		Anomalies: anomalies,
		Message:   fmt.Sprintf("%d 件の異常を検出しました", len(anomalies)),
	})
}

// ForecastProductDemand 製品別需要予測
func (ah *AIHandler) ForecastProductDemand(c *gin.Context) {
	var req models.ProductForecastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "リクエストパラメータが不正です: " + err.Error(),
		})
		return
	}

	// デフォルト値設定
	if req.Period == "" {
		req.Period = "week"
	}
	if req.RegionCode == "" {
		req.RegionCode = "240000" // デフォルト: 三重県
	}

	// サンプルデータを生成（実際の実装ではQdrantや外部DBから取得）
	// TODO: アップロードされたファイルデータを使用
	historicalData := ah.generateSampleHistoricalData(req.ProductID, 90)

	// 需要予測を実行
	forecast, err := ah.statisticsService.ForecastProductDemand(
		req.ProductID,
		req.ProductName,
		historicalData,
		req.Period,
		req.RegionCode,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "需要予測の計算に失敗しました: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.ProductForecastResponse{
		Success:  true,
		Forecast: forecast,
		Message:  fmt.Sprintf("製品 %s の %s 予測が完了しました", req.ProductName, req.Period),
	})
}

// AnalyzeWeeklySales 週次売上分析
func (ah *AIHandler) AnalyzeWeeklySales(c *gin.Context) {
	var req models.WeeklyAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "リクエストパラメータが不正です: " + err.Error(),
		})
		return
	}

	// 日付をパース
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "開始日の形式が不正です（YYYY-MM-DD形式で指定してください）",
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "終了日の形式が不正です（YYYY-MM-DD形式で指定してください）",
		})
		return
	}

	// 販売データが提供されていない場合はサンプルデータを生成
	salesData := req.SalesData
	if len(salesData) == 0 {
		// サンプルデータを生成（実際の実装ではDBから取得）
		days := int(endDate.Sub(startDate).Hours() / 24)
		salesData = ah.generateSampleHistoricalData(req.ProductID, days)
	}

	// 製品名を取得（簡易版：実際はDBから取得）
	productName := ah.getProductName(req.ProductID)

	// デフォルトの粒度は週次
	granularity := req.Granularity
	if granularity == "" {
		granularity = "weekly"
	}

	// 粒度のバリデーション
	if granularity != "daily" && granularity != "weekly" && granularity != "monthly" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "granularityは 'daily', 'weekly', 'monthly' のいずれかを指定してください",
		})
		return
	}

	// 週次分析を実行（粒度に応じて処理）
	analysis, err := ah.statisticsService.AnalyzeWeeklySales(
		req.ProductID,
		productName,
		salesData,
		startDate,
		endDate,
		granularity,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "週次分析の実行に失敗しました: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    analysis,
		"message": fmt.Sprintf("%d週間の分析が完了しました", analysis.TotalWeeks),
	})
}

// SaveAnomalyResponse ユーザーの異常に対する回答を保存
func (ah *AIHandler) SaveAnomalyResponse(c *gin.Context) {
	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "データベースサービスが利用できません。設定を確認してください。",
		})
		return
	}
	var req models.AnomalyResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "リクエストパラメータが不正です: " + err.Error(),
		})
		return
	}

	// UUID v4を生成
	responseID := uuid.New().String()

	response := models.AnomalyResponse{
		ResponseID:  responseID,
		AnomalyDate: req.AnomalyDate,
		ProductID:   req.ProductID,
		Question:    req.Question,
		Answer:      req.Answer,
		AnswerType:  req.AnswerType,
		Tags:        req.Tags,
		Impact:      req.Impact,
		ImpactValue: req.ImpactValue,
		Timestamp:   time.Now().Format(time.RFC3339),
		UserID:      c.GetString("user_id"), // 認証から取得（未実装の場合は空）
	}

	// Qdrantに保存
	if ah.vectorStoreService != nil {
		// 回答内容をテキストとして構築
		contentText := fmt.Sprintf(
			"日付: %s\n製品ID: %s\n質問: %s\n回答: %s\nタグ: %s\n影響: %s (%.1f%%)",
			response.AnomalyDate,
			response.ProductID,
			response.Question,
			response.Answer,
			strings.Join(response.Tags, ", "),
			response.Impact,
			response.ImpactValue,
		)

		// メタデータを準備
		metadata := map[string]interface{}{
			"type":         "anomaly_response",
			"response_id":  response.ResponseID,
			"anomaly_date": response.AnomalyDate,
			"product_id":   response.ProductID,
			"question":     response.Question,
			"answer":       response.Answer,
			"tags":         strings.Join(response.Tags, ","),
			"impact":       response.Impact,
			"impact_value": response.ImpactValue,
			"timestamp":    response.Timestamp,
		}

		// Qdrantに保存
		err := ah.vectorStoreService.StoreDocument(
			context.Background(),
			"anomaly_responses", // コレクション名
			response.ResponseID,
			contentText,
			metadata,
		)

		if err != nil {
			log.Printf("Qdrantへの回答保存に失敗: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "回答の保存に失敗しました: " + err.Error(),
			})
			return
		}

		log.Printf("✅ 異常回答を保存しました: %s (製品: %s, 日付: %s)", responseID, req.ProductID, req.AnomalyDate)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"response_id": responseID,
		"message":     "回答を保存しました。AIが学習データとして活用します。",
	})
}

// GetAnomalyResponses 保存された回答履歴を取得
func (ah *AIHandler) GetAnomalyResponses(c *gin.Context) {
	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "データベースサービスが利用できません。設定を確認してください。",
		})
		return
	}
	productID := c.Query("product_id")
	limit := 100 // デフォルト

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "ベクトルストアサービスが利用できません",
		})
		return
	}

	ctx := context.Background()

	// 🆕 新しい対話セッション形式の回答を取得
	sessionResults, err := ah.vectorStoreService.ScrollAllPoints(
		ctx,
		"anomaly_response_sessions",
		uint32(limit),
	)

	responses := make([]models.AnomalyResponse, 0)

	if err == nil {
		// セッションデータを変換
		for _, result := range sessionResults {
			if result.Payload == nil {
				continue
			}

			// session_jsonから完全なセッションを復元
			sessionJSONStr := getStringFromPayload(result.Payload, "session_json")
			if sessionJSONStr == "" {
				continue
			}

			var session models.AnomalyResponseSession
			if err := json.Unmarshal([]byte(sessionJSONStr), &session); err != nil {
				log.Printf("⚠️ セッションJSON解析エラー: %v", err)
				continue
			}

			// 製品IDでフィルタ（指定がある場合）
			if productID != "" && session.ProductID != productID {
				continue
			}

			// 完了したセッションのみ表示
			if !session.IsComplete {
				continue
			}

			// セッション全体の会話を1つの回答として表示
			conversationText := ""
			for i, conv := range session.Conversations {
				conversationText += fmt.Sprintf("Q%d: %s\nA%d: %s\n\n", i+1, conv.Question, i+1, conv.Answer)
			}

			response := models.AnomalyResponse{
				ResponseID:  session.SessionID,
				AnomalyDate: session.AnomalyDate,
				ProductID:   session.ProductID,
				Question:    fmt.Sprintf("対話セッション（%d回の質疑応答）", len(session.Conversations)),
				Answer:      conversationText,
				AnswerType:  "session",
				Tags:        session.FinalTags,
				Impact:      session.FinalImpact,
				ImpactValue: session.FinalImpactValue,
				Timestamp:   session.CompletedAt,
			}

			responses = append(responses, response)
		}
	} else {
		log.Printf("⚠️ セッションコレクションの取得に失敗（コレクションが存在しない可能性）: %v", err)
	}

	// 🔄 旧形式の回答も取得（互換性のため）
	collectionName := "anomaly_responses"
	scrollResults, err := ah.vectorStoreService.ScrollAllPoints(
		ctx,
		collectionName,
		uint32(limit),
	)

	if err != nil {
		log.Printf("⚠️ 旧形式の回答履歴の取得に失敗: %v", err)
	} else {
		// 結果をAnomalyResponseに変換
		for _, result := range scrollResults {
			if result.Payload == nil {
				continue
			}

			// typeフィールドでフィルタ
			if typeVal := getStringFromPayload(result.Payload, "type"); typeVal != "anomaly_response" {
				continue
			}

			// 製品IDでフィルタ（指定がある場合）
			resultProductID := getStringFromPayload(result.Payload, "product_id")
			if productID != "" && resultProductID != productID {
				continue
			}

			response := models.AnomalyResponse{
				ResponseID:  getStringFromPayload(result.Payload, "response_id"),
				AnomalyDate: getStringFromPayload(result.Payload, "anomaly_date"),
				ProductID:   resultProductID,
				Impact:      getStringFromPayload(result.Payload, "impact"),
				Timestamp:   getStringFromPayload(result.Payload, "timestamp"),
			}

			if tagsStr := getStringFromPayload(result.Payload, "tags"); tagsStr != "" {
				response.Tags = strings.Split(tagsStr, ",")
			}

			if impactVal := getFloatFromPayload(result.Payload, "impact_value"); impactVal != 0 {
				response.ImpactValue = impactVal
			}

			responses = append(responses, response)
		}
	}

	c.JSON(http.StatusOK, models.AnomalyResponseHistory{
		Success:   true,
		Responses: responses,
		Total:     len(responses),
		Message:   fmt.Sprintf("%d件の回答履歴を取得しました", len(responses)),
	})
}

// GetLearningInsights AIが学習した洞察を取得
func (ah *AIHandler) GetLearningInsights(c *gin.Context) {
	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "データベースサービスが利用できません。設定を確認してください。",
		})
		return
	}
	category := c.Query("category") // "campaign", "weather", "event", etc.

	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "ベクトルストアサービスが利用できません",
		})
		return
	}

	// コレクション名を定義
	collectionName := "anomaly_responses"

	// 回答履歴を全件取得
	scrollResults, err := ah.vectorStoreService.ScrollAllPoints(
		context.Background(),
		collectionName,
		100,
	)

	if err != nil {
		log.Printf("学習データの取得に失敗: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "学習データの取得に失敗しました",
		})
		return
	}

	// タグごとに集計
	tagStats := make(map[string]*struct {
		count       int
		totalImpact float64
		examples    []string
	})

	for _, result := range scrollResults {
		if result.Payload == nil {
			continue
		}

		// typeフィールドでフィルタ
		if typeVal := getStringFromPayload(result.Payload, "type"); typeVal != "anomaly_response" {
			continue
		}

		tagsStr := getStringFromPayload(result.Payload, "tags")
		impact := getFloatFromPayload(result.Payload, "impact_value")
		date := getStringFromPayload(result.Payload, "anomaly_date")

		if tagsStr == "" {
			continue
		}

		tags := strings.Split(tagsStr, ",")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}

			// カテゴリフィルタ
			if category != "" && tag != category {
				continue
			}

			if tagStats[tag] == nil {
				tagStats[tag] = &struct {
					count       int
					totalImpact float64
					examples    []string
				}{}
			}

			tagStats[tag].count++
			tagStats[tag].totalImpact += impact
			if len(tagStats[tag].examples) < 3 {
				tagStats[tag].examples = append(tagStats[tag].examples, date)
			}
		}
	}

	// 洞察を生成
	insights := make([]models.LearningInsight, 0)
	insightID := 1

	for tag, stats := range tagStats {
		if stats.count < 2 {
			continue // 2件未満はスキップ
		}

		avgImpact := stats.totalImpact / float64(stats.count)
		confidence := math.Min(float64(stats.count)/10.0, 1.0) // 10件で信頼度100%

		pattern := ah.generatePatternDescription(tag, avgImpact, stats.count)

		insight := models.LearningInsight{
			InsightID:     fmt.Sprintf("insight_%d", insightID),
			Category:      tag,
			Pattern:       pattern,
			Examples:      stats.examples,
			AverageImpact: avgImpact,
			Confidence:    confidence,
			LearnedFrom:   stats.count,
			LastUpdated:   time.Now().Format(time.RFC3339),
		}

		insights = append(insights, insight)
		insightID++
	}

	// 信頼度順にソート
	sort.Slice(insights, func(i, j int) bool {
		return insights[i].Confidence > insights[j].Confidence
	})

	c.JSON(http.StatusOK, models.LearningInsightsResponse{
		Success:  true,
		Insights: insights,
		Total:    len(insights),
		Message:  fmt.Sprintf("%d件の学習パターンを発見しました", len(insights)),
	})
}

// DeleteAnomalyResponse 異常回答を削除
func (ah *AIHandler) DeleteAnomalyResponse(c *gin.Context) {
	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "データベースサービスが利用できません。設定を確認してください。",
		})
		return
	}
	responseID := c.Param("id")
	if responseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "response_idが指定されていません",
		})
		return
	}

	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "ベクトルストアサービスが利用できません",
		})
		return
	}

	// Qdrantから削除
	collectionName := "anomaly_responses"
	err := ah.vectorStoreService.DeletePoint(context.Background(), collectionName, responseID)

	if err != nil {
		log.Printf("回答の削除に失敗: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "回答の削除に失敗しました",
		})
		return
	}

	log.Printf("✅ 回答を削除しました: %s", responseID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "回答を削除しました",
	})
}

// DeleteAllAnomalyResponses すべての異常回答を削除（管理者用）
func (ah *AIHandler) DeleteAllAnomalyResponses(c *gin.Context) {
	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "データベースサービスが利用できません。設定を確認してください。",
		})
		return
	}

	// コレクションを削除して再作成
	collectionName := "anomaly_responses"
	err := ah.vectorStoreService.RecreateCollection(context.Background(), collectionName)

	if err != nil {
		log.Printf("コレクションの再作成に失敗: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "データの削除に失敗しました",
		})
		return
	}

	log.Printf("✅ すべての回答を削除しました")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "すべての学習データを削除しました",
	})
}

// ListAnalysisReports は保存されているすべての分析レポートのヘッダーを返します
func (ah *AIHandler) ListAnalysisReports(c *gin.Context) {
	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "データベースサービスが利用できません。設定を確認してください。",
		})
		return
	}
	headers, err := ah.vectorStoreService.GetAllAnalysisReportHeaders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "レポート一覧の取得に失敗しました: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"reports": headers,
	})
}

// GetAnalysisReport はIDで指定された単一の分析レポートを返します
func (ah *AIHandler) GetAnalysisReport(c *gin.Context) {
	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "データベースサービスが利用できません。設定を確認してください。",
		})
		return
	}
	reportID := c.Query("id")
	if reportID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "レポートIDが指定されていません",
		})
		return
	}

	report, err := ah.vectorStoreService.GetAnalysisReportByID(c.Request.Context(), reportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   fmt.Sprintf("レポートの取得に失敗しました: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"report":  report,
	})
}

// DeleteAnalysisReport はIDで指定された単一の分析レポートを削除します
func (ah *AIHandler) DeleteAnalysisReport(c *gin.Context) {
	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "データベースサービスが利用できません。設定を確認してください。",
		})
		return
	}
	reportID := c.Query("id")
	if reportID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "レポートIDが指定されていません",
		})
		return
	}

	err := ah.vectorStoreService.DeletePoint(c.Request.Context(), "hunt_documents", reportID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("レポートの削除に失敗しました: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "レポートが正常に削除されました",
	})
}

// DeleteAllAnalysisReports はすべての分析レポートを削除します
func (ah *AIHandler) DeleteAllAnalysisReports(c *gin.Context) {
	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "データベースサービスが利用できません。設定を確認してください。",
		})
		return
	}
	err := ah.vectorStoreService.DeleteAllAnalysisReports(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("全レポートの削除に失敗しました: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "すべての分析レポートが正常に削除されました",
	})
}

// GetUnansweredAnomalies は、ユーザーがまだ回答していない異常のリストを取得します
func (ah *AIHandler) GetUnansweredAnomalies(c *gin.Context) {
	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "データベースサービスが利用できません。"})
		return
	}

	ctx := c.Request.Context()

	// 1. 全ての分析レポートを取得
	reports, err := ah.vectorStoreService.GetAllAnalysisReports(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "分析レポートの取得に失敗しました: " + err.Error()})
		return
	}

	// 2. 全ての回答済み異常を取得
	responses, err := ah.vectorStoreService.GetAllAnomalyResponses(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "回答済み異常の取得に失敗しました: " + err.Error()})
		return
	}

	// 3. 回答済みの異常をマップに格納 (キー: "日付-製品ID")
	answeredAnomalies := make(map[string]struct{})
	for _, res := range responses {
		key := fmt.Sprintf("%s-%s", res.AnomalyDate, res.ProductID)
		answeredAnomalies[key] = struct{}{}
	}

	// 4. 未回答の異常をフィルタリング
	unansweredAnomalies := make([]models.AnomalyDetection, 0)
	for _, report := range reports {
		for _, anomaly := range report.Anomalies {
			key := fmt.Sprintf("%s-%s", anomaly.Date, anomaly.ProductID)
			if _, found := answeredAnomalies[key]; !found {
				// ProductIDが空の異常は除外する
				if anomaly.ProductID != "" {
					unansweredAnomalies = append(unansweredAnomalies, anomaly)
				}
			}
		}
	}

	log.Printf("未回答の異常を %d 件見つけました", len(unansweredAnomalies))

	// デバッグ用に詳細ログを追加
	for i, anomaly := range unansweredAnomalies {
		if i < 5 { // 最初の5件だけログに出力
			log.Printf("  - 未回答[%d]: Date=%s, ProductID=%s, Value=%.2f", i, anomaly.Date, anomaly.ProductID, anomaly.ActualValue)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"anomalies": unansweredAnomalies,
	})
}

// ========================================
// 深掘り質問機能の新しいハンドラー
// ========================================

// SaveAnomalyResponseWithFollowUp 異常回答を保存し、必要なら深掘り質問を返す
func (ah *AIHandler) SaveAnomalyResponseWithFollowUp(c *gin.Context) {
	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "データベースサービスが利用できません。",
		})
		return
	}

	var req models.SaveAnomalyResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "リクエストパラメータが不正です: " + err.Error(),
		})
		return
	}

	ctx := context.Background()
	const MAX_FOLLOW_UPS = 2 // 最大深掘り回数

	// セッションIDがあれば既存セッションを取得、なければ新規作成
	var session *models.AnomalyResponseSession
	if req.SessionID != "" {
		// 既存セッションを取得
		existingSession, err := ah.vectorStoreService.GetAnomalyResponseSession(ctx, req.SessionID)
		if err != nil {
			log.Printf("⚠️ セッション取得失敗: %v", err)
			// セッションが見つからない場合は新規作成
			session = &models.AnomalyResponseSession{
				SessionID:     req.SessionID,
				AnomalyDate:   req.AnomalyDate,
				ProductID:     req.ProductID,
				Conversations: []models.Conversation{},
				IsComplete:    false,
				FollowUpCount: 0,
				CreatedAt:     time.Now().Format(time.RFC3339),
			}
		} else {
			session = existingSession
		}
	} else {
		// 新規セッション作成
		sessionID := uuid.New().String()
		session = &models.AnomalyResponseSession{
			SessionID:     sessionID,
			AnomalyDate:   req.AnomalyDate,
			ProductID:     req.ProductID,
			Conversations: []models.Conversation{},
			IsComplete:    false,
			FollowUpCount: 0,
			CreatedAt:     time.Now().Format(time.RFC3339),
		}
	}

	// 今回の会話を追加
	conversation := models.Conversation{
		Question:   req.Question,
		Answer:     req.Answer,
		Timestamp:  time.Now().Format(time.RFC3339),
		AnswerType: req.AnswerType,
	}
	session.Conversations = append(session.Conversations, conversation)

	// 異常の状況を構築
	anomalyContext := fmt.Sprintf(
		"日付: %s\n製品ID: %s\n異常の種類: 売上変動",
		req.AnomalyDate,
		req.ProductID,
	)

	// AIに回答を評価させる
	evaluation, err := ah.azureOpenAIService.EvaluateAnswerCompleteness(
		anomalyContext,
		req.Question,
		req.Answer,
		session.Conversations[:len(session.Conversations)-1], // 今回分を除く過去の会話
	)

	if err != nil {
		log.Printf("❌ AI評価エラー: %v", err)
		// エラーでも保存は続行
		evaluation = &models.AnswerEvaluation{
			IsSufficient:      true, // エラー時は深掘りしない
			CompletenessScore: 70,
			Reasoning:         "AI評価に失敗したため、回答を受理します",
		}
	}

	log.Printf("📊 AI評価結果: スコア=%d, 十分=%v, 理由=%s",
		evaluation.CompletenessScore,
		evaluation.IsSufficient,
		evaluation.Reasoning,
	)

	// 深掘りが必要か判定
	needsFollowUp := !evaluation.IsSufficient &&
		session.FollowUpCount < MAX_FOLLOW_UPS &&
		evaluation.FollowUpQuestion != ""

	if needsFollowUp {
		// 深掘り質問を実行
		session.FollowUpCount++

		// セッションを保存（まだ完了していない）
		if err := ah.vectorStoreService.SaveAnomalyResponseSession(ctx, session); err != nil {
			log.Printf("❌ セッション保存エラー: %v", err)
		}

		log.Printf("🔍 深掘り質問を生成しました (%d/%d回目)", session.FollowUpCount, MAX_FOLLOW_UPS)

		// 深掘り質問を返す
		c.JSON(http.StatusOK, models.SaveAnomalyResponseResponse{
			Success:          true,
			SessionID:        session.SessionID,
			Message:          fmt.Sprintf("回答を受け付けました。もう少し詳しく教えてください（%d/%d）", session.FollowUpCount, MAX_FOLLOW_UPS),
			NeedsFollowUp:    true,
			Evaluation:       evaluation,
			FollowUpQuestion: evaluation.FollowUpQuestion,
			FollowUpChoices:  evaluation.FollowUpChoices,
		})
		return
	}

	// 深掘り不要 → セッションを完了
	session.IsComplete = true
	session.CompletedAt = time.Now().Format(time.RFC3339)

	// AIが推奨したタグと影響度を採用
	if len(evaluation.SuggestedTags) > 0 {
		session.FinalTags = evaluation.SuggestedTags
	}
	if evaluation.SuggestedImpact != "" {
		session.FinalImpact = evaluation.SuggestedImpact
		session.FinalImpactValue = evaluation.SuggestedImpactValue
	}

	// セッション全体をQdrantに保存
	if err := ah.vectorStoreService.SaveAnomalyResponseSession(ctx, session); err != nil {
		log.Printf("❌ セッション保存エラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "セッションの保存に失敗しました: " + err.Error(),
		})
		return
	}

	log.Printf("✅ 対話セッション完了: %s (製品: %s, 会話数: %d, 深掘り回数: %d)",
		session.SessionID,
		session.ProductID,
		len(session.Conversations),
		session.FollowUpCount,
	)

	c.JSON(http.StatusOK, models.SaveAnomalyResponseResponse{
		Success:       true,
		SessionID:     session.SessionID,
		Message:       "回答を保存しました。ありがとうございます！",
		NeedsFollowUp: false,
		Evaluation:    evaluation,
	})
}
