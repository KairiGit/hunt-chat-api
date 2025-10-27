package handler

// Build version: 2025-10-16-delete-response-v1
// Vercel: Added delete endpoints for anomaly responses

import (
	"log"
	"net/http"
	"sync"

	config "hunt-chat-api/configs"
	"hunt-chat-api/pkg/handlers"
	"hunt-chat-api/pkg/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var (
	app  *gin.Engine
	once sync.Once
)

// setupApp はGinアプリケーションを初期化します。
// サーバーレス環境では、リクエストごとに初期化が走らないようsync.Onceで一度だけ実行します。
func setupApp() *gin.Engine {
	once.Do(func() {
		log.Printf("🟢 [setupApp] Initializing Gin application - anomaly-save-fix-v1")

		// .envファイルはVercelの環境変数設定から読み込まれるため、ここではgodotenvを呼び出しません。
		cfg := config.LoadConfig()

		log.Printf("🟢 [setupApp] Config loaded successfully")

		// Ginルーターの初期化
		r := gin.Default()

		// サービスの初期化
		monitoringService := services.NewMonitoringService()
		azureOpenAIService := services.NewAzureOpenAIService(
			cfg.AzureOpenAIEndpoint,
			cfg.AzureOpenAIAPIKey,
			cfg.AzureOpenAIAPIVersion,
			cfg.AzureOpenAIChatDeploymentName,
			cfg.AzureOpenAIEmbeddingDeploymentName,
		)
		vectorStoreService, err := services.NewVectorStoreService(azureOpenAIService, cfg.QdrantURL, cfg.QdrantAPIKey)
		if err != nil {
			log.Printf("FATAL: Failed to initialize VectorStoreService in Vercel function: %v", err)
		}

		// ハンドラーの初期化
		weatherHandler := handlers.NewWeatherHandler()
		demandForecastHandler := handlers.NewDemandForecastHandler(weatherHandler.GetWeatherService())
		economicSymbolMapping := map[string]string{
			"NIKKEI": "moc/nikkei_daily.csv",
		}
		economicService := services.NewEconomicService(".", economicSymbolMapping)
		economicHandler := handlers.NewEconomicHandler(vectorStoreService)
		aiHandler := handlers.NewAIHandler(azureOpenAIService, weatherHandler.GetWeatherService(), economicService, demandForecastHandler.GetDemandForecastService(), vectorStoreService)
		adminHandler := handlers.NewAdminHandler(cfg)
		monitoringHandler := handlers.NewMonitoringHandler(monitoringService)

		// ミドルウェアの登録
		r.Use(monitoringService.LoggingMiddleware())
		config := cors.DefaultConfig()
		config.AllowAllOrigins = true
		r.Use(cors.New(config))

		// 認証ミドルウェア
		authMiddleware := func(apiKey string) gin.HandlerFunc {
			return func(c *gin.Context) {
				if apiKey == "" || apiKey == "default_secret_key" {
					c.Next()
					return
				}
				providedKey := c.GetHeader("X-API-KEY")
				if providedKey == "" {
					log.Printf("⚠️ [認証] API Keyが提供されていません。一時的に許可します。")
					c.Next()
					return
				}
				if providedKey != apiKey {
					log.Printf("❌ [認証] 無効なAPI Key: %s", providedKey)
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
					return
				}
				c.Next()
			}
		}

		// ヘルスチェックエンドポイント
		r.GET("/health", handlers.HealthCheck)

		// APIルートの定義
		v1 := r.Group("/api/v1")
		v1.Use(authMiddleware(cfg.APIKey))
		{
			v1.GET("/hello", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Hello from Vercel!"})
			})

			// 管理者向けAPI
			admin := v1.Group("/admin")
			{
				admin.GET("/health-status", adminHandler.GetHealthStatus)
				admin.POST("/maintenance/start", adminHandler.StartMaintenance)
				admin.POST("/maintenance/stop", adminHandler.StopMaintenance)
			}

			// モニタリングAPI
			monitoring := v1.Group("/monitoring")
			{
				monitoring.GET("/logs", monitoringHandler.GetLogs)
			}

			// 気象データAPI
			weather := v1.Group("/weather")
			{
				weather.GET("/test", weatherHandler.TestWeatherAPI)
				weather.GET("/regions", weatherHandler.GetRegionCodes)
				weather.GET("/forecast/:regionCode", weatherHandler.GetForecastData)
				weather.GET("/forecast", weatherHandler.GetForecastData)
				weather.GET("/tokyo", weatherHandler.GetTokyoWeatherData)
				weather.GET("/region/:regionCode", weatherHandler.GetWeatherByRegion)
				weather.GET("/historical/:regionCode", weatherHandler.GetHistoricalWeatherData)
				weather.GET("/historical", weatherHandler.GetHistoricalWeatherData)
				weather.GET("/historical/:regionCode/date", weatherHandler.GetHistoricalWeatherDataByDate)
				weather.GET("/historical/:regionCode/range", weatherHandler.GetHistoricalWeatherDataRange)
				weather.GET("/historical-range", weatherHandler.GetAvailableHistoricalDataRange)
				weather.GET("/suzuka/monthly", weatherHandler.GetSuzukaMonthlyWeatherSummary)
				weather.GET("/analysis/:regionCode", weatherHandler.GetWeatherDataAnalysis)
				weather.GET("/analysis", weatherHandler.GetWeatherDataAnalysis)
				weather.GET("/trends/:regionCode", weatherHandler.GetWeatherTrendAnalysis)
				weather.GET("/trends", weatherHandler.GetWeatherTrendAnalysis)
				weather.GET("/category/:regionCode", weatherHandler.GetWeatherDataByCategory)
				weather.GET("/category", weatherHandler.GetWeatherDataByCategory)
			}

			// 需要予測API
			demand := v1.Group("/demand")
			{
				demand.POST("/forecast", demandForecastHandler.PredictDemand)
				demand.GET("/forecast/suzuka", demandForecastHandler.GetDemandForecastForSuzuka)
				demand.GET("/settings", demandForecastHandler.GetDemandForecastSettings)
				demand.GET("/insights/:regionCode", demandForecastHandler.GetDemandInsights)
				demand.GET("/insights", demandForecastHandler.GetDemandInsights)
				demand.GET("/analytics/:regionCode", demandForecastHandler.GetDemandAnalytics)
				demand.GET("/analytics", demandForecastHandler.GetDemandAnalytics)
				demand.GET("/anomalies", demandForecastHandler.DetectAnomalies)
			}

			// AI統合API
			ai := v1.Group("/ai")
			{
				ai.GET("/capabilities", aiHandler.GetAICapabilities)
				ai.POST("/analyze-weather", aiHandler.AnalyzeWeatherData)
				ai.POST("/demand-insights", aiHandler.GenerateDemandInsights)
				ai.POST("/predict-demand", aiHandler.PredictDemandWithAI)
				ai.POST("/explain-forecast", aiHandler.ExplainForecast)
				ai.GET("/generate-question", aiHandler.GenerateAnomalyQuestion)
				ai.POST("/chat-input", aiHandler.ChatInput)
				ai.POST("/analyze-file", func(c *gin.Context) {
					log.Printf("🟢 [api/index.go] /analyze-file エンドポイント呼び出し - Build: 2025-10-16-anomaly-save-fix-v1")
					c.Header("X-Backend-Version", "2025-10-16-anomaly-save-fix-v1")
					c.Header("X-Handler-Called", "true")
					aiHandler.AnalyzeFile(c)
				})
				ai.POST("/predict-sales", aiHandler.PredictSales)
				ai.POST("/forecast-product", aiHandler.ForecastProductDemand)
				ai.POST("/analyze-weekly", aiHandler.AnalyzeWeeklySales)
				ai.POST("/detect-anomalies", aiHandler.DetectAnomaliesInSales)
				ai.POST("/anomaly-response", aiHandler.SaveAnomalyResponse)
				ai.POST("/anomaly-response-with-followup", aiHandler.SaveAnomalyResponseWithFollowUp)
				ai.GET("/anomaly-responses", aiHandler.GetAnomalyResponses)
				ai.DELETE("/anomaly-response/:id", aiHandler.DeleteAnomalyResponse)
				ai.DELETE("/anomaly-responses", aiHandler.DeleteAllAnomalyResponses)
				ai.GET("/learning-insights", aiHandler.GetLearningInsights)
				ai.GET("/analysis-reports", aiHandler.ListAnalysisReports)
				ai.GET("/analysis-report", aiHandler.GetAnalysisReport)
				ai.DELETE("/analysis-report", aiHandler.DeleteAnalysisReport)
				ai.DELETE("/analysis-reports", aiHandler.DeleteAllAnalysisReports)
				ai.GET("/unanswered-anomalies", aiHandler.GetUnansweredAnomalies)
			}

			// 経済/金融データAPI（CSV疑似yfinance）
			econ := v1.Group("/econ")
			{
				econ.GET("/series", economicHandler.GetSeries)
				econ.GET("/sales/series", economicHandler.GetSalesSeries)
				econ.GET("/returns", economicHandler.GetReturns)
				econ.POST("/register", economicHandler.RegisterSymbol)
				econ.POST("/import", economicHandler.ImportCSV)
				econ.POST("/lagged-correlation", economicHandler.AnalyzeLaggedCorrelation)
				econ.POST("/sales/import", economicHandler.ImportSales)
				econ.POST("/sales/lagged-correlation", economicHandler.AnalyzeProductEconLagged)
				econ.POST("/sales/lagged-correlation/windowed", economicHandler.AnalyzeWindowedLag)
				econ.POST("/sales/granger", economicHandler.GrangerCausality)
				econ.POST("/aggregate", economicHandler.AggregateEconomic)
				econ.POST("/sales/aggregate", economicHandler.AggregateSales)
			}
		}

		app = r
	})
	return app
} // Handler はVercelからのすべてのリクエストを処理するエントリーポイントです。
func Handler(w http.ResponseWriter, r *http.Request) {
	// デバッグ: リクエストの詳細をログ出力
	log.Printf("🔵 [Handler] Request received: %s %s", r.Method, r.URL.Path)
	log.Printf("🔵 [Handler] Headers: %v", r.Header)

	// バージョン情報をレスポンスヘッダーに追加
	w.Header().Set("X-Backend-Version", "2025-10-16-anomaly-save-fix-v1")
	w.Header().Set("X-Handler-Called", "true")

	// Ginアプリケーションをセットアップ（初回のみ実行される）
	app := setupApp()

	log.Printf("🔵 [Handler] Calling Gin ServeHTTP")
	// Ginにリクエストを処理させる
	app.ServeHTTP(w, r)
	log.Printf("🔵 [Handler] Gin ServeHTTP completed")
}
