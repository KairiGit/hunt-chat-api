package handlers

import (
	"bytes"
	"context"
	"encoding/csv"
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
	"github.com/qdrant/go-client/qdrant"
	"github.com/xuri/excelize/v2"
)

// AIHandler AI統合ハンドラー
type AIHandler struct {
	azureOpenAIService    *services.AzureOpenAIService
	weatherService        *services.WeatherService
	demandForecastService *services.DemandForecastService
	vectorStoreService    *services.VectorStoreService
	statisticsService     *services.StatisticsService
}

// NewAIHandler 新しいAI統合ハンドラーを作成
func NewAIHandler(azureOpenAIService *services.AzureOpenAIService, weatherService *services.WeatherService, demandForecastService *services.DemandForecastService, vectorStoreService *services.VectorStoreService) *AIHandler {
	return &AIHandler{
		azureOpenAIService:    azureOpenAIService,
		weatherService:        weatherService,
		demandForecastService: demandForecastService,
		vectorStoreService:    vectorStoreService,
		statisticsService:     services.NewStatisticsService(weatherService),
	}
}

// findIndex finds the index of the first candidate in a slice
func findIndex(slice []string, candidates ...string) int {
	for _, candidate := range candidates {
		for i, item := range slice {
			if strings.EqualFold(item, candidate) {
				return i
			}
		}
	}
	return -1
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// AnalyzeFile: Logic-based file analysis with monthly aggregation
func (ah *AIHandler) AnalyzeFile(c *gin.Context) {
	c.Request.ParseMultipartForm(10 << 20) // 10MB limit

	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ファイルの取得に失敗しました。"})
		return
	}
	defer file.Close()

	var rows [][]string
	fileName := fileHeader.Filename

	if strings.HasSuffix(strings.ToLower(fileName), ".xlsx") {
		f, err := excelize.OpenReader(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Excelファイルの読み込みに失敗しました。"})
			return
		}
		rows, err = f.GetRows(f.GetSheetName(0))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Excelシートの行取得に失敗しました。"})
			return
		}
	} else if strings.HasSuffix(strings.ToLower(fileName), ".csv") {
		r := csv.NewReader(file)
		rows, err = r.ReadAll()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "CSVファイルの解析に失敗しました。"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "サポートされていないファイル形式です。.xlsxまたは.csvをアップロードしてください。"})
		return
	}

	if len(rows) < 2 { // Header + at least one data row
		c.JSON(http.StatusBadRequest, gin.H{"error": "ファイルにはヘッダー行と少なくとも1行のデータが必要です。"})
		return
	}

	header := rows[0]
	dataRows := rows[1:]

	dateColIdx := findIndex(header, "date", "日付")
	productColIdx := findIndex(header, "product", "product_id", "商品", "商品ID", "製品", "製品名", "製品ID")
	salesColIdx := findIndex(header, "sales", "quantity", "販売数", "数量")

	var missingCols []string
	if dateColIdx == -1 {
		missingCols = append(missingCols, "日付")
	}
	if productColIdx == -1 {
		missingCols = append(missingCols, "製品")
	}
	if salesColIdx == -1 {
		missingCols = append(missingCols, "販売数")
	}

	if len(missingCols) > 0 {
		errMsg := fmt.Sprintf("必要な列が見つかりませんでした: %s。ファイルのヘッダー行を確認してください。", strings.Join(missingCols, ", "))
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}

	type monthlySales struct {
		TotalSales int
		DataPoints int
	}
	productSales := make(map[string]map[time.Month]*monthlySales)

	for _, row := range dataRows {
		if len(row) > dateColIdx && len(row) > productColIdx && len(row) > salesColIdx {
			dateStr := row[dateColIdx]
			product := row[productColIdx]
			salesStr := row[salesColIdx]

			var t time.Time
			t, err = time.Parse("2006-01-02", dateStr)
			if err != nil {
				t, _ = time.Parse("2006/1/2", dateStr)
			}

			sales, convErr := strconv.Atoi(salesStr)
			if product != "" && t != (time.Time{}) && convErr == nil {
				month := t.Month()
				if productSales[product] == nil {
					productSales[product] = make(map[time.Month]*monthlySales)
				}
				if productSales[product][month] == nil {
					productSales[product][month] = &monthlySales{}
				}
				productSales[product][month].TotalSales += sales
				productSales[product][month].DataPoints++
			}
		}
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("ファイル概要:\n- ファイル名: %s\n- 総データ行数: %d\n- 列名: %s\n\n", fileName, len(dataRows), strings.Join(header, ", ")))

	if len(productSales) > 0 {
		summary.WriteString("製品別の月次売上分析:\n")
		products := make([]string, 0, len(productSales))
		for p := range productSales {
			products = append(products, p)
		}
		sort.Strings(products)

		for _, product := range products {
			monthlyData := productSales[product]
			var total, monthCount int
			var bestMonth, worstMonth time.Month
			minSales, maxSales := -1, -1

			for month, salesData := range monthlyData {
				avgSales := salesData.TotalSales / salesData.DataPoints
				total += avgSales
				monthCount++
				if minSales == -1 || avgSales < minSales {
					minSales = avgSales
					worstMonth = month
				}
				if maxSales == -1 || avgSales > maxSales {
					maxSales = avgSales
					bestMonth = month
				}
			}

			summary.WriteString(fmt.Sprintf("- 製品: %s\n", product))
			if monthCount > 0 {
				summary.WriteString(fmt.Sprintf("  - 平均月間売上: %d個\n", total/monthCount))
				summary.WriteString(fmt.Sprintf("  - ベスト月: %s (%d個)\n", bestMonth.String(), maxSales))
				summary.WriteString(fmt.Sprintf("  - ワースト月: %s (%d個)\n", worstMonth.String(), minSales))
			}
		}
		summary.WriteString("\n")
	}

	topN := 5
	dataRowsSample := rows[1:int(math.Min(float64(topN+1), float64(len(rows))))]
	toString := func(sample [][]string) string {
		var b bytes.Buffer
		w := csv.NewWriter(&b)
		w.Write(header)
		w.WriteAll(sample)
		return b.String()
	}
	if len(dataRowsSample) > 0 {
		summary.WriteString("データサンプル:\n")
		summary.WriteString(toString(dataRowsSample))
	}

	// === 目標① 統計分析の実行 ===
	// 販売データを WeatherSalesData 形式に変換
	var salesData []models.WeatherSalesData
	for _, row := range dataRows {
		if len(row) > dateColIdx && len(row) > productColIdx && len(row) > salesColIdx {
			dateStr := row[dateColIdx]
			product := row[productColIdx]
			salesStr := row[salesColIdx]

			var t time.Time
			t, _ = time.Parse("2006-01-02", dateStr)
			if t == (time.Time{}) {
				t, _ = time.Parse("2006/1/2", dateStr)
			}

			sales, convErr := strconv.ParseFloat(salesStr, 64)
			if product != "" && t != (time.Time{}) && convErr == nil {
				salesData = append(salesData, models.WeatherSalesData{
					Date:      t.Format("2006-01-02"),
					ProductID: product,
					Sales:     sales,
				})
			}
		}
	}

	// デフォルトの地域コード（三重県）
	regionCode := "240000"
	if rc := c.Query("region_code"); rc != "" {
		regionCode = rc
	}

	log.Printf("📂 ファイル分析開始: %s, 販売データ件数: %d, 地域コード: %s", fileName, len(salesData), regionCode)

	// 統計分析を実行
	var analysisReport *models.AnalysisReport
	if len(salesData) > 0 {
		// 日付範囲を確認
		if len(salesData) > 0 {
			log.Printf("📅 販売データの最初の日付: %s, 最後の日付: %s", salesData[0].Date, salesData[len(salesData)-1].Date)
		}

		// statisticsServiceが初期化されているか確認
		if ah.statisticsService == nil {
			log.Printf("❌ StatisticsService が初期化されていません")
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"summary": summary.String(),
				"error":   "統計分析サービスが利用できません",
			})
			return
		}

		// AI分析を呼び出し（エラーハンドリング強化）
		var aiInsights string
		if ah.azureOpenAIService != nil {
			insights, aiErr := ah.azureOpenAIService.ProcessChatWithContext(
				"以下の販売データを分析して、需要予測に役立つ洞察を提供してください。",
				summary.String(),
			)
			if aiErr != nil {
				aiInsights = "AI分析は利用できませんでした。"
				log.Printf("⚠️ AI分析エラー: %v", aiErr)
			} else {
				aiInsights = insights
			}
		} else {
			aiInsights = "AIサービスが初期化されていません。"
			log.Printf("⚠️ AIサービスが nil です")
		}

		// 統計レポート作成（エラーハンドリング強化）
		report, err := ah.statisticsService.CreateAnalysisReport(
			fileName,
			salesData,
			regionCode,
			aiInsights,
		)
		if err != nil {
			log.Printf("❌ 統計レポート作成エラー: %v", err)
			// エラーが発生してもサマリーは返す
			// 診断情報を含める
			diagnosticInfo := fmt.Sprintf(
				"販売データ件数: %d件, 気象データ取得: 失敗, エラー詳細: %v",
				len(salesData),
				err,
			)
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"summary": summary.String(),
				"error":   fmt.Sprintf("統計分析でエラーが発生しました。%s", diagnosticInfo),
			})
			return
		} else {
			analysisReport = report

			// レポート内容をログ出力（デバッグ用）
			log.Printf("📊 分析レポート作成完了:")
			log.Printf("  - レポートID: %s", report.ReportID)
			log.Printf("  - 日付範囲: %s", report.DateRange)
			log.Printf("  - 気象データマッチ: %d件", report.WeatherMatches)
			log.Printf("  - 相関分析結果: %d件", len(report.Correlations))
			for i, corr := range report.Correlations {
				log.Printf("    [%d] %s: %.3f (%s)", i+1, corr.Factor, corr.CorrelationCoef, corr.Interpretation)
			}
			if report.Regression != nil {
				log.Printf("  - 回帰分析: %s", report.Regression.Description)
			}
			log.Printf("  - 推奨事項: %d件", len(report.Recommendations))

			// === 目標② 分析結果をQdrantに保存 ===
			go func() {
				ctx := context.Background()
				reportJSON, _ := json.Marshal(report)
				err := ah.vectorStoreService.SaveAnalysisReport(ctx, string(reportJSON), "sales_weather_analysis")
				if err != nil {
					log.Printf("分析レポートのQdrant保存に失敗: %v", err)
				} else {
					log.Printf("分析レポート %s をQdrantに保存しました", report.ReportID)
				}
			}()
		}
	}

	// レスポンスに統計分析結果を含める
	response := gin.H{
		"success": true,
		"summary": summary.String(),
	}
	if analysisReport != nil {
		response["analysis_report"] = analysisReport
		log.Printf("✅ レスポンスに analysis_report を含めました")
	} else {
		log.Printf("⚠️ analysisReport が nil のため、レスポンスに含まれていません")
	}

	c.JSON(http.StatusOK, response)
}

type ChatInputRequest struct {
	ChatMessage string `json:"chat_message"`
	Context     string `json:"context,omitempty"`
	SessionID   string `json:"session_id,omitempty"` // セッションID（会話の継続性）
	UserID      string `json:"user_id,omitempty"`    // ユーザーID（履歴の紐付け）
}

func (ah *AIHandler) ChatInput(c *gin.Context) {
	var req ChatInputRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "リクエストの形式が正しくありません: " + err.Error()})
		return
	}
	if req.ChatMessage == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "チャットメッセージが必要です。"})
		return
	}

	// セッションIDが指定されていない場合は新規生成
	if req.SessionID == "" {
		req.SessionID = uuid.New().String()
	}

	ctx := c.Request.Context()

	// メタデータを抽出（意図やキーワード）
	intent, keywords, _ := ah.azureOpenAIService.ExtractMetadataFromMessage(req.ChatMessage)

	// ユーザーメッセージをチャット履歴として保存
	userEntry := models.ChatHistoryEntry{
		ID:        uuid.New().String(),
		SessionID: req.SessionID,
		UserID:    req.UserID,
		Role:      "user",
		Message:   req.ChatMessage,
		Context:   req.Context,
		Timestamp: time.Now().Format(time.RFC3339),
		Tags:      keywords,
		Metadata: models.Metadata{
			Intent:        intent,
			TopicKeywords: keywords,
		},
		CreatedAt: time.Now(),
	}

	// 非同期でチャット履歴を保存
	go func() {
		if err := ah.vectorStoreService.SaveChatHistory(context.Background(), userEntry); err != nil {
			log.Printf("ユーザーメッセージの履歴保存に失敗: %v", err)
		} else {
			log.Printf("✅ ユーザーメッセージを履歴に保存: SessionID=%s", req.SessionID)
		}
	}()

	// RAG: 類似した過去の会話を検索（チャット履歴から）
	var ragContext strings.Builder
	var relevantHistoryTexts []string
	var contextSources []string

	if req.Context != "" {
		ragContext.WriteString(req.Context) // ファイル分析のコンテキストを維持
		contextSources = append(contextSources, "現在のファイル分析")
	}

	// 🔍 過去のチャット履歴から関連する会話を検索
	chatHistory, err := ah.vectorStoreService.SearchChatHistory(ctx, req.ChatMessage, "", req.UserID, 3)
	if err != nil {
		log.Printf("チャット履歴検索に失敗: %v", err)
	} else if len(chatHistory) > 0 {
		ragContext.WriteString("\n\n## 過去の関連する会話履歴:\n")
		for i, entry := range chatHistory {
			historyText := fmt.Sprintf("[%s] %s: %s", entry.Timestamp, entry.Role, entry.Message)
			relevantHistoryTexts = append(relevantHistoryTexts, historyText)
			ragContext.WriteString(fmt.Sprintf("%d. %s (関連度: %.2f)\n", i+1, historyText, entry.Metadata.RelevanceScore))
			contextSources = append(contextSources, fmt.Sprintf("過去の会話 (%s)", entry.Timestamp))
		}
		log.Printf("📚 %d件の関連する過去の会話を取得しました", len(chatHistory))
	}

	// 一般的なドキュメント検索（hunt_chat_documentsから）
	searchResults, err := ah.vectorStoreService.Search(ctx, req.ChatMessage, 2)
	if err != nil {
		log.Printf("ベクトル検索に失敗: %v", err)
	} else if len(searchResults) > 0 {
		ragContext.WriteString("\n\n## 関連するドキュメント:\n")
		for _, point := range searchResults {
			if textPayload, ok := point.Payload["text"]; ok {
				if text, ok := textPayload.GetKind().(*qdrant.Value_StringValue); ok {
					ragContext.WriteString(fmt.Sprintf("- %s (類似度: %.2f)\n", text.StringValue, point.Score))
					contextSources = append(contextSources, "ナレッジベース")
				}
			}
		}
	}

	// 分析レポートを検索（質問が分析関連の場合）
	if strings.Contains(strings.ToLower(req.ChatMessage), "分析") ||
		strings.Contains(strings.ToLower(req.ChatMessage), "相関") ||
		strings.Contains(strings.ToLower(req.ChatMessage), "ファイル") ||
		strings.Contains(strings.ToLower(req.ChatMessage), "レポート") {

		analysisResults, err := ah.vectorStoreService.SearchAnalysisReports(ctx, req.ChatMessage, 2)
		if err != nil {
			log.Printf("分析レポート検索に失敗: %v", err)
		} else if len(analysisResults) > 0 {
			ragContext.WriteString("\n\n## 関連する過去の分析レポート:\n")
			for _, point := range analysisResults {
				if textPayload, ok := point.Payload["text"]; ok {
					if text, ok := textPayload.GetKind().(*qdrant.Value_StringValue); ok {
						var report models.AnalysisReport
						if json.Unmarshal([]byte(text.StringValue), &report) == nil {
							ragContext.WriteString(fmt.Sprintf("\n### レポート: %s\n", report.FileName))
							ragContext.WriteString(fmt.Sprintf("- 分析日: %s\n", report.AnalysisDate))
							ragContext.WriteString(fmt.Sprintf("- データ点数: %d\n", report.DataPoints))
							ragContext.WriteString(fmt.Sprintf("- サマリー:\n%s\n", report.Summary))
							if len(report.Correlations) > 0 {
								ragContext.WriteString("- 相関分析結果:\n")
								for _, corr := range report.Correlations {
									ragContext.WriteString(fmt.Sprintf("  * %s: %.3f (%s)\n",
										corr.Factor, corr.CorrelationCoef, corr.Interpretation))
								}
							}
							if report.Regression != nil {
								ragContext.WriteString(fmt.Sprintf("- 回帰分析: %s\n", report.Regression.Description))
							}
							contextSources = append(contextSources, fmt.Sprintf("分析レポート (%s)", report.FileName))
						}
					}
				}
			}
		}
	}

	// 🤖 AIに応答を生成させる（過去の履歴を活用）
	aiResponse, err := ah.azureOpenAIService.ProcessChatWithHistory(
		req.ChatMessage,
		ragContext.String(),
		relevantHistoryTexts,
	)
	if err != nil {
		log.Printf("AI処理エラー詳細: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI処理中にエラーが発生しました: " + err.Error()})
		return
	}

	// AIの応答をチャット履歴として保存
	assistantEntry := models.ChatHistoryEntry{
		ID:        uuid.New().String(),
		SessionID: req.SessionID,
		UserID:    req.UserID,
		Role:      "assistant",
		Message:   aiResponse,
		Context:   req.Context,
		Timestamp: time.Now().Format(time.RFC3339),
		Tags:      keywords,
		Metadata: models.Metadata{
			Intent:        intent,
			TopicKeywords: keywords,
		},
		CreatedAt: time.Now(),
	}

	// 非同期でAI応答を履歴に保存
	go func() {
		if err := ah.vectorStoreService.SaveChatHistory(context.Background(), assistantEntry); err != nil {
			log.Printf("AI応答の履歴保存に失敗: %v", err)
		} else {
			log.Printf("✅ AI応答を履歴に保存: SessionID=%s", req.SessionID)
		}
	}()

	// レスポンスを返す（履歴情報を含む）
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"response": gin.H{
			"text":               aiResponse,
			"session_id":         req.SessionID,
			"relevant_history":   relevantHistoryTexts,
			"context_sources":    contextSources,
			"conversation_count": len(chatHistory),
		},
	})
}

type AnalyzeWeatherDataRequest struct {
	RegionCode string `json:"region_code"`
	Days       int    `json:"days"`
}

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
	question, err := ah.azureOpenAIService.GenerateQuestionFromAnomaly(targetAnomaly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AIからの質問生成に失敗しました: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "異常を検知し、質問を生成しました。", "question": question, "source_anomaly": targetAnomaly})
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
		Sales []float64 `json:"sales" binding:"required"`
		Dates []string  `json:"dates" binding:"required"`
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

	// 異常検知を実行
	anomalies := ah.statisticsService.DetectAnomalies(req.Sales, req.Dates)

	// 各異常に対してAIが質問を生成
	for i := range anomalies {
		anomalies[i].AIQuestion = ah.statisticsService.GenerateAIQuestion(anomalies[i])
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

	// 週次分析を実行
	analysis, err := ah.statisticsService.AnalyzeWeeklySales(
		req.ProductID,
		productName,
		salesData,
		startDate,
		endDate,
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

// getProductName 製品IDから製品名を取得（簡易版）
func (ah *AIHandler) getProductName(productID string) string {
	productNames := map[string]string{
		"P001": "製品A",
		"P002": "製品B",
		"P003": "製品C",
		"P004": "製品D",
		"P005": "製品E",
	}

	if name, exists := productNames[productID]; exists {
		return name
	}
	return "不明な製品"
}

// generateSampleHistoricalData サンプルの履歴データを生成（テスト用）
func (ah *AIHandler) generateSampleHistoricalData(productID string, days int) []models.SalesDataPoint {
	data := make([]models.SalesDataPoint, days)
	baseDate := time.Now().AddDate(0, 0, -days)
	baseSales := 100.0

	for i := 0; i < days; i++ {
		date := baseDate.AddDate(0, 0, i)
		dayOfWeek := []string{"日", "月", "火", "水", "木", "金", "土"}[date.Weekday()]

		// 曜日効果
		weekdayMultiplier := 1.0
		switch date.Weekday() {
		case time.Saturday, time.Sunday:
			weekdayMultiplier = 1.3 // 週末は30%増
		case time.Friday:
			weekdayMultiplier = 1.15 // 金曜は15%増
		}

		// 季節効果
		seasonalMultiplier := 1.0
		month := date.Month()
		if month >= 6 && month <= 8 {
			seasonalMultiplier = 1.2 // 夏は20%増
		} else if month == 12 || month <= 2 {
			seasonalMultiplier = 0.9 // 冬は10%減
		}

		// トレンド効果（徐々に増加）
		trendEffect := 1.0 + (float64(i) / float64(days) * 0.1)

		// ランダムノイズ
		noise := 1.0 + (float64(i%10)-5)/50.0

		sales := baseSales * weekdayMultiplier * seasonalMultiplier * trendEffect * noise

		data[i] = models.SalesDataPoint{
			Date:        date.Format("2006-01-02"),
			Sales:       sales,
			Temperature: 15.0 + float64(month)*1.5 + float64(i%10-5)*0.5,
			DayOfWeek:   dayOfWeek,
		}
	}

	return data
}

// SaveAnomalyResponse ユーザーの異常に対する回答を保存
func (ah *AIHandler) SaveAnomalyResponse(c *gin.Context) {
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

	// コレクションが存在することを確認
	collectionName := "anomaly_responses"

	// Qdrantから検索
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "type",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: "anomaly_response",
							},
						},
					},
				},
			},
		},
	}

	// 製品IDでフィルタ
	if productID != "" {
		filter.Must = append(filter.Must, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key: "product_id",
					Match: &qdrant.Match{
						MatchValue: &qdrant.Match_Keyword{
							Keyword: productID,
						},
					},
				},
			},
		})
	}

	// ダミークエリで検索（フィルタのみ適用）
	searchResults, err := ah.vectorStoreService.SearchWithFilter(
		context.Background(),
		collectionName,
		"異常", // ダミーテキスト
		uint64(limit),
		filter,
	)

	if err != nil {
		log.Printf("回答履歴の取得に失敗: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "回答履歴の取得に失敗しました",
		})
		return
	}

	// 結果をAnomalyResponseに変換
	responses := make([]models.AnomalyResponse, 0)
	for _, result := range searchResults {
		if result.Payload == nil {
			continue
		}

		response := models.AnomalyResponse{
			ResponseID:  getStringFromPayload(result.Payload, "response_id"),
			AnomalyDate: getStringFromPayload(result.Payload, "anomaly_date"),
			ProductID:   getStringFromPayload(result.Payload, "product_id"),
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

	c.JSON(http.StatusOK, models.AnomalyResponseHistory{
		Success:   true,
		Responses: responses,
		Total:     len(responses),
		Message:   fmt.Sprintf("%d件の回答履歴を取得しました", len(responses)),
	})
}

// GetLearningInsights AIが学習した洞察を取得
func (ah *AIHandler) GetLearningInsights(c *gin.Context) {
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

	// 回答履歴を取得
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "type",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: "anomaly_response",
							},
						},
					},
				},
			},
		},
	}

	searchResults, err := ah.vectorStoreService.SearchWithFilter(
		context.Background(),
		collectionName,
		"パターン分析",
		100,
		filter,
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

	for _, result := range searchResults {
		if result.Payload == nil {
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

// generatePatternDescription パターンの説明を生成
func (ah *AIHandler) generatePatternDescription(tag string, avgImpact float64, count int) string {
	impactStr := "影響"
	if avgImpact > 0 {
		impactStr = fmt.Sprintf("平均+%.1f%%の需要増加", avgImpact)
	} else if avgImpact < 0 {
		impactStr = fmt.Sprintf("平均%.1f%%の需要減少", math.Abs(avgImpact))
	}

	return fmt.Sprintf("%sが発生した際、%sの傾向があります（%d件の実績から学習）", tag, impactStr, count)
}

// ヘルパー関数: Payloadから文字列を取得
func getStringFromPayload(payload map[string]*qdrant.Value, key string) string {
	if val, ok := payload[key]; ok && val != nil {
		if strVal := val.GetStringValue(); strVal != "" {
			return strVal
		}
	}
	return ""
}

// ヘルパー関数: Payloadから数値を取得
func getFloatFromPayload(payload map[string]*qdrant.Value, key string) float64 {
	if val, ok := payload[key]; ok && val != nil {
		if doubleVal := val.GetDoubleValue(); doubleVal != 0 {
			return doubleVal
		}
		if intVal := val.GetIntegerValue(); intVal != 0 {
			return float64(intVal)
		}
	}
	return 0
}
