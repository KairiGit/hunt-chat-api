package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"hunt-chat-api/configs"
	"hunt-chat-api/internal/handlers"
	"hunt-chat-api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// テスト環境の設定
	gin.SetMode(gin.TestMode)
	
	// .envファイルを読み込み（テスト環境では無視される可能性がある）
	godotenv.Load("../../.env")
	
	// テスト実行
	code := m.Run()
	
	// 終了
	os.Exit(code)
}

func TestApplicationSetup(t *testing.T) {
	// 設定の読み込みテスト
	cfg := config.LoadConfig()
	assert.NotNil(t, cfg, "Config should not be nil")
	
	// サービスの初期化テスト
	azureOpenAIService := services.NewAzureOpenAIService(
		cfg.AzureOpenAIEndpoint,
		cfg.AzureOpenAIAPIKey,
		cfg.AzureOpenAIAPIVersion,
		cfg.AzureOpenAIDeploymentName,
	)
	assert.NotNil(t, azureOpenAIService, "AzureOpenAIService should not be nil")
	
	// ハンドラーの初期化テスト
	weatherHandler := handlers.NewWeatherHandler()
	assert.NotNil(t, weatherHandler, "WeatherHandler should not be nil")
	
	demandForecastHandler := handlers.NewDemandForecastHandler(weatherHandler.GetWeatherService())
	assert.NotNil(t, demandForecastHandler, "DemandForecastHandler should not be nil")
	
	aiHandler := handlers.NewAIHandler(azureOpenAIService, weatherHandler.GetWeatherService())
	assert.NotNil(t, aiHandler, "AIHandler should not be nil")
}

func TestRouterSetup(t *testing.T) {
	// ルーターの初期化
	r := gin.New()
	
	// ヘルスチェックエンドポイント
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "HUNT Chat-API",
		})
	})
	
	// APIバージョン1のルートグループ
	v1 := r.Group("/api/v1")
	{
		v1.GET("/hello", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Hello from HUNT Chat-API!",
			})
		})
	}
	
	// ヘルスチェックのテスト
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Hello APIのテスト
	req, _ = http.NewRequest("GET", "/api/v1/hello", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEnvironmentVariables(t *testing.T) {
	// テスト用の環境変数を設定
	testEnvVars := map[string]string{
		"AZURE_OPENAI_ENDPOINT": "https://test.openai.azure.com/",
		"AZURE_OPENAI_API_KEY":  "test-key",
		"AZURE_OPENAI_MODEL":    "gpt-4",
	}
	
	// 環境変数を設定
	for key, value := range testEnvVars {
		os.Setenv(key, value)
	}
	
	// テスト後にクリーンアップ
	defer func() {
		for key := range testEnvVars {
			os.Unsetenv(key)
		}
	}()
	
	for envVar := range testEnvVars {
		value := os.Getenv(envVar)
		assert.NotEmpty(t, value, "Environment variable %s should not be empty", envVar)
	}
}
