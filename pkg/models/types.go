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
	PredictedValue      float64            `json:"predicted_value"`
	ConfidenceInterval  ConfidenceInterval `json:"confidence_interval"`
	Confidence          float64            `json:"confidence"`           // 0-1, based on R²
	PredictionFactors   []string           `json:"prediction_factors"`   // List of factors considered
	RegressionEquation  string             `json:"regression_equation"`  // e.g., "y = 3.71x + 123.47"
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
	Deviation     float64 `json:"deviation"`      // Absolute deviation from expected
	ZScore        float64 `json:"z_score"`        // Standard deviations from mean
	AnomalyType   string  `json:"anomaly_type"`   // "急増" or "急減"
	Severity      string  `json:"severity"`       // "low", "medium", "high", "critical"
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

