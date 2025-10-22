package handlers

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"hunt-chat-api/pkg/services"

	"github.com/gin-gonic/gin"
)

// EconomicHandler exposes economic/market series endpoints backed by CSV pseudo-yfinance.
type EconomicHandler struct {
	svc *services.EconomicService
	vs  *services.VectorStoreService
}

func NewEconomicHandler(vs *services.VectorStoreService) *EconomicHandler {
	// Default mapping: NIKKEI -> data/econ/nikkei_daily.csv
	mapping := map[string]string{
		"NIKKEI": "data/econ/nikkei_daily.csv",
	}
	svc := services.NewEconomicService("", mapping)
	return &EconomicHandler{svc: svc, vs: vs}
}

func (h *EconomicHandler) GetSeries(c *gin.Context) {
	symbol := c.DefaultQuery("symbol", "NIKKEI")
	startStr := c.DefaultQuery("start", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	endStr := c.DefaultQuery("end", time.Now().Format("2006-01-02"))

	start, err1 := time.Parse("2006-01-02", startStr)
	end, err2 := time.Parse("2006-01-02", endStr)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start/end format (use YYYY-MM-DD)"})
		return
	}

	series, err := h.svc.GetMarketSeries(symbol, start, end)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// JSON-friendly shape
	out := make([]gin.H, 0, len(series))
	for _, p := range series {
		out = append(out, gin.H{"date": p.Date.Format("2006-01-02"), "value": p.Value})
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol": symbol,
		"start":  start.Format("2006-01-02"),
		"end":    end.Format("2006-01-02"),
		"series": out,
		"count":  len(out),
	})
}

func (h *EconomicHandler) GetReturns(c *gin.Context) {
	symbol := c.DefaultQuery("symbol", "NIKKEI")
	startStr := c.DefaultQuery("start", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	endStr := c.DefaultQuery("end", time.Now().Format("2006-01-02"))

	start, err1 := time.Parse("2006-01-02", startStr)
	end, err2 := time.Parse("2006-01-02", endStr)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start/end format (use YYYY-MM-DD)"})
		return
	}

	rets, err := h.svc.GetPctChange(symbol, start, end)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(rets))
	for _, p := range rets {
		out = append(out, gin.H{"date": p.Date.Format("2006-01-02"), "pct_change": p.Value})
	}
	c.JSON(http.StatusOK, gin.H{"symbol": symbol, "returns": out, "count": len(out)})
}

func (h *EconomicHandler) RegisterSymbol(c *gin.Context) {
	var req struct {
		Symbol  string `json:"symbol"`
		CSVPath string `json:"csv_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Symbol == "" || req.CSVPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol and csv_path are required"})
		return
	}
	h.svc.RegisterSymbol(req.Symbol, req.CSVPath)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ImportCSV imports an uploaded CSV (multipart file or JSON csv_text) and stores new daily points in Qdrant.
// Params: symbol (query/form/json)
func (h *EconomicHandler) ImportCSV(c *gin.Context) {
	// Get symbol from query or form or JSON
	symbol := c.DefaultQuery("symbol", "")
	if symbol == "" {
		symbol = c.PostForm("symbol")
	}
	type jsonReq struct {
		Symbol  string `json:"symbol"`
		CSVText string `json:"csv_text"`
	}
	var jr jsonReq
	// If JSON body, bind regardless of symbol presence to allow csv_text usage
	if strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		if err := c.ShouldBindJSON(&jr); err == nil {
			if symbol == "" {
				symbol = jr.Symbol
			}
		}
	} else {
		if symbol == "" { // as a fallback, try JSON once when symbol missing
			_ = c.ShouldBindJSON(&jr)
			symbol = jr.Symbol
		}
	}
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	// Read CSV bytes from multipart file or json csv_text
	var data []byte
	var err error
	if strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		if jr.CSVText == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "csv_text is required in JSON body"})
			return
		}
		data = []byte(jr.CSVText)
	} else if file, err := c.FormFile("file"); err == nil {
		f, err := file.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		defer f.Close()
		var b []byte
		b, err = io.ReadAll(f)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		data = b
	} else {
		// try raw body
		var b []byte
		b, err = io.ReadAll(c.Request.Body)
		if err != nil || len(b) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "csv not provided (multipart file 'file' or json 'csv_text')"})
			return
		}
		data = b
	}

	// Parse CSV into series points
	points, err := h.svc.ParseCSVBytes(data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "hint": "CSVはUTF-8。日付ヘッダ=Date/年月日/日付/データ日付、値ヘッダ=Close/Adj Close/Price/Value/終値 のいずれかを含めてください"})
		return
	}
	if len(points) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no valid rows", "hint": "日付と終値の行が有効でない可能性があります"})
		return
	}

	// Get latest stored date for symbol (bound by timeout)
	var latest string
	if h.vs != nil {
		ctxLatest, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()
		latest, err = h.vs.GetLatestEconomicDate(ctxLatest, symbol)
		if err != nil {
			latest = ""
		}
	}

	// Filter new points only (date > latest)
	newBatch := make([]struct {
		Date  string
		Value float64
	}, 0, len(points))
	for _, p := range points {
		d := p.Date.Format("2006-01-02")
		if latest == "" || d > latest {
			newBatch = append(newBatch, struct {
				Date  string
				Value float64
			}{Date: d, Value: p.Value})
		}
	}
	if len(newBatch) == 0 {
		c.JSON(http.StatusOK, gin.H{"symbol": symbol, "stored": 0, "message": "no new data"})
		return
	}

	// Store to Qdrant (bound by timeout)
	if h.vs != nil {
		ctxUpsert, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
		defer cancel()
		if err := h.vs.StoreEconomicDailyBatch(ctxUpsert, symbol, newBatch); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"symbol": symbol, "stored": len(newBatch)})
}

// AnalyzeLaggedCorrelation computes lagged correlations between a given product sales series (provided) and an economic symbol stored in Qdrant.
// Query/body params:
// - symbol (economic, e.g., NIKKEI)
// - start, end (YYYY-MM-DD)
// - sales: [{date, sales}] as JSON array in body
func (h *EconomicHandler) AnalyzeLaggedCorrelation(c *gin.Context) {
	type salesPoint struct { Date string `json:"date"`; Sales float64 `json:"sales"` }
	var input struct {
		Symbol string       `json:"symbol"`
		Start  string       `json:"start"`
		End    string       `json:"end"`
		Sales  []salesPoint `json:"sales"`
		MaxLag int          `json:"max_lag"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	if input.Symbol == "" || len(input.Sales) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol and sales are required"})
		return
	}
	if input.MaxLag <= 0 { input.MaxLag = 14 }
	start, err1 := time.Parse("2006-01-02", input.Start)
	end, err2 := time.Parse("2006-01-02", input.End)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start/end format (use YYYY-MM-DD)"})
		return
	}
	if h.vs == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vector store unavailable"})
		return
	}
	// Fetch economic series from Qdrant
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	econ, err := h.vs.GetEconomicSeries(ctx, input.Symbol, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(econ) < 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient economic series"})
		return
	}
	// Prepare aligned arrays
	xDates := make([]string, 0, len(econ))
	xVals := make([]float64, 0, len(econ))
	for _, p := range econ {
		xDates = append(xDates, p.Date)
		xVals = append(xVals, p.Value)
	}
	yDates := make([]string, 0, len(input.Sales))
	yVals := make([]float64, 0, len(input.Sales))
	for _, spt := range input.Sales {
		yDates = append(yDates, spt.Date)
		yVals = append(yVals, spt.Sales)
	}
	// Use statistics service (standalone instance is sufficient)
	stats := services.NewStatisticsService(nil, nil)
	results, err := stats.CalculateLaggedCorrelations(xDates, xVals, yDates, yVals, input.MaxLag)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"symbol":   input.Symbol,
		"start":    input.Start,
		"end":      input.End,
		"max_lag":  input.MaxLag,
		"results":  results,
		"top":      results[0],
	})
}

// ImportSales imports daily sales data for a product into Qdrant.
// JSON body: { product_id: string, data: [{date: YYYY-MM-DD, sales: number}] }
func (h *EconomicHandler) ImportSales(c *gin.Context) {
	if h.vs == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vector store unavailable"})
		return
	}
	var req struct {
		ProductID string `json:"product_id"`
		Data      []struct {
			Date  string  `json:"date"`
			Sales float64 `json:"sales"`
		} `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ProductID == "" || len(req.Data) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id and data are required"})
		return
	}
	// Get latest stored date to do incremental upsert
	ctxLatest, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	latest, _ := h.vs.GetLatestSalesDate(ctxLatest, req.ProductID)
	// Filter new points
	batch := make([]struct{ Date string; Sales float64 }, 0, len(req.Data))
	for _, d := range req.Data {
		if d.Date == "" { continue }
		if latest == "" || d.Date > latest {
			batch = append(batch, struct{ Date string; Sales float64 }{Date: d.Date, Sales: d.Sales})
		}
	}
	if len(batch) == 0 {
		c.JSON(http.StatusOK, gin.H{"product_id": req.ProductID, "stored": 0, "message": "no new data"})
		return
	}
	ctxUpsert, cancel2 := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel2()
	if err := h.vs.StoreSalesDailyBatch(ctxUpsert, req.ProductID, batch); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_id": req.ProductID, "stored": len(batch)})
}

// AnalyzeProductEconLagged runs lagged correlation using server-side stored series.
// JSON body: { product_id, symbol, start, end, max_lag? }
func (h *EconomicHandler) AnalyzeProductEconLagged(c *gin.Context) {
	if h.vs == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vector store unavailable"})
		return
	}
	var req struct {
		ProductID string `json:"product_id"`
		Symbol    string `json:"symbol"`
		Start     string `json:"start"`
		End       string `json:"end"`
		MaxLag    int    `json:"max_lag"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ProductID == "" || req.Symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id and symbol are required"})
		return
	}
	if req.MaxLag <= 0 { req.MaxLag = 21 }
	start, err1 := time.Parse("2006-01-02", req.Start)
	end, err2 := time.Parse("2006-01-02", req.End)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start/end format (use YYYY-MM-DD)"})
		return
	}
	// Fetch both series from Qdrant
	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()
	econ, err := h.vs.GetEconomicSeries(ctx, req.Symbol, start, end)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	sales, err := h.vs.GetSalesSeries(ctx, req.ProductID, start, end)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	if len(econ) < 5 || len(sales) < 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient data"})
		return
	}
	xDates := make([]string, 0, len(econ))
	xVals := make([]float64, 0, len(econ))
	for _, p := range econ { xDates = append(xDates, p.Date); xVals = append(xVals, p.Value) }
	yDates := make([]string, 0, len(sales))
	yVals := make([]float64, 0, len(sales))
	for _, p := range sales { yDates = append(yDates, p.Date); yVals = append(yVals, p.Sales) }
	stats := services.NewStatisticsService(nil, nil)
	results, err := stats.CalculateLaggedCorrelations(xDates, xVals, yDates, yVals, req.MaxLag)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{
		"product_id": req.ProductID,
		"symbol":     req.Symbol,
		"start":      req.Start,
		"end":        req.End,
		"max_lag":    req.MaxLag,
		"results":    results,
		"top":        results[0],
	})
}
