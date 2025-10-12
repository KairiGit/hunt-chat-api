package models

// ChatRequest represents an incoming chat request
type ChatRequest struct {
	Message string `json:"message" binding:"required"`
	Context string `json:"context,omitempty"`
}

// ChatResponse represents the response from the chat API
type ChatResponse struct {
	Response  string `json:"response"`
	Timestamp string `json:"timestamp"`
	Model     string `json:"model"`
}

// DemandForecastRequest represents a demand forecast request
type DemandForecastRequest struct {
	ProductID       string                 `json:"product_id" binding:"required"`
	TimeRange       string                 `json:"time_range" binding:"required"`
	HistoricalData  []HistoricalDataPoint  `json:"historical_data"`
	ExternalFactors map[string]interface{} `json:"external_factors,omitempty"`
}

// HistoricalDataPoint represents a single data point in historical data
type HistoricalDataPoint struct {
	Date   string   `json:"date"`
	Sales  float64  `json:"sales"`
	Events []string `json:"events,omitempty"`
}

// DemandForecastResponse represents the response from demand forecast
type DemandForecastResponse struct {
	ProductID       string                 `json:"product_id"`
	ForecastedSales float64                `json:"forecasted_sales"`
	Confidence      float64                `json:"confidence"`
	Factors         map[string]interface{} `json:"factors"`
	Timestamp       string                 `json:"timestamp"`
}

// SalesRecord represents a single sales record.
// This will be used to import historical sales data.
type SalesRecord struct {
	Date          string  `json:"date"`
	ProductID     string  `json:"product_id"`
	ProductName   string  `json:"product_name"`
	SalesAmount   float64 `json:"sales_amount"`
	SalesQuantity int     `json:"sales_quantity"`
	Region        string  `json:"region"`
}

// Anomaly represents a detected anomaly in the sales data.
type Anomaly struct {
	Date        string  `json:"date"`
	ProductID   string  `json:"product_id"`
	Description string  `json:"description"`
	ImpactScore float64 `json:"impact_score"`
	Trigger     string  `json:"trigger"` // e.g., "weather", "event"
	Weather     string  `json:"weather,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// CorrelationResult represents the result of correlation analysis
type CorrelationResult struct {
	Factor          string  `json:"factor"`           // e.g., "temperature", "humidity"
	CorrelationCoef float64 `json:"correlation_coef"` // Pearson correlation coefficient (-1 to 1)
	PValue          float64 `json:"p_value"`          // Statistical significance
	SampleSize      int     `json:"sample_size"`      // Number of data points used
	Interpretation  string  `json:"interpretation"`   // Human-readable interpretation
}

// RegressionResult represents the result of regression analysis
type RegressionResult struct {
	Slope       float64 `json:"slope"`       // Regression slope
	Intercept   float64 `json:"intercept"`   // Regression intercept
	RSquared    float64 `json:"r_squared"`   // R² (coefficient of determination)
	Prediction  float64 `json:"prediction"`  // Predicted value
	Confidence  float64 `json:"confidence"`  // Confidence level
	Description string  `json:"description"` // Description of the result
}

// AnalysisReport represents a comprehensive analysis report
type AnalysisReport struct {
	ReportID        string              `json:"report_id"`
	FileName        string              `json:"file_name"`
	AnalysisDate    string              `json:"analysis_date"`
	DataPoints      int                 `json:"data_points"`
	DateRange       string              `json:"date_range"`
	WeatherMatches  int                 `json:"weather_matches"`
	Summary         string              `json:"summary"`
	Correlations    []CorrelationResult `json:"correlations"`
	Regression      *RegressionResult   `json:"regression,omitempty"`
	AIInsights      string              `json:"ai_insights"`
	Recommendations []string            `json:"recommendations"`
}

// WeatherSalesData represents a single data point combining weather and sales
type WeatherSalesData struct {
	Date        string  `json:"date"`
	ProductID   string  `json:"product_id"`
	Sales       float64 `json:"sales"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	Weather     string  `json:"weather"`
}

// SalesPrediction represents a future sales prediction with confidence interval
type SalesPrediction struct {
	PredictedValue     float64            `json:"predicted_value"`
	ConfidenceInterval ConfidenceInterval `json:"confidence_interval"`
	Confidence         float64            `json:"confidence"`          // 0-1, based on R²
	PredictionFactors  []string           `json:"prediction_factors"`  // List of factors considered
	RegressionEquation string             `json:"regression_equation"` // e.g., "y = 3.71x + 123.47"
}

// ConfidenceInterval represents the confidence interval for a prediction
type ConfidenceInterval struct {
	Lower      float64 `json:"lower"`
	Upper      float64 `json:"upper"`
	Confidence float64 `json:"confidence"` // e.g., 0.95 for 95%
}

// AnomalyDetection represents a detected anomaly in the data
type AnomalyDetection struct {
	Date          string  `json:"date"`
	ActualValue   float64 `json:"actual_value"`
	ExpectedValue float64 `json:"expected_value"`
	Deviation     float64 `json:"deviation"`             // Absolute deviation from expected
	ZScore        float64 `json:"z_score"`               // Standard deviations from mean
	AnomalyType   string  `json:"anomaly_type"`          // "急増" or "急減"
	Severity      string  `json:"severity"`              // "low", "medium", "high", "critical"
	AIQuestion    string  `json:"ai_question,omitempty"` // AI-generated question
}

// PredictionRequest represents a request for sales prediction
type PredictionRequest struct {
	ProductID         string  `json:"product_id" binding:"required"`
	FutureTemperature float64 `json:"future_temperature" binding:"required"`
	ConfidenceLevel   float64 `json:"confidence_level"` // 0.90, 0.95, 0.99
	DateRange         string  `json:"date_range"`       // e.g., "2022-01-01:2024-12-31"
}

// PredictionResponse represents the response for sales prediction
type PredictionResponse struct {
	Success    bool            `json:"success"`
	Prediction SalesPrediction `json:"prediction"`
	Message    string          `json:"message,omitempty"`
}

// AnomalyDetectionResponse represents the response for anomaly detection
type AnomalyDetectionResponse struct {
	Success   bool               `json:"success"`
	Anomalies []AnomalyDetection `json:"anomalies"`
	Message   string             `json:"message,omitempty"`
}

// ProductForecastRequest represents a request for product-specific forecast
type ProductForecastRequest struct {
	ProductID   string `json:"product_id" binding:"required"`
	ProductName string `json:"product_name,omitempty"`
	Period      string `json:"period" binding:"required"` // "week", "2weeks", "month"
	RegionCode  string `json:"region_code"`
	StartDate   string `json:"start_date"` // Historical data start date
	EndDate     string `json:"end_date"`   // Historical data end date
}

// ProductForecast represents a forecast for a specific product
type ProductForecast struct {
	ProductID          string             `json:"product_id"`
	ProductName        string             `json:"product_name"`
	ForecastPeriod     string             `json:"forecast_period"` // "2025-01-15 〜 2025-01-21"
	PredictedTotal     float64            `json:"predicted_total"` // Total demand for the period
	DailyAverage       float64            `json:"daily_average"`   // Average per day
	ConfidenceInterval ConfidenceInterval `json:"confidence_interval"`
	Confidence         float64            `json:"confidence"`            // Model confidence (R²)
	DailyBreakdown     []DailyForecast    `json:"daily_breakdown"`       // Day-by-day forecast
	Factors            []string           `json:"factors"`               // Factors considered
	Seasonality        string             `json:"seasonality,omitempty"` // e.g., "夏季需要増加傾向"
	Recommendations    []string           `json:"recommendations"`
}

// DailyForecast represents a single day's forecast
type DailyForecast struct {
	Date           string  `json:"date"`
	DayOfWeek      string  `json:"day_of_week"` // "月", "火", etc.
	PredictedValue float64 `json:"predicted_value"`
	Temperature    float64 `json:"temperature,omitempty"`
	Weather        string  `json:"weather,omitempty"`
}

// ProductForecastResponse represents the response for product forecast
type ProductForecastResponse struct {
	Success  bool            `json:"success"`
	Forecast ProductForecast `json:"forecast"`
	Message  string          `json:"message,omitempty"`
}

// ProductSalesHistory represents historical sales data for a product
type ProductSalesHistory struct {
	ProductID   string            `json:"product_id"`
	ProductName string            `json:"product_name"`
	DataPoints  []SalesDataPoint  `json:"data_points"`
	Statistics  ProductStatistics `json:"statistics"`
}

// SalesDataPoint represents a single sales record
type SalesDataPoint struct {
	Date        string  `json:"date"`
	Sales       float64 `json:"sales"`
	Temperature float64 `json:"temperature,omitempty"`
	DayOfWeek   string  `json:"day_of_week,omitempty"`
}

// ProductStatistics represents statistical summary for a product
type ProductStatistics struct {
	Mean           float64            `json:"mean"`
	Median         float64            `json:"median"`
	StdDev         float64            `json:"std_dev"`
	Min            float64            `json:"min"`
	Max            float64            `json:"max"`
	WeekdayAverage map[string]float64 `json:"weekday_average"` // Average by day of week
	MonthlyAverage map[string]float64 `json:"monthly_average"` // Average by month
	TrendDirection string             `json:"trend_direction"` // "増加", "減少", "安定"
}

// WeeklyAnalysisRequest represents a request for weekly sales analysis
type WeeklyAnalysisRequest struct {
	ProductID string           `json:"product_id" binding:"required"`
	StartDate string           `json:"start_date" binding:"required"` // YYYY-MM-DD
	EndDate   string           `json:"end_date" binding:"required"`   // YYYY-MM-DD
	SalesData []SalesDataPoint `json:"sales_data"`
}

// WeeklyAnalysisResponse represents weekly aggregated analysis
type WeeklyAnalysisResponse struct {
	ProductID       string             `json:"product_id"`
	ProductName     string             `json:"product_name"`
	AnalysisPeriod  string             `json:"analysis_period"`
	TotalWeeks      int                `json:"total_weeks"`
	WeeklySummary   []WeeklySummary    `json:"weekly_summary"`
	OverallStats    WeeklyOverallStats `json:"overall_stats"`
	Trends          WeeklyTrends       `json:"trends"`
	Recommendations []string           `json:"recommendations"`
}

// WeeklySummary represents summary for a single week
type WeeklySummary struct {
	WeekNumber     int     `json:"week_number"` // Week number in the period
	WeekStart      string  `json:"week_start"`  // YYYY-MM-DD
	WeekEnd        string  `json:"week_end"`    // YYYY-MM-DD
	TotalSales     float64 `json:"total_sales"`
	AverageSales   float64 `json:"average_sales"`
	MinSales       float64 `json:"min_sales"`
	MaxSales       float64 `json:"max_sales"`
	BusinessDays   int     `json:"business_days"`
	WeekOverWeek   float64 `json:"week_over_week"` // % change from previous week
	StdDev         float64 `json:"std_dev"`
	AvgTemperature float64 `json:"avg_temperature"`
}

// WeeklyOverallStats represents overall statistics across all weeks
type WeeklyOverallStats struct {
	AverageWeeklySales float64 `json:"average_weekly_sales"`
	MedianWeeklySales  float64 `json:"median_weekly_sales"`
	StdDevWeeklySales  float64 `json:"std_dev_weekly_sales"`
	BestWeek           int     `json:"best_week"`
	WorstWeek          int     `json:"worst_week"`
	GrowthRate         float64 `json:"growth_rate"` // Overall % growth
	Volatility         float64 `json:"volatility"`  // Coefficient of variation
}

// WeeklyTrends represents trend analysis
type WeeklyTrends struct {
	Direction     string  `json:"direction"`   // "上昇", "下降", "横ばい"
	Strength      float64 `json:"strength"`    // 0-1
	Seasonality   string  `json:"seasonality"` // Detected seasonal pattern
	PeakWeek      int     `json:"peak_week"`
	LowWeek       int     `json:"low_week"`
	AverageGrowth float64 `json:"average_growth"` // Average week-over-week %
}
