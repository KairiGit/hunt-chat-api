package main

import (
	"log"
	"net/http"

	config "hunt-chat-api/configs"
	"hunt-chat-api/pkg/handlers"
	"hunt-chat-api/pkg/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// .envファイルを読み込み
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}

	// 設定の読み込み
	cfg := config.LoadConfig()
	// ログにAPIキーとエンドポイントを出力（デバッグ用）
	log.Printf("DEBUG: Loaded AZURE_OPENAI_API_KEY (first 10 chars): %s...", cfg.AzureOpenAIAPIKey[:min(10, len(cfg.AzureOpenAIAPIKey))])
	log.Printf("DEBUG: Loaded AZURE_OPENAI_ENDPOINT: %s", cfg.AzureOpenAIEndpoint)

	// Ginルーターの初期化
	r := gin.Default()

	// CORSミドルウェアの設定
	r.Use(cors.Default())

	// サービスの初期化
	azureOpenAIService := services.NewAzureOpenAIService(
		cfg.AzureOpenAIEndpoint,
		cfg.AzureOpenAIAPIKey,
		cfg.AzureOpenAIAPIVersion,
		cfg.AzureOpenAIChatDeploymentName,
		cfg.AzureOpenAIEmbeddingDeploymentName,
	)
	vectorStoreService, err := services.NewVectorStoreService(azureOpenAIService, cfg.QdrantURL, cfg.QdrantAPIKey)
	if err != nil {
		log.Printf("FATAL: Failed to initialize VectorStoreService: %v", err)
		// Continue running without vector store for now
	}

	// ハンドラーの初期化
	weatherHandler := handlers.NewWeatherHandler()
	demandForecastHandler := handlers.NewDemandForecastHandler(weatherHandler.GetWeatherService())
	aiHandler := handlers.NewAIHandler(azureOpenAIService, weatherHandler.GetWeatherService(), demandForecastHandler.GetDemandForecastService(), vectorStoreService)

	// 認証ミドルウェア
	authMiddleware := func(apiKey string) gin.HandlerFunc {
		return func(c *gin.Context) {
			// APIキーがデフォルト値の場合は認証をスキップ（ローカル開発を容易にするため）
			if apiKey == "" || apiKey == "default_secret_key" {
				c.Next()
				return
			}

			providedKey := c.GetHeader("X-API-KEY")
			if providedKey != apiKey {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				return
			}
			c.Next()
		}
	}

	// ヘルスチェックエンドポイント
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "HUNT Chat-API",
		})
	})

	// APIバージョン1のルートグループ
	v1 := r.Group("/api/v1")
	v1.Use(authMiddleware(cfg.APIKey)) // ミドルウェアをグループに適用
	{
		v1.GET("/hello", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Hello from HUNT Chat-API!",
			})
		})

		// 気象データAPI
		weather := v1.Group("/weather")
		{
			weather.GET("/test", weatherHandler.TestWeatherAPI)
			weather.GET("/regions", weatherHandler.GetRegionCodes)
			weather.GET("/forecast/:regionCode", weatherHandler.GetForecastData)
			weather.GET("/forecast", weatherHandler.GetForecastData) // デフォルト：東京
			weather.GET("/tokyo", weatherHandler.GetTokyoWeatherData)
			weather.GET("/region/:regionCode", weatherHandler.GetWeatherByRegion)

			// 過去データAPI
			weather.GET("/historical/:regionCode", weatherHandler.GetHistoricalWeatherData)
			weather.GET("/historical", weatherHandler.GetHistoricalWeatherData) // デフォルト：東京
			weather.GET("/historical/:regionCode/date", weatherHandler.GetHistoricalWeatherDataByDate)
			weather.GET("/historical/:regionCode/range", weatherHandler.GetHistoricalWeatherDataRange)
			weather.GET("/historical-range", weatherHandler.GetAvailableHistoricalDataRange)

			// 三重県鈴鹿市専用API
			weather.GET("/suzuka/monthly", weatherHandler.GetSuzukaMonthlyWeatherSummary)
			weather.GET("/analysis/:regionCode", weatherHandler.GetWeatherDataAnalysis)
			weather.GET("/analysis", weatherHandler.GetWeatherDataAnalysis) // デフォルト：三重県
			weather.GET("/trends/:regionCode", weatherHandler.GetWeatherTrendAnalysis)
			weather.GET("/trends", weatherHandler.GetWeatherTrendAnalysis) // デフォルト：三重県
			weather.GET("/category/:regionCode", weatherHandler.GetWeatherDataByCategory)
			weather.GET("/category", weatherHandler.GetWeatherDataByCategory) // デフォルト：三重県
		}

		// 需要予測API
		demand := v1.Group("/demand")
		{
			demand.POST("/forecast", demandForecastHandler.PredictDemand)
			demand.GET("/forecast/suzuka", demandForecastHandler.GetDemandForecastForSuzuka)
			demand.GET("/settings", demandForecastHandler.GetDemandForecastSettings)
			demand.GET("/insights/:regionCode", demandForecastHandler.GetDemandInsights)
			demand.GET("/insights", demandForecastHandler.GetDemandInsights) // デフォルト：三重県
			demand.GET("/analytics/:regionCode", demandForecastHandler.GetDemandAnalytics)
			demand.GET("/analytics", demandForecastHandler.GetDemandAnalytics) // デフォルト：三重県
			demand.GET("/anomalies", demandForecastHandler.DetectAnomalies)    // 異常検知
		}

		// AI統合API
		ai := v1.Group("/ai")
		{
			ai.GET("/capabilities", aiHandler.GetAICapabilities)
			ai.POST("/analyze-weather", aiHandler.AnalyzeWeatherData)
			ai.POST("/demand-insights", aiHandler.GenerateDemandInsights)
			ai.POST("/predict-demand", aiHandler.PredictDemandWithAI)
			ai.POST("/explain-forecast", aiHandler.ExplainForecast)
			ai.GET("/generate-question", aiHandler.GenerateAnomalyQuestion) // 異常から質問を生成
			ai.POST("/chat-input", aiHandler.ChatInput)
			ai.POST("/analyze-file", aiHandler.AnalyzeFile)
			ai.POST("/predict-sales", aiHandler.PredictSales)                    // 売上予測API
			ai.POST("/detect-anomalies", aiHandler.DetectAnomaliesInSales)       // 異常検知API
			ai.POST("/forecast-product", aiHandler.ForecastProductDemand)        // 製品別需要予測API
			ai.POST("/analyze-weekly", aiHandler.AnalyzeWeeklySales)             // 週次分析API
			ai.POST("/anomaly-response", aiHandler.SaveAnomalyResponse)          // 異常への回答保存API
			ai.GET("/anomaly-responses", aiHandler.GetAnomalyResponses)          // 回答履歴取得API
			ai.GET("/learning-insights", aiHandler.GetLearningInsights)          // AI学習洞察取得API
			ai.GET("/analysis-reports", aiHandler.ListAnalysisReports)           // 分析レポート一覧取得API
			ai.DELETE("/analysis-reports", aiHandler.DeleteAllAnalysisReports)   // 全分析レポート削除API
			ai.GET("/analysis-report", aiHandler.GetAnalysisReport)              // 分析レポート詳細取得API
			ai.DELETE("/analysis-report", aiHandler.DeleteAnalysisReport)        // 分析レポート削除API
			ai.DELETE("/anomaly-response/:id", aiHandler.DeleteAnomalyResponse)  // 回答削除API
			ai.DELETE("/anomaly-responses", aiHandler.DeleteAllAnomalyResponses) // 全回答削除API
			ai.GET("/unanswered-anomalies", aiHandler.GetUnansweredAnomalies)    // 未回答の異常を取得
		}
	} // サーバー起動
	log.Println("Starting HUNT Chat-API server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
