package handlers

import (
	"net/http"
	"strconv"
	"time"

	"hunt-chat-api/internal/services"

	"github.com/gin-gonic/gin"
)

// WeatherHandler 気象データハンドラー
type WeatherHandler struct {
	weatherService *services.WeatherService
}

// NewWeatherHandler 新しい気象データハンドラーを作成
func NewWeatherHandler() *WeatherHandler {
	return &WeatherHandler{
		weatherService: services.NewWeatherService(),
	}
}

// GetWeatherService WeatherServiceを取得
func (wh *WeatherHandler) GetWeatherService() *services.WeatherService {
	return wh.weatherService
}

// GetForecastData 予報データを取得するハンドラー
func (wh *WeatherHandler) GetForecastData(c *gin.Context) {
	regionCode := c.Param("regionCode")
	if regionCode == "" {
		regionCode = "130000" // デフォルト：東京都
	}

	forecastData, err := wh.weatherService.GetForecastData(regionCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    forecastData,
		"count":   len(forecastData),
	})
}

// GetTokyoWeatherData 東京の気象データを取得するハンドラー
func (wh *WeatherHandler) GetTokyoWeatherData(c *gin.Context) {
	weatherData, err := wh.weatherService.GetTokyoWeatherData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    weatherData,
		"count":   len(weatherData),
	})
}

// GetRegionCodes 地域コード一覧を取得するハンドラー
func (wh *WeatherHandler) GetRegionCodes(c *gin.Context) {
	regionCodes := wh.weatherService.GetRegionCodes()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    regionCodes,
	})
}

// TestWeatherAPI 気象庁APIのテストハンドラー
func (wh *WeatherHandler) TestWeatherAPI(c *gin.Context) {
	// バックグラウンドでテストを実行
	go wh.weatherService.TestWeatherAPI()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "気象庁APIテストを開始しました。ログを確認してください。",
	})
}

// GetWeatherByRegion 指定地域の気象データを取得
func (wh *WeatherHandler) GetWeatherByRegion(c *gin.Context) {
	regionCode := c.Param("regionCode")
	if regionCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "地域コードが指定されていません",
		})
		return
	}

	// 日数制限（オプション）
	limit := 7 // デフォルト7日分
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	forecastData, err := wh.weatherService.GetForecastData(regionCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 簡易的な制限処理
	if len(forecastData) > limit {
		forecastData = forecastData[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"region_code": regionCode,
		"data":        forecastData,
		"count":       len(forecastData),
		"limit":       limit,
	})
}

// GetHistoricalWeatherData 過去の気象データを取得するハンドラー
func (wh *WeatherHandler) GetHistoricalWeatherData(c *gin.Context) {
	regionCode := c.Param("regionCode")
	if regionCode == "" {
		regionCode = "130000" // デフォルト：東京都
	}

	// 日数パラメータの取得
	days := 7 // デフォルト7日分
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	// 期間指定での取得
	historicalData, err := wh.weatherService.GetHistoricalWeatherDataByRange(regionCode, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"region_code": regionCode,
		"data":        historicalData,
		"count":       len(historicalData),
		"days":        days,
	})
}

// GetHistoricalWeatherDataByDate 指定日の過去気象データを取得
func (wh *WeatherHandler) GetHistoricalWeatherDataByDate(c *gin.Context) {
	regionCode := c.Param("regionCode")
	if regionCode == "" {
		regionCode = "130000" // デフォルト：東京都
	}

	dateStr := c.Query("date")
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "日付パラメータ (date) が必要です。形式: YYYY-MM-DD",
		})
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "日付形式が正しくありません。形式: YYYY-MM-DD",
		})
		return
	}

	// 単日データの取得
	historicalData, err := wh.weatherService.GetHistoricalWeatherData(regionCode, date, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"region_code": regionCode,
		"date":        dateStr,
		"data":        historicalData,
		"count":       len(historicalData),
	})
}

// GetHistoricalWeatherDataRange 期間指定での過去気象データを取得
func (wh *WeatherHandler) GetHistoricalWeatherDataRange(c *gin.Context) {
	regionCode := c.Param("regionCode")
	if regionCode == "" {
		regionCode = "130000" // デフォルト：東京都
	}

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "開始日 (start_date) と終了日 (end_date) が必要です。形式: YYYY-MM-DD",
		})
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "開始日の形式が正しくありません。形式: YYYY-MM-DD",
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "終了日の形式が正しくありません。形式: YYYY-MM-DD",
		})
		return
	}

	// 期間データの取得
	historicalData, err := wh.weatherService.GetHistoricalWeatherData(regionCode, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"region_code": regionCode,
		"start_date":  startDateStr,
		"end_date":    endDateStr,
		"data":        historicalData,
		"count":       len(historicalData),
	})
}

// GetAvailableHistoricalDataRange 利用可能な過去データの期間を取得
func (wh *WeatherHandler) GetAvailableHistoricalDataRange(c *gin.Context) {
	dataRange := wh.weatherService.GetAvailableHistoricalDataRange()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dataRange,
	})
}

// GetSuzukaMonthlyWeatherSummary 三重県鈴鹿市の過去一か月分の気象データをまとめて取得
func (wh *WeatherHandler) GetSuzukaMonthlyWeatherSummary(c *gin.Context) {
	// 三重県のコード: 240000
	regionCode := "240000"
	
	// 過去30日分のデータを取得
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 90 {
			days = d
		}
	}

	// 集計タイプの指定
	summaryType := c.Query("type")
	if summaryType == "" {
		summaryType = "daily" // デフォルト：日別
	}

	// データ取得
	weatherSummary, err := wh.weatherService.GetSuzukaWeatherSummary(days, summaryType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"region_code":  regionCode,
		"region_name":  "三重県（鈴鹿市含む）",
		"days":         days,
		"summary_type": summaryType,
		"data":         weatherSummary,
	})
}

// GetWeatherDataAnalysis 気象データの分析結果を取得
func (wh *WeatherHandler) GetWeatherDataAnalysis(c *gin.Context) {
	regionCode := c.Param("regionCode")
	if regionCode == "" {
		regionCode = "240000" // デフォルト：三重県
	}

	// 分析期間の指定
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	// 分析タイプの指定
	analysisType := c.Query("analysis_type")
	if analysisType == "" {
		analysisType = "comprehensive" // デフォルト：包括的分析
	}

	// 分析データを取得
	analysis, err := wh.weatherService.GetWeatherDataAnalysis(regionCode, days, analysisType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"region_code":   regionCode,
		"days":          days,
		"analysis_type": analysisType,
		"data":          analysis,
	})
}

// GetWeatherTrendAnalysis 気象データのトレンド分析を取得
func (wh *WeatherHandler) GetWeatherTrendAnalysis(c *gin.Context) {
	regionCode := c.Param("regionCode")
	if regionCode == "" {
		regionCode = "240000" // デフォルト：三重県
	}

	// 分析期間の指定
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	// トレンド分析を取得
	trendAnalysis, err := wh.weatherService.GetWeatherTrendAnalysis(regionCode, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"region_code": regionCode,
		"days":        days,
		"data":        trendAnalysis,
	})
}

// GetWeatherDataByCategory カテゴリ別の気象データを取得
func (wh *WeatherHandler) GetWeatherDataByCategory(c *gin.Context) {
	regionCode := c.Param("regionCode")
	if regionCode == "" {
		regionCode = "240000" // デフォルト：三重県
	}

	// カテゴリの指定
	category := c.Query("category")
	if category == "" {
		category = "all" // デフォルト：全カテゴリ
	}

	// 期間の指定
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	// カテゴリ別データを取得
	categoryData, err := wh.weatherService.GetWeatherDataByCategory(regionCode, category, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"region_code": regionCode,
		"category":    category,
		"days":        days,
		"data":        categoryData,
	})
}
