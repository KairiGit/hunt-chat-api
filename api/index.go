package handler

// Build version: 2025-10-16-debug-v5-FORCE-REBUILD
// Vercel: Please rebuild this Go binary with pkg/ dependencies

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
		log.Printf("🟢 [setupApp] Initializing Gin application - v5")

		// .envファイルはVercelの環境変数設定から読み込まれるため、ここではgodotenvを呼び出しません。
		cfg := config.LoadConfig()

		log.Printf("🟢 [setupApp] Config loaded successfully")

		// Ginルーターの初期化
		r := gin.Default()

		// CORSミドルウェアの設定
		config := cors.DefaultConfig()
		config.AllowAllOrigins = true // VercelのプレビューURLなど、あらゆるオリジンを許可
		r.Use(cors.New(config))

		// サービスの初期化
		azureOpenAIService := services.NewAzureOpenAIService(
			cfg.AzureOpenAIEndpoint,
			cfg.AzureOpenAIAPIKey,
			cfg.AzureOpenAIAPIVersion,
			cfg.AzureOpenAIChatDeploymentName,
			cfg.AzureOpenAIEmbeddingDeploymentName,
		)
		vectorStoreService := services.NewVectorStoreService(azureOpenAIService, cfg.QdrantURL, cfg.QdrantAPIKey)

		// ハンドラーの初期化
		weatherHandler := handlers.NewWeatherHandler()
		demandForecastHandler := handlers.NewDemandForecastHandler(weatherHandler.GetWeatherService())
		aiHandler := handlers.NewAIHandler(azureOpenAIService, weatherHandler.GetWeatherService(), demandForecastHandler.GetDemandForecastService(), vectorStoreService)

		// 認証ミドルウェア
		authMiddleware := func(apiKey string) gin.HandlerFunc {
			return func(c *gin.Context) {
				// Vercel環境では一時的に認証をスキップ（デバッグ用）
				// TODO: 本番環境では必ず認証を有効化すること
				if apiKey == "" || apiKey == "default_secret_key" {
					c.Next()
					return
				}

				providedKey := c.GetHeader("X-API-KEY")
				// API Keyが提供されていない場合も一時的に許可（デバッグ用）
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
		r.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy", "version": "2024-10-15-v2"})
		})

		// APIルートの定義
		v1 := r.Group("/api/v1")
		v1.Use(authMiddleware(cfg.APIKey)) // ミドルウェアをグループに適用
		{
			v1.GET("/hello", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Hello from Vercel!"})
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
				ai.POST("/analyze-file", func(c *gin.Context) {
					log.Printf("🟢 [api/index.go] /analyze-file エンドポイント呼び出し - Build: 2025-10-16-debug-v5")

					// 🔍 診断: リクエストがここまで到達していることを確認
					c.Header("X-Backend-Version", "2025-10-16-debug-v5")
					c.Header("X-Handler-Called", "true")

					aiHandler.AnalyzeFile(c)
				})
			}
		}

		app = r
	})
	return app
}

// Handler はVercelからのすべてのリクエストを処理するエントリーポイントです。
func Handler(w http.ResponseWriter, r *http.Request) {
	// デバッグ: リクエストの詳細をログ出力
	log.Printf("🔵 [Handler] Request received: %s %s", r.Method, r.URL.Path)
	log.Printf("🔵 [Handler] Headers: %v", r.Header)

	// バージョン情報をレスポンスヘッダーに追加
	w.Header().Set("X-Backend-Version", "2025-10-16-debug-v5")
	w.Header().Set("X-Handler-Called", "true")

	// Ginアプリケーションをセットアップ（初回のみ実行される）
	app := setupApp()

	log.Printf("🔵 [Handler] Calling Gin ServeHTTP")
	// Ginにリクエストを処理させる
	app.ServeHTTP(w, r)
	log.Printf("🔵 [Handler] Gin ServeHTTP completed")
}
