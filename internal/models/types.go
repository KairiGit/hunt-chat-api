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
	ProductID      string                 `json:"product_id" binding:"required"`
	TimeRange      string                 `json:"time_range" binding:"required"`
	HistoricalData []HistoricalDataPoint  `json:"historical_data"`
	ExternalFactors map[string]interface{} `json:"external_factors,omitempty"`
}

// HistoricalDataPoint represents a single data point in historical data
type HistoricalDataPoint struct {
	Date   string  `json:"date"`
	Sales  float64 `json:"sales"`
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