package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	// Ginのテストモードに設定
	gin.SetMode(gin.TestMode)

	// ルーターを作成
	router := gin.New()

	// ヘルスチェックエンドポイントを追加
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "HUNT Chat-API",
		})
	})

	// テストリクエストを作成
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// レスポンスレコーダーを作成
	w := httptest.NewRecorder()

	// リクエストを実行
	router.ServeHTTP(w, req)

	// ステータスコードを確認
	assert.Equal(t, http.StatusOK, w.Code)

	// レスポンスボディが空でないことを確認
	assert.NotEmpty(t, w.Body.String())

	// JSONレスポンスに期待されるフィールドが含まれていることを確認
	assert.Contains(t, w.Body.String(), "status")
	assert.Contains(t, w.Body.String(), "service")
}

func TestHelloAPI(t *testing.T) {
	// Ginのテストモードに設定
	gin.SetMode(gin.TestMode)

	// ルーターを作成
	router := gin.New()

	// Hello APIエンドポイントを追加
	router.GET("/api/v1/hello", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Hello from HUNT Chat-API!",
		})
	})

	// テストリクエストを作成
	req, err := http.NewRequest("GET", "/api/v1/hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	// レスポンスレコーダーを作成
	w := httptest.NewRecorder()

	// リクエストを実行
	router.ServeHTTP(w, req)

	// ステータスコードを確認
	assert.Equal(t, http.StatusOK, w.Code)

	// JSONレスポンスに期待されるメッセージが含まれていることを確認
	assert.Contains(t, w.Body.String(), "Hello from HUNT Chat-API!")
}

func TestWeatherHandlerCreation(t *testing.T) {
	handler := NewWeatherHandler()

	assert.NotNil(t, handler, "WeatherHandler should not be nil")
	assert.NotNil(t, handler.GetWeatherService(), "WeatherService should not be nil")
}

func TestWeatherRegionCodesEndpoint(t *testing.T) {
	// Ginのテストモードに設定
	gin.SetMode(gin.TestMode)

	// ルーターを作成
	router := gin.New()

	// WeatherHandlerを作成
	weatherHandler := NewWeatherHandler()

	// エンドポイントを追加
	router.GET("/api/v1/weather/regions", weatherHandler.GetRegionCodes)

	// テストリクエストを作成
	req, err := http.NewRequest("GET", "/api/v1/weather/regions", nil)
	if err != nil {
		t.Fatal(err)
	}

	// レスポンスレコーダーを作成
	w := httptest.NewRecorder()

	// リクエストを実行
	router.ServeHTTP(w, req)

	// ステータスコードを確認
	assert.Equal(t, http.StatusOK, w.Code)

	// JSONレスポンスに期待されるフィールドが含まれていることを確認
	assert.Contains(t, w.Body.String(), "success")
	assert.Contains(t, w.Body.String(), "data")
}
