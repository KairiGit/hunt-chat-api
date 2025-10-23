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
	gran := c.DefaultQuery("granularity", "daily") // daily|weekly|monthly
	// source: qdrant (no fallback), csv (csv only), auto (try qdrant then fallback to csv)
	source := strings.ToLower(c.DefaultQuery("source", "qdrant"))
	startStr := c.DefaultQuery("start", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	endStr := c.DefaultQuery("end", time.Now().Format("2006-01-02"))

	start, err1 := time.Parse("2006-01-02", startStr)
	end, err2 := time.Parse("2006-01-02", endStr)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start/end format (use YYYY-MM-DD)"})
		return
	}

	var out []gin.H
	// Prefer Qdrant unless explicitly forced to CSV
	if h.vs != nil && source != "csv" {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
		defer cancel()
		if strings.ToLower(gran) == "daily" {
			econ, err := h.vs.GetEconomicSeries(ctx, symbol, start, end)
			if err == nil && len(econ) > 0 {
				out = make([]gin.H, 0, len(econ))
				for _, p := range econ {
					out = append(out, gin.H{"date": p.Date, "value": p.Value})
				}
			}
		} else {
			// aggregated from stored aggregates if available; fallback to on-the-fly aggregation
			econAgg, err := h.vs.GetEconomicAggregatedSeries(ctx, symbol, start, end, strings.ToLower(gran))
			if err == nil && len(econAgg) > 0 {
				out = make([]gin.H, 0, len(econAgg))
				for _, p := range econAgg {
					out = append(out, gin.H{"period": p.Period, "value": p.Value})
				}
			} else {
				// fallback: fetch daily then aggregate on the fly
				econ, err2 := h.vs.GetEconomicSeries(ctx, symbol, start, end)
				if err2 == nil && len(econ) > 0 {
					periods := services.AggregateSeriesDailyTo(econ, strings.ToLower(gran), "last")
					out = make([]gin.H, 0, len(periods))
					for _, p := range periods {
						out = append(out, gin.H{"period": p.Period, "value": p.Value})
					}
				}
			}
		}
	}
	// Fallback to CSV only when explicitly requested (source=csv) or auto
	if len(out) == 0 && (source == "csv" || source == "auto") {
		series, err := h.svc.GetMarketSeries(symbol, start, end)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		out = make([]gin.H, 0, len(series))
		for _, p := range series {
			out = append(out, gin.H{"date": p.Date.Format("2006-01-02"), "value": p.Value})
		}
		gran = "daily"
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":      symbol,
		"start":       start.Format("2006-01-02"),
		"end":         end.Format("2006-01-02"),
		"granularity": gran,
		"series":      out,
		"count":       len(out),
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

// GetSalesSeries returns sales series from Qdrant for a product, optionally aggregated.
// Query: product_id (required), start, end (YYYY-MM-DD), granularity=daily|weekly|monthly
func (h *EconomicHandler) GetSalesSeries(c *gin.Context) {
	if h.vs == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vector store unavailable"})
		return
	}
	productID := c.Query("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id is required"})
		return
	}
	gran := strings.ToLower(c.DefaultQuery("granularity", "daily"))
	startStr := c.DefaultQuery("start", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	endStr := c.DefaultQuery("end", time.Now().Format("2006-01-02"))
	start, err1 := time.Parse("2006-01-02", startStr)
	end, err2 := time.Parse("2006-01-02", endStr)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start/end format (use YYYY-MM-DD)"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	if gran == "daily" {
		series, err := h.vs.GetSalesSeries(ctx, productID, start, end)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out := make([]gin.H, 0, len(series))
		for _, p := range series {
			out = append(out, gin.H{"date": p.Date, "sales": p.Sales})
		}
		c.JSON(http.StatusOK, gin.H{"product_id": productID, "granularity": gran, "series": out, "count": len(out)})
		return
	}
	// aggregated
	agg, err := h.vs.GetSalesAggregatedSeries(ctx, productID, start, end, gran)
	if err == nil && len(agg) > 0 {
		out := make([]gin.H, 0, len(agg))
		for _, p := range agg {
			out = append(out, gin.H{"period": p.Period, "sales": p.Value})
		}
		c.JSON(http.StatusOK, gin.H{"product_id": productID, "granularity": gran, "series": out, "count": len(out)})
		return
	}
	// fallback: aggregate on the fly
	daily, err := h.vs.GetSalesSeries(ctx, productID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	periods := services.AggregateSalesDailyTo(daily, gran, "sum")
	out := make([]gin.H, 0, len(periods))
	for _, p := range periods {
		out = append(out, gin.H{"period": p.Period, "sales": p.Value})
	}
	c.JSON(http.StatusOK, gin.H{"product_id": productID, "granularity": gran, "series": out, "count": len(out)})
}

// AggregateEconomic aggregates daily economic series to weekly/monthly and stores to Qdrant.
// JSON: { symbol, start, end, granularity, method? }
func (h *EconomicHandler) AggregateEconomic(c *gin.Context) {
	if h.vs == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vector store unavailable"})
		return
	}
	var req struct {
		Symbol      string `json:"symbol"`
		Start       string `json:"start"`
		End         string `json:"end"`
		Granularity string `json:"granularity"`
		Method      string `json:"method"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Symbol == "" || req.Granularity == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol and granularity are required"})
		return
	}
	start, err1 := time.Parse("2006-01-02", req.Start)
	end, err2 := time.Parse("2006-01-02", req.End)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start/end format (use YYYY-MM-DD)"})
		return
	}
	if req.Method == "" {
		req.Method = "last"
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	daily, err := h.vs.GetEconomicSeries(ctx, req.Symbol, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	periods := services.AggregateSeriesDailyTo(daily, strings.ToLower(req.Granularity), req.Method)
	if len(periods) == 0 {
		c.JSON(http.StatusOK, gin.H{"symbol": req.Symbol, "stored": 0})
		return
	}
	if err := h.vs.StoreEconomicAggregates(ctx, req.Symbol, strings.ToLower(req.Granularity), req.Method, periods); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"symbol": req.Symbol, "stored": len(periods)})
}

// AggregateSales aggregates daily sales series to weekly/monthly and stores to Qdrant.
// JSON: { product_id, start, end, granularity, method? }
func (h *EconomicHandler) AggregateSales(c *gin.Context) {
	if h.vs == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vector store unavailable"})
		return
	}
	var req struct {
		ProductID   string `json:"product_id"`
		Start       string `json:"start"`
		End         string `json:"end"`
		Granularity string `json:"granularity"`
		Method      string `json:"method"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ProductID == "" || req.Granularity == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id and granularity are required"})
		return
	}
	start, err1 := time.Parse("2006-01-02", req.Start)
	end, err2 := time.Parse("2006-01-02", req.End)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start/end format (use YYYY-MM-DD)"})
		return
	}
	if req.Method == "" {
		req.Method = "sum"
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	daily, err := h.vs.GetSalesSeries(ctx, req.ProductID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	periods := services.AggregateSalesDailyTo(daily, strings.ToLower(req.Granularity), req.Method)
	if len(periods) == 0 {
		c.JSON(http.StatusOK, gin.H{"product_id": req.ProductID, "stored": 0})
		return
	}
	if err := h.vs.StoreSalesAggregates(ctx, req.ProductID, strings.ToLower(req.Granularity), req.Method, periods); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_id": req.ProductID, "stored": len(periods)})
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
	type salesPoint struct {
		Date  string  `json:"date"`
		Sales float64 `json:"sales"`
	}
	var input struct {
		Symbol string       `json:"symbol"`
		Start  string       `json:"start"`
		End    string       `json:"end"`
		Sales  []salesPoint `json:"sales"`
		MaxLag int          `json:"max_lag"`
		Gran   string       `json:"granularity"` // optional: daily|weekly|monthly
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	if input.Symbol == "" || len(input.Sales) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol and sales are required"})
		return
	}
	if input.MaxLag <= 0 {
		input.MaxLag = 14
	}
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
	// Fetch economic series from Qdrant (optionally aggregated)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	var econ []struct {
		Date  string
		Value float64
	}
	var err error
	if strings.ToLower(input.Gran) == "weekly" || strings.ToLower(input.Gran) == "monthly" {
		agg, err := h.vs.GetEconomicAggregatedSeries(ctx, input.Symbol, start, end, strings.ToLower(input.Gran))
		if err == nil && len(agg) > 0 {
			econ = make([]struct {
				Date  string
				Value float64
			}, 0, len(agg))
			for _, p := range agg {
				econ = append(econ, struct {
					Date  string
					Value float64
				}{Date: p.EndDate, Value: p.Value})
			}
		} else {
			dly, err2 := h.vs.GetEconomicSeries(ctx, input.Symbol, start, end)
			if err2 == nil && len(dly) > 0 {
				ap := services.AggregateSeriesDailyTo(dly, strings.ToLower(input.Gran), "last")
				econ = make([]struct {
					Date  string
					Value float64
				}, 0, len(ap))
				for _, p := range ap {
					econ = append(econ, struct {
						Date  string
						Value float64
					}{Date: p.EndDate, Value: p.Value})
				}
			}
		}
	} else {
		econ, err = h.vs.GetEconomicSeries(ctx, input.Symbol, start, end)
	}
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
	stats := services.NewStatisticsService(nil, nil, nil)
	results, err := stats.CalculateLaggedCorrelations(xDates, xVals, yDates, yVals, input.MaxLag)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"symbol":      input.Symbol,
		"start":       input.Start,
		"end":         input.End,
		"max_lag":     input.MaxLag,
		"granularity": strings.ToLower(input.Gran),
		"results":     results,
		"top":         results[0],
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
	batch := make([]struct {
		Date  string
		Sales float64
	}, 0, len(req.Data))
	for _, d := range req.Data {
		if d.Date == "" {
			continue
		}
		if latest == "" || d.Date > latest {
			batch = append(batch, struct {
				Date  string
				Sales float64
			}{Date: d.Date, Sales: d.Sales})
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
		Gran      string `json:"granularity"` // daily|weekly|monthly
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ProductID == "" || req.Symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id and symbol are required"})
		return
	}
	if req.MaxLag <= 0 {
		req.MaxLag = 21
	}
	start, err1 := time.Parse("2006-01-02", req.Start)
	end, err2 := time.Parse("2006-01-02", req.End)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start/end format (use YYYY-MM-DD)"})
		return
	}
	// Fetch both series from Qdrant
	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()
	var econ []struct {
		Date  string
		Value float64
	}
	var err error
	if strings.ToLower(req.Gran) == "weekly" || strings.ToLower(req.Gran) == "monthly" {
		agg, err := h.vs.GetEconomicAggregatedSeries(ctx, req.Symbol, start, end, strings.ToLower(req.Gran))
		if err == nil && len(agg) > 0 {
			for _, p := range agg {
				econ = append(econ, struct {
					Date  string
					Value float64
				}{Date: p.EndDate, Value: p.Value})
			}
		} else {
			dly, err2 := h.vs.GetEconomicSeries(ctx, req.Symbol, start, end)
			if err2 == nil && len(dly) > 0 {
				ap := services.AggregateSeriesDailyTo(dly, strings.ToLower(req.Gran), "last")
				for _, p := range ap {
					econ = append(econ, struct {
						Date  string
						Value float64
					}{Date: p.EndDate, Value: p.Value})
				}
			}
		}
	} else {
		econ, err = h.vs.GetEconomicSeries(ctx, req.Symbol, start, end)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// sales series (optionally aggregated)
	var sales []struct {
		Date  string
		Sales float64
	}
	var errSales error
	if strings.ToLower(req.Gran) == "weekly" || strings.ToLower(req.Gran) == "monthly" {
		agg, err := h.vs.GetSalesAggregatedSeries(ctx, req.ProductID, start, end, strings.ToLower(req.Gran))
		if err == nil && len(agg) > 0 {
			for _, p := range agg {
				sales = append(sales, struct {
					Date  string
					Sales float64
				}{Date: p.EndDate, Sales: p.Value})
			}
		} else {
			dly, err2 := h.vs.GetSalesSeries(ctx, req.ProductID, start, end)
			if err2 == nil && len(dly) > 0 {
				ap := services.AggregateSalesDailyTo(dly, strings.ToLower(req.Gran), "sum")
				for _, p := range ap {
					sales = append(sales, struct {
						Date  string
						Sales float64
					}{Date: p.EndDate, Sales: p.Value})
				}
			}
		}
	} else {
		sales, errSales = h.vs.GetSalesSeries(ctx, req.ProductID, start, end)
	}
	if errSales != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errSales.Error()})
		return
	}
	if len(econ) < 5 || len(sales) < 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient data"})
		return
	}
	xDates := make([]string, 0, len(econ))
	xVals := make([]float64, 0, len(econ))
	for _, p := range econ {
		xDates = append(xDates, p.Date)
		xVals = append(xVals, p.Value)
	}
	yDates := make([]string, 0, len(sales))
	yVals := make([]float64, 0, len(sales))
	for _, p := range sales {
		yDates = append(yDates, p.Date)
		yVals = append(yVals, p.Sales)
	}
	stats := services.NewStatisticsService(nil, nil, nil)
	results, err := stats.CalculateLaggedCorrelations(xDates, xVals, yDates, yVals, req.MaxLag)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"product_id":  req.ProductID,
		"symbol":      req.Symbol,
		"start":       req.Start,
		"end":         req.End,
		"max_lag":     req.MaxLag,
		"granularity": strings.ToLower(req.Gran),
		"results":     results,
		"top":         results[0],
	})
}

// AnalyzeWindowedLag scans sliding windows and returns best lag per window.
// JSON: { product_id, symbol, start, end, max_lag, window_days, step_days, granularity?, transform? ("none"|"diff"|"detrend") }
func (h *EconomicHandler) AnalyzeWindowedLag(c *gin.Context) {
	if h.vs == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vector store unavailable"})
		return
	}
	var req struct {
		ProductID  string `json:"product_id"`
		Symbol     string `json:"symbol"`
		Start      string `json:"start"`
		End        string `json:"end"`
		MaxLag     int    `json:"max_lag"`
		WindowDays int    `json:"window_days"`
		StepDays   int    `json:"step_days"`
		Gran       string `json:"granularity"`
		Transform  string `json:"transform"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ProductID == "" || req.Symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id and symbol are required"})
		return
	}
	if req.MaxLag <= 0 {
		req.MaxLag = 21
	}
	if req.WindowDays <= 0 {
		req.WindowDays = 90
	}
	if req.StepDays <= 0 {
		req.StepDays = req.WindowDays / 3
	}
	start, err1 := time.Parse("2006-01-02", req.Start)
	end, err2 := time.Parse("2006-01-02", req.End)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start/end format (use YYYY-MM-DD)"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 35*time.Second)
	defer cancel()
	// econ series (support aggregated)
	var econ []struct {
		Date  string
		Value float64
	}
	var err error
	if strings.ToLower(req.Gran) == "weekly" || strings.ToLower(req.Gran) == "monthly" {
		agg, errA := h.vs.GetEconomicAggregatedSeries(ctx, req.Symbol, start, end, strings.ToLower(req.Gran))
		if errA == nil && len(agg) > 0 {
			for _, p := range agg {
				econ = append(econ, struct {
					Date  string
					Value float64
				}{Date: p.EndDate, Value: p.Value})
			}
		} else {
			dly, err2 := h.vs.GetEconomicSeries(ctx, req.Symbol, start, end)
			if err2 == nil && len(dly) > 0 {
				ap := services.AggregateSeriesDailyTo(dly, strings.ToLower(req.Gran), "last")
				for _, p := range ap {
					econ = append(econ, struct {
						Date  string
						Value float64
					}{Date: p.EndDate, Value: p.Value})
				}
			}
		}
	} else {
		econ, err = h.vs.GetEconomicSeries(ctx, req.Symbol, start, end)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	// sales series
	var sales []struct {
		Date  string
		Sales float64
	}
	if strings.ToLower(req.Gran) == "weekly" || strings.ToLower(req.Gran) == "monthly" {
		agg, errS := h.vs.GetSalesAggregatedSeries(ctx, req.ProductID, start, end, strings.ToLower(req.Gran))
		if errS == nil && len(agg) > 0 {
			for _, p := range agg {
				sales = append(sales, struct {
					Date  string
					Sales float64
				}{Date: p.EndDate, Sales: p.Value})
			}
		} else {
			dly, err2 := h.vs.GetSalesSeries(ctx, req.ProductID, start, end)
			if err2 == nil && len(dly) > 0 {
				ap := services.AggregateSalesDailyTo(dly, strings.ToLower(req.Gran), "sum")
				for _, p := range ap {
					sales = append(sales, struct {
						Date  string
						Sales float64
					}{Date: p.EndDate, Sales: p.Value})
				}
			}
		}
	} else {
		var errS error
		sales, errS = h.vs.GetSalesSeries(ctx, req.ProductID, start, end)
		if errS != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": errS.Error()})
			return
		}
	}
	if len(econ) < 5 || len(sales) < 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient data"})
		return
	}
	// build aligned arrays by dates present in econ
	xDates := make([]string, 0, len(econ))
	xVals := make([]float64, 0, len(econ))
	for _, p := range econ {
		xDates = append(xDates, p.Date)
		xVals = append(xVals, p.Value)
	}
	yMap := make(map[string]float64, len(sales))
	for _, p := range sales {
		yMap[p.Date] = p.Sales
	}
	yDates := make([]string, 0, len(xDates))
	yVals := make([]float64, 0, len(xDates))
	for _, d := range xDates {
		if v, ok := yMap[d]; ok {
			yDates = append(yDates, d)
			yVals = append(yVals, v)
		}
	}
	// transforms
	stats := services.NewStatisticsService(nil, nil, nil)
	switch strings.ToLower(req.Transform) {
	case "diff":
		xVals = stats.FirstDifference(xVals)
		yVals = stats.FirstDifference(yVals)
		// shift dates as well
		if len(xDates) > len(xVals) {
			xDates = xDates[1:]
		}
		if len(yDates) > len(yVals) {
			yDates = yDates[1:]
		}
	case "detrend":
		xVals = stats.Detrend(xVals)
		yVals = stats.Detrend(yVals)
	}
	results, err := stats.CalculateLaggedCorrelationsWindowed(xDates, xVals, yDates, yVals, req.MaxLag, req.WindowDays, req.StepDays)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"product_id": req.ProductID, "symbol": req.Symbol, "granularity": strings.ToLower(req.Gran), "transform": strings.ToLower(req.Transform), "windows": results})
}

// GrangerCausality runs a simple Granger test between econ x and product sales y.
// JSON: { product_id, symbol, start, end, order, granularity? }
func (h *EconomicHandler) GrangerCausality(c *gin.Context) {
	if h.vs == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vector store unavailable"})
		return
	}
	var req struct {
		ProductID string `json:"product_id"`
		Symbol    string `json:"symbol"`
		Start     string `json:"start"`
		End       string `json:"end"`
		Order     int    `json:"order"`
		Gran      string `json:"granularity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ProductID == "" || req.Symbol == "" || req.Order <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id, symbol and positive order are required"})
		return
	}
	start, err1 := time.Parse("2006-01-02", req.Start)
	end, err2 := time.Parse("2006-01-02", req.End)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start/end format (use YYYY-MM-DD)"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 35*time.Second)
	defer cancel()
	// fetch econ and sales, optionally aggregated
	var econ []struct {
		Date  string
		Value float64
	}
	if strings.ToLower(req.Gran) == "weekly" || strings.ToLower(req.Gran) == "monthly" {
		agg, _ := h.vs.GetEconomicAggregatedSeries(ctx, req.Symbol, start, end, strings.ToLower(req.Gran))
		for _, p := range agg {
			econ = append(econ, struct {
				Date  string
				Value float64
			}{Date: p.EndDate, Value: p.Value})
		}
	} else {
		econ, _ = h.vs.GetEconomicSeries(ctx, req.Symbol, start, end)
	}
	var sales []struct {
		Date  string
		Sales float64
	}
	if strings.ToLower(req.Gran) == "weekly" || strings.ToLower(req.Gran) == "monthly" {
		agg, _ := h.vs.GetSalesAggregatedSeries(ctx, req.ProductID, start, end, strings.ToLower(req.Gran))
		for _, p := range agg {
			sales = append(sales, struct {
				Date  string
				Sales float64
			}{Date: p.EndDate, Sales: p.Value})
		}
	} else {
		sales, _ = h.vs.GetSalesSeries(ctx, req.ProductID, start, end)
	}
	if len(econ) < req.Order+5 || len(sales) < req.Order+5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient data"})
		return
	}
	// align series by date intersection
	xMap := map[string]float64{}
	for _, p := range econ {
		xMap[p.Date] = p.Value
	}
	var xVals, yVals []float64
	for _, p := range sales {
		if xv, ok := xMap[p.Date]; ok {
			xVals = append(xVals, xv)
			yVals = append(yVals, p.Sales)
		}
	}
	if len(xVals) < req.Order+5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient overlap"})
		return
	}
	stats := services.NewStatisticsService(nil, nil, nil)
	// Test x->y and y->x
	Fx, px, err := stats.SimpleGrangerCausality(yVals, xVals, req.Order)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	Fy, py, err := stats.SimpleGrangerCausality(xVals, yVals, req.Order)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	direction := "none"
	if px < 0.05 && py >= 0.05 {
		direction = "econ->sales"
	}
	if py < 0.05 && px >= 0.05 {
		direction = "sales->econ"
	}
	if px < 0.05 && py < 0.05 {
		direction = "bidirectional"
	}
	c.JSON(http.StatusOK, gin.H{"product_id": req.ProductID, "symbol": req.Symbol, "order": req.Order, "granularity": strings.ToLower(req.Gran), "x_to_y": gin.H{"F": Fx, "p": px}, "y_to_x": gin.H{"F": Fy, "p": py}, "direction": direction})
}
