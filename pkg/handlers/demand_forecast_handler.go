package handlers

import (
	"net/http"
	"strconv"

	"hunt-chat-api/pkg/services"

	"github.com/gin-gonic/gin"
)

// DemandForecastHandler 需要予測ハンドラー
type DemandForecastHandler struct {
	demandForecastService *services.DemandForecastService
}

// NewDemandForecastHandler 新しい需要予測ハンドラーを作成
func NewDemandForecastHandler(weatherService *services.WeatherService) *DemandForecastHandler {
	return &DemandForecastHandler{
		demandForecastService: services.NewDemandForecastService(weatherService),
	}
}

// GetDemandForecastService は、ハンドラーが持つ需要予測サービスへの参照を返す
func (dfh *DemandForecastHandler) GetDemandForecastService() *services.DemandForecastService {
	return dfh.demandForecastService
}

// PredictDemand 需要予測を実行
func (dfh *DemandForecastHandler) PredictDemand(c *gin.Context) {
	var request services.DemandForecastRequest

	// リクエストボディをバインド
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "リクエストの解析に失敗しました: " + err.Error(),
		})
		return
	}

	// デフォルト値の設定
	if request.RegionCode == "" {
		request.RegionCode = "240000" // 三重県
	}
	if request.ProductCategory == "" {
		request.ProductCategory = "飲料"
	}
	if request.ForecastDays == 0 {
		request.ForecastDays = 7
	}
	if request.HistoricalDays == 0 {
		request.HistoricalDays = 30
	}

	// 需要予測を実行
	forecast, err := dfh.demandForecastService.PredictDemand(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "需要予測の実行に失敗しました: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    forecast,
	})
}

// GetDemandForecastForSuzuka 三重県鈴鹿市の需要予測を取得（簡易版）
func (dfh *DemandForecastHandler) GetDemandForecastForSuzuka(c *gin.Context) {
	// クエリパラメータの取得
	productCategory := c.Query("product_category")
	if productCategory == "" {
		productCategory = "飲料"
	}

	forecastDays := 7
	if daysStr := c.Query("forecast_days"); daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 && days <= 30 {
			forecastDays = days
		}
	}

	historicalDays := 30
	if daysStr := c.Query("historical_days"); daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 && days <= 365 {
			historicalDays = days
		}
	}

	// デフォルトの暗黙知を設定
	tacitKnowledge := []services.TacitKnowledgeItem{
		{
			Type:        "seasonal",
			Description: "夏季は冷たい飲料の需要が増加",
			Weight:      0.3,
			Condition:   "hot_day",
		},
		{
			Type:        "event",
			Description: "週末は需要が20%増加",
			Weight:      0.2,
			Condition:   "weekend",
		},
		{
			Type:        "weather",
			Description: "雨天時は屋内消費が増加",
			Weight:      0.15,
			Condition:   "rainy",
		},
	}

	// デフォルトの季節要因を設定
	seasonalFactors := services.SeasonalFactors{
		SummerDemandIncrease: 0.4,
		WinterDemandIncrease: 0.2,
		RainySeasonImpact:    0.1,
		HolidayImpact:        0.25,
	}

	// デフォルトの外部要因を設定
	externalFactors := services.ExternalFactors{
		EconomicIndex:         1.0,
		CompetitorActivity:    0.8,
		MarketingCampaign:     1.2,
		SupplyChainDisruption: 0.9,
	}

	// リクエストを構築
	request := services.DemandForecastRequest{
		RegionCode:      "240000", // 三重県
		ProductCategory: productCategory,
		ForecastDays:    forecastDays,
		HistoricalDays:  historicalDays,
		TacitKnowledge:  tacitKnowledge,
		SeasonalFactors: seasonalFactors,
		ExternalFactors: externalFactors,
	}

	// 需要予測を実行
	forecast, err := dfh.demandForecastService.PredictDemand(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "需要予測の実行に失敗しました: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    forecast,
	})
}

// GetDemandForecastSettings 需要予測設定を取得
func (dfh *DemandForecastHandler) GetDemandForecastSettings(c *gin.Context) {
	settings := gin.H{
		"available_regions": map[string]string{
			"240000": "三重県",
			"130000": "東京都",
			"270000": "大阪府",
		},
		"available_products": []string{
			"飲料",
			"アイス",
			"冷房器具",
			"暖房器具",
			"傘",
			"レインコート",
			"その他",
		},
		"forecast_range": gin.H{
			"min_days": 1,
			"max_days": 30,
		},
		"historical_range": gin.H{
			"min_days": 7,
			"max_days": 365,
		},
		"tacit_knowledge_types": []string{
			"seasonal",
			"event",
			"weather",
			"promotional",
			"competitive",
		},
		"confidence_levels": gin.H{
			"high":   0.8,
			"medium": 0.6,
			"low":    0.4,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settings,
	})
}

// GetDemandInsights 需要インサイトを取得
func (dfh *DemandForecastHandler) GetDemandInsights(c *gin.Context) {
	regionCode := c.Param("regionCode")
	if regionCode == "" {
		regionCode = "240000"
	}

	productCategory := c.Query("product_category")
	if productCategory == "" {
		productCategory = "飲料"
	}

	// 簡易的なインサイトを生成
	insights := gin.H{
		"region_code":      regionCode,
		"product_category": productCategory,
		"insights": []gin.H{
			{
				"type":        "weather_correlation",
				"title":       "気象との相関",
				"description": "気温が30℃を超えると需要が40%増加します",
				"confidence":  0.85,
				"impact":      0.4,
			},
			{
				"type":        "seasonal_pattern",
				"title":       "季節パターン",
				"description": "夏季（6-8月）は年間平均より50%高い需要が見込まれます",
				"confidence":  0.92,
				"impact":      0.5,
			},
			{
				"type":        "tacit_knowledge",
				"title":       "専門知識",
				"description": "週末は平日より20%需要が増加する傾向があります",
				"confidence":  0.78,
				"impact":      0.2,
			},
			{
				"type":        "trend_analysis",
				"title":       "トレンド分析",
				"description": "過去30日間で需要は緩やかに増加傾向にあります",
				"confidence":  0.75,
				"impact":      0.15,
			},
		},
		"recommendations": []gin.H{
			{
				"priority": "高",
				"action":   "高温予報日の在庫増強",
				"reason":   "気温上昇による需要急増に対応",
			},
			{
				"priority": "中",
				"action":   "週末前の仕入れ調整",
				"reason":   "週末需要増加に対応",
			},
			{
				"priority": "低",
				"action":   "マーケティング活動の強化",
				"reason":   "需要トレンドの維持・向上",
			},
		},
		"generated_at": "2025-07-18 02:20:22",
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    insights,
	})
}

// GetDemandAnalytics 需要分析データを取得
func (dfh *DemandForecastHandler) GetDemandAnalytics(c *gin.Context) {
	regionCode := c.Param("regionCode")
	if regionCode == "" {
		regionCode = "240000"
	}

	productCategory := c.Query("product_category")
	if productCategory == "" {
		productCategory = "飲料"
	}

	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	// 簡易的な分析データを生成
	analytics := gin.H{
		"region_code":      regionCode,
		"product_category": productCategory,
		"analysis_period":  days,
		"demand_patterns": gin.H{
			"peak_days":      []string{"土曜日", "日曜日", "祝日"},
			"low_days":       []string{"火曜日", "水曜日"},
			"seasonal_peaks": []string{"7月", "8月", "12月"},
		},
		"weather_correlation": gin.H{
			"temperature_correlation":   0.85,
			"humidity_correlation":      -0.3,
			"precipitation_correlation": -0.4,
		},
		"forecast_accuracy": gin.H{
			"last_7_days":  0.88,
			"last_30_days": 0.82,
			"last_90_days": 0.79,
		},
		"performance_metrics": gin.H{
			"mae":  45.2, // 平均絶対誤差
			"mape": 8.5,  // 平均絶対パーセント誤差
			"rmse": 67.8, // 二乗平均平方根誤差
		},
		"generated_at": "2025-07-18 02:20:22",
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    analytics,
	})
}

// DetectAnomalies 異常検知を実行
func (dfh *DemandForecastHandler) DetectAnomalies(c *gin.Context) {
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

	anomalies, err := dfh.demandForecastService.DetectAnomalies(regionCode, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "異常検知の実行に失敗しました: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    anomalies,
		"count":   len(anomalies),
	})
}
