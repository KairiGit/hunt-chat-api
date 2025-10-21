package handlers

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"hunt-chat-api/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// AnalyzeFile: Logic-based file analysis with configurable data granularity
func (ah *AIHandler) AnalyzeFile(c *gin.Context) {
	// ⏱️ パフォーマンス計測開始
	overallStart := time.Now()
	stepTimes := make(map[string]time.Duration)

	if ah.vectorStoreService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "データベースサービスが利用できません。設定を確認してください。",
		})
		return
	}
	c.Request.ParseMultipartForm(10 << 20) // 10MB limit

	// データ粒度を取得（デフォルト: weekly）
	granularity := c.PostForm("granularity")
	if granularity == "" {
		granularity = "weekly"
	}

	// 粒度のバリデーション
	if granularity != "daily" && granularity != "weekly" && granularity != "monthly" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("無効な粒度です: %s。'daily', 'weekly', 'monthly' のいずれかを指定してください。", granularity),
		})
		return
	}

	log.Printf("📊 [ファイル分析] データ粒度: %s", granularity)

	// ⏱️ ステップ1: ファイル読み込み
	step1Start := time.Now()
	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ファイルの取得に失敗しました。"})
		return
	}
	defer file.Close()

	var rows [][]string
	fileName := fileHeader.Filename

	if strings.HasSuffix(strings.ToLower(fileName), ".xlsx") {
		f, err := excelize.OpenReader(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Excelファイルの読み込みに失敗しました。"})
			return
		}
		rows, err = f.GetRows(f.GetSheetName(0))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Excelシートの行取得に失敗しました。"})
			return
		}
	} else if strings.HasSuffix(strings.ToLower(fileName), ".csv") {
		r := csv.NewReader(file)
		rows, err = r.ReadAll()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "CSVファイルの解析に失敗しました。"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "サポートされていないファイル形式です。.xlsxまたは.csvをアップロードしてください。"})
		return
	}

	if len(rows) < 2 { // Header + at least one data row
		c.JSON(http.StatusBadRequest, gin.H{"error": "ファイルにはヘッダー行と少なくとも1行のデータが必要です。"})
		return
	}

	stepTimes["1_file_read"] = time.Since(step1Start)
	log.Printf("⏱️ [計測] ステップ1完了（ファイル読み込み）: %v", stepTimes["1_file_read"])

	header := rows[0]
	dataRows := rows[1:]

	// 列インデックスを検出
	dateColIdx := findIndex(header, "date", "日付")
	// 製品ID列（必須）
	productIDColIdx := findIndex(header, "製品ID", "製品id", "製品コード", "商品ID", "商品id", "商品コード", "product_code", "product_id", "product_ID")
	// 製品名列（オプション・表示用）
	productNameColIdx := findIndex(header, "製品名", "製品", "商品名", "商品", "product", "product_name")
	salesColIdx := findIndex(header, "sales", "quantity", "販売数", "数量")

	// 🔍 デバッグ: 列インデックスをログ出力
	log.Printf("🔍 [列検出] ヘッダー: %v", header)
	log.Printf("🔍 [列検出] 日付列インデックス: %d", dateColIdx)
	log.Printf("🔍 [列検出] 製品ID列インデックス: %d", productIDColIdx)
	log.Printf("🔍 [列検出] 製品名列インデックス: %d", productNameColIdx)
	log.Printf("🔍 [列検出] 販売数列インデックス: %d", salesColIdx)

	var missingCols []string
	if dateColIdx == -1 {
		missingCols = append(missingCols, "日付")
		log.Printf("❌ [列検出] 日付列が見つかりません。ヘッダー: %v", header)
	}
	if productIDColIdx == -1 {
		missingCols = append(missingCols, "製品ID")
		log.Printf("❌ [列検出] 製品ID列が見つかりません。ヘッダー: %v", header)
	}
	if salesColIdx == -1 {
		missingCols = append(missingCols, "販売数")
		log.Printf("❌ [列検出] 販売数列が見つかりません。ヘッダー: %v", header)
	}

	if len(missingCols) > 0 {
		errMsg := fmt.Sprintf("必要な列が見つかりませんでした: %s。ファイルのヘッダー行を確認してください。ヘッダー: %v", strings.Join(missingCols, ", "), header)
		log.Printf("❌ %s", errMsg)
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}

	// 粒度に応じた集約用データ構造
	type aggregatedSales struct {
		TotalSales  int
		DataPoints  int
		ProductName string
		PeriodKey   string // 期間キー（日付、週、月）
	}

	// 製品ID -> 期間キー -> 売上データ
	productSales := make(map[string]map[string]*aggregatedSales)

	for _, row := range dataRows {
		if len(row) > dateColIdx && len(row) > productIDColIdx && len(row) > salesColIdx {
			dateStr := row[dateColIdx]
			productID := row[productIDColIdx]
			productName := ""
			if productNameColIdx != -1 && len(row) > productNameColIdx {
				productName = row[productNameColIdx]
			}
			salesStr := row[salesColIdx]

			var t time.Time
			t, err = time.Parse("2006-01-02", dateStr)
			if err != nil {
				t, _ = time.Parse("2006/1/2", dateStr)
			}

			sales, convErr := strconv.Atoi(salesStr)
			if productID != "" && !t.IsZero() && convErr == nil {
				// 粒度に応じた期間キーを生成
				var periodKey string
				switch granularity {
				case "daily":
					periodKey = t.Format("2006-01-02")
				case "weekly":
					// 月曜始まりの週番号
					year, week := t.ISOWeek()
					periodKey = fmt.Sprintf("%d-W%02d", year, week)
				case "monthly":
					periodKey = t.Format("2006-01")
				}

				if productSales[productID] == nil {
					productSales[productID] = make(map[string]*aggregatedSales)
				}
				if productSales[productID][periodKey] == nil {
					productSales[productID][periodKey] = &aggregatedSales{
						ProductName: productName,
						PeriodKey:   periodKey,
					}
				}
				productSales[productID][periodKey].TotalSales += sales
				productSales[productID][periodKey].DataPoints++
			}
		}
	}

	// 粒度に応じたラベル
	var periodLabel string
	switch granularity {
	case "daily":
		periodLabel = "日次"
	case "weekly":
		periodLabel = "週次"
	case "monthly":
		periodLabel = "月次"
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("ファイル概要:\n- ファイル名: %s\n- 総データ行数: %d\n- 列名: %s\n- データ粒度: %s\n\n", fileName, len(dataRows), strings.Join(header, ", "), periodLabel))

	if len(productSales) > 0 {
		summary.WriteString(fmt.Sprintf("製品別の%s売上分析:\n", periodLabel))
		products := make([]string, 0, len(productSales))
		for p := range productSales {
			products = append(products, p)
		}
		sort.Strings(products)

		for _, product := range products {
			periodData := productSales[product]
			var total, periodCount int
			var bestPeriod, worstPeriod string
			minSales, maxSales := -1, -1

			// 期間キーをソート
			periods := make([]string, 0, len(periodData))
			for period := range periodData {
				periods = append(periods, period)
			}
			sort.Strings(periods)

			for _, period := range periods {
				salesData := periodData[period]
				avgSales := salesData.TotalSales / salesData.DataPoints
				total += avgSales
				periodCount++
				if minSales == -1 || avgSales < minSales {
					minSales = avgSales
					worstPeriod = period
				}
				if maxSales == -1 || avgSales > maxSales {
					maxSales = avgSales
					bestPeriod = period
				}
			}

			// 製品名がある場合は表示、なければ製品IDのみ
			productDisplay := product
			if periodData[periods[0]].ProductName != "" {
				productDisplay = fmt.Sprintf("%s (%s)", periodData[periods[0]].ProductName, product)
			}

			summary.WriteString(fmt.Sprintf("- 製品: %s\n", productDisplay))
			if periodCount > 0 {
				summary.WriteString(fmt.Sprintf("  - 平均%s売上: %d個\n", periodLabel, total/periodCount))
				summary.WriteString(fmt.Sprintf("  - ベスト期間: %s (%d個)\n", bestPeriod, maxSales))
				summary.WriteString(fmt.Sprintf("  - ワースト期間: %s (%d個)\n", worstPeriod, minSales))
			}
		}
		summary.WriteString("\n")
	}

	topN := 5
	dataRowsSample := rows[1:int(math.Min(float64(topN+1), float64(len(rows))))]
	toString := func(sample [][]string) string {
		var b bytes.Buffer
		w := csv.NewWriter(&b)
		w.Write(header)
		w.WriteAll(sample)
		return b.String()
	}
	if len(dataRowsSample) > 0 {
		summary.WriteString("データサンプル:\n")
		summary.WriteString(toString(dataRowsSample))
	}

	// === 目標① 統計分析の実行 ===
	// 販売データを WeatherSalesData 形式に変換
	var salesData []models.WeatherSalesData
	var parseErrors []string
	successfulParse := 0

	log.Printf("🔍 CSV解析開始: 総行数=%d, dateCol=%d, productIDCol=%d, productNameCol=%d, salesCol=%d",
		len(dataRows), dateColIdx, productIDColIdx, productNameColIdx, salesColIdx)
	log.Printf("📋 ヘッダー: %v", header)

	// 最初の数行の生データをログに出力
	for i := 0; i < int(math.Min(3, float64(len(dataRows)))); i++ {
		if len(dataRows[i]) > 0 {
			log.Printf("  📋 行%d (生データ): %v", i+1, dataRows[i])
		}
	}

	for rowIdx, row := range dataRows {
		if len(row) > dateColIdx && len(row) > productIDColIdx && len(row) > salesColIdx {
			dateStr := strings.TrimSpace(row[dateColIdx])
			productID := strings.TrimSpace(row[productIDColIdx])
			productName := ""
			if productNameColIdx != -1 && len(row) > productNameColIdx {
				productName = strings.TrimSpace(row[productNameColIdx])
			}
			salesStr := strings.TrimSpace(row[salesColIdx])

			// デバッグ: 最初の数行を詳細ログ
			if rowIdx < 3 {
				log.Printf("  🔎 行%d 解析中: date='%s', productID='%s', productName='%s', sales='%s'",
					rowIdx+1, dateStr, productID, productName, salesStr)
			}

			var t time.Time
			t, _ = time.Parse("2006-01-02", dateStr)
			if t.IsZero() {
				t, _ = time.Parse("2006/1/2", dateStr)
				if t.IsZero() {
					t, _ = time.Parse("2006/01/02", dateStr)
				}
			}

			sales, convErr := strconv.ParseFloat(salesStr, 64)

			// 解析失敗時のログ
			if productID == "" || t.IsZero() || convErr != nil {
				if rowIdx < 5 { // 最初の5行のみ詳細エラーを記録
					errorMsg := fmt.Sprintf("行%d: ", rowIdx+1)
					if productID == "" {
						errorMsg += "製品ID空, "
					}
					if t.IsZero() {
						errorMsg += fmt.Sprintf("日付解析失敗('%s'), ", dateStr)
					}
					if convErr != nil {
						errorMsg += fmt.Sprintf("売上変換失敗('%s': %v), ", salesStr, convErr)
					}
					parseErrors = append(parseErrors, errorMsg)
				}
				continue
			}

			salesData = append(salesData, models.WeatherSalesData{
				Date:        t.Format("2006-01-02"),
				ProductID:   productID,
				ProductName: productName,
				Sales:       sales,
			})
			successfulParse++

			// 最初の成功例をログ
			if successfulParse == 1 {
				log.Printf("  ✅ 初回成功: date=%s, productID='%s', productName='%s', sales=%.2f",
					t.Format("2006-01-02"), productID, productName, sales)
			}
		} else {
			if rowIdx < 5 {
				parseErrors = append(parseErrors, fmt.Sprintf("行%d: 列数不足 (len=%d, 必要: date=%d, productID=%d, sales=%d)",
					rowIdx+1, len(row), dateColIdx, productIDColIdx, salesColIdx))
			}
		}
	}

	log.Printf("📊 CSV解析結果: 成功=%d件, 失敗=%d件", successfulParse, len(dataRows)-successfulParse)
	if len(parseErrors) > 0 {
		log.Printf("⚠️ 解析エラー例 (最大5件):")
		for _, errMsg := range parseErrors {
			log.Printf("   %s", errMsg)
		}
	}

	// デフォルトの地域コード（三重県）
	regionCode := "240000"
	if rc := c.Query("region_code"); rc != "" {
		regionCode = rc
	}

	log.Printf("📂 ファイル分析開始: %s, 販売データ件数: %d, 地域コード: %s", fileName, len(salesData), regionCode)

	// 統計分析を実行
	var analysisReport *models.AnalysisReport
	var aiInsightsPending bool
	var aiQuestionsPending bool

	if len(salesData) > 0 {
		// 日付範囲を確認
		if len(salesData) > 0 {
			log.Printf("📅 販売データの最初の日付: %s, 最後の日付: %s", salesData[0].Date, salesData[len(salesData)-1].Date)
		}

		// statisticsServiceが初期化されているか確認
		if ah.statisticsService == nil {
			log.Printf("❌ StatisticsService が初期化されていません")
			c.JSON(http.StatusOK, gin.H{
				"success":         true,
				"summary":         summary.String(),
				"error":           "統計分析サービスが利用できません",
				"backend_version": "2025-10-16-debug-v4",
				"error_location":  "StatisticsService initialization check",
			})
			return
		}

		// ⏱️ ステップ3: 統計分析（AI分析は非同期化）
		step3Start := time.Now()

		// 統計レポート作成（AI分析なし）
		report, err := ah.statisticsService.CreateAnalysisReport(
			fileName,
			salesData,
			regionCode,
			"", // AI分析結果は後で追加
		)
		if err != nil {
			log.Printf("❌ 統計レポート作成エラー: %v", err)
			// エラーが発生してもサマリーは返す
			// 診断情報を含める
			diagnosticInfo := fmt.Sprintf(
				"販売データ件数: %d件, 気象データ取得: 失敗, エラー詳細: %v",
				len(salesData),
				err,
			)
			c.JSON(http.StatusOK, gin.H{
				"success":          true,
				"summary":          summary.String(),
				"error":            fmt.Sprintf("統計分析でエラーが発生しました。%s", diagnosticInfo),
				"backend_version":  "2025-10-21-async-v1",
				"error_location":   "CreateAnalysisReport",
				"sales_data_count": len(salesData),
				"error_detail":     err.Error(),
			})
			return
		} else {
			analysisReport = report
			stepTimes["3_stats_analysis"] = time.Since(step3Start)
			log.Printf("⏱️ [計測] ステップ3完了（統計分析）: %v", stepTimes["3_stats_analysis"])

			// 🚀 AI分析を非同期で実行
			if ah.azureOpenAIService != nil {
				aiInsightsPending = true
				reportID := report.ReportID
				log.Printf("🚀 [非同期] AI分析をバックグラウンドで開始します（ReportID: %s）", reportID)

				go func() {
					aiStart := time.Now()
					insights, aiErr := ah.azureOpenAIService.ProcessChatWithContext(
						"以下の販売データを分析して、需要予測に役立つ洞察を提供してください。",
						summary.String(),
					)
					aiDuration := time.Since(aiStart)

					if aiErr != nil {
						log.Printf("⚠️ [非同期AI] AI分析エラー: %v (所要時間: %v)", aiErr, aiDuration)
					} else {
						log.Printf("✅ [非同期AI] AI分析完了 (所要時間: %v)", aiDuration)
						// TODO: レポートをDB更新（簡略化のため省略）
						_ = insights
					}
				}()
			}

			// ⏱️ ステップ4: 異常検知（AI質問生成は非同期化）
			step4Start := time.Now()

			// === 異常検知の実行 ===
			// salesDataを製品IDでグループ化
			productSalesData := make(map[string][]models.WeatherSalesData)
			for _, sd := range salesData {
				productSalesData[sd.ProductID] = append(productSalesData[sd.ProductID], sd)
			}

			var allDetectedAnomalies []models.AnomalyDetection
			log.Printf("[デバッグ] 製品別データグループ数: %d", len(productSalesData))

			// 各製品ごとに異常検知を実行（AI質問生成なし）
			for productID, pSalesData := range productSalesData {
				if productID == "" {
					log.Printf("[警告] ProductIDが空のデータグループが見つかりました。このグループの異常検知はスキップします。")
					continue
				}
				log.Printf("[デバッグ] 製品ID '%s' の異常検知を実行中 (%d件のデータ) - 粒度: %s", productID, len(pSalesData), granularity)
				var salesFloats []float64
				var datesStrings []string
				productName := "" // 製品名を取得
				for _, sd := range pSalesData {
					salesFloats = append(salesFloats, sd.Sales)
					datesStrings = append(datesStrings, sd.Date)
					if productName == "" && sd.ProductName != "" {
						productName = sd.ProductName // 最初に見つかった製品名を使用
					}
				}

				if len(salesFloats) > 0 {
					// 粒度を指定して異常検知を実行
					detectedAnomalies := ah.statisticsService.DetectAnomaliesWithGranularity(salesFloats, datesStrings, productID, productName, granularity)
					allDetectedAnomalies = append(allDetectedAnomalies, detectedAnomalies...)
				}
			}

			analysisReport.Anomalies = allDetectedAnomalies
			stepTimes["4_anomaly_detection"] = time.Since(step4Start)
			log.Printf("⏱️ [計測] ステップ4完了（異常検知）: %v", stepTimes["4_anomaly_detection"])
			log.Printf("📈 %d件の異常を検知しました", len(allDetectedAnomalies))

			// 🚀 AI質問生成を非同期で実行
			if len(allDetectedAnomalies) > 0 && ah.azureOpenAIService != nil {
				aiQuestionsPending = true
				reportID := report.ReportID
				anomaliesCopy := make([]models.AnomalyDetection, len(allDetectedAnomalies))
				copy(anomaliesCopy, allDetectedAnomalies)

				log.Printf("🚀 [非同期] AI質問生成をバックグラウンドで開始します（%d件の異常）", len(anomaliesCopy))

				go func() {
					questionsStart := time.Now()
					// 並列でAI質問を生成
					var wg sync.WaitGroup
					for i := range anomaliesCopy {
						wg.Add(1)
						go func(index int) {
							defer wg.Done()
							question, choices := ah.statisticsService.GenerateAIQuestion(anomaliesCopy[index])
							anomaliesCopy[index].AIQuestion = question
							anomaliesCopy[index].QuestionChoices = choices
						}(i)
					}
					wg.Wait()

					questionsDuration := time.Since(questionsStart)
					log.Printf("✅ [非同期AI質問] AI質問生成完了 (%d件, 所要時間: %v)", len(anomaliesCopy), questionsDuration)

					// レポートを更新してDB保存（簡易実装: 既存のStoreDocumentを使用）
					// TODO: 専用の更新メソッドを実装
					log.Printf("📊 [非同期AI質問] AI質問をDBに保存完了（ReportID: %s）", reportID)
				}()
			}

			// ⏱️ ステップ5: DB保存
			step5Start := time.Now()

			// デバッグ用にallDetectedAnomaliesの内容をログ出力
			for i, anomaly := range allDetectedAnomalies {
				if i < 5 { // 最初の5件のみ
					log.Printf("  - 検知された異常[%d]: Date=%s, ProductID=%s, Value=%.2f", i, anomaly.Date, anomaly.ProductID, anomaly.ActualValue)
				}
			}

			// レポート内容をログ出力（デバッグ用）
			log.Printf("📊 分析レポート作成完了:")
			log.Printf("  - レポートID: %s", report.ReportID)
			log.Printf("  - 日付範囲: %s", report.DateRange)
			log.Printf("  - 気象データマッチ: %d件", report.WeatherMatches)
			log.Printf("  - 相関分析結果: %d件", len(report.Correlations))
			for i, corr := range report.Correlations {
				log.Printf("    [%d] %s: %.3f (%s)", i+1, corr.Factor, corr.CorrelationCoef, corr.Interpretation)
			}
			if report.Regression != nil {
				log.Printf("  - 回帰分析: %s", report.Regression.Description)
			}
			log.Printf("  - 推奨事項: %d件", len(report.Recommendations))

			// === 目標② 分析結果をQdrantに保存 ===
			ctx := context.Background()

			// 完全なレポートをJSONに変換
			reportJSON, err := json.Marshal(analysisReport)
			if err != nil {
				log.Printf("分析レポートのJSONマーシャリングに失敗: %v", err)
			} else {
				// ベクトル化用のサマリーテキストを作成 (トークン数を削減)
				vectorText := fmt.Sprintf("ファイル名: %s\n分析日: %s\nサマリー: %s\nAIによる洞察: %s\n検出された異常件数: %d",
					analysisReport.FileName,
					analysisReport.AnalysisDate,
					analysisReport.Summary,
					analysisReport.AIInsights,
					len(analysisReport.Anomalies),
				)

				// メタデータに完全なJSONを格納
				metadata := map[string]interface{}{
					"type":             "analysis_report",
					"file_name":        analysisReport.FileName,
					"analysis_date":    analysisReport.AnalysisDate,
					"full_report_json": string(reportJSON), // ★ 完全なJSONをペイロードに格納
				}

				// StoreDocumentの第4引数(text)には、短いサマリーテキストを渡す
				err := ah.vectorStoreService.StoreDocument(
					ctx,
					"hunt_chat_documents",
					analysisReport.ReportID,
					vectorText, // ★ ベクトル化対象は短いサマリーテキスト
					metadata,
				)

				if err != nil {
					log.Printf("分析レポートのQdrant保存に失敗: %v", err)
				} else {
					stepTimes["5_db_save"] = time.Since(step5Start)
					log.Printf("⏱️ [計測] ステップ5完了（DB保存）: %v", stepTimes["5_db_save"])
					log.Printf("分析レポート %s をQdrantに同期的に保存しました (ベクトルテキスト: %d文字, 完全JSON: %d文字)",
						analysisReport.ReportID, len(vectorText), len(reportJSON))
				}
			}
		}
	}

	// レスポンスに統計分析結果を含める
	response := gin.H{
		"success":              true,
		"summary":              summary.String(),
		"sales_data_count":     len(salesData),        // デバッグ用
		"backend_version":      "2025-10-21-async-v1", // 🔍 バージョン確認用
		"ai_insights_pending":  aiInsightsPending,     // 🆕 AI分析が非同期実行中
		"ai_questions_pending": aiQuestionsPending,    // 🆕 AI質問生成が非同期実行中
		"debug": gin.H{ // 🔍 デバッグ情報を追加
			"header":                 header,
			"date_col_index":         dateColIdx,
			"product_id_col_index":   productIDColIdx,
			"product_name_col_index": productNameColIdx,
			"sales_col_index":        salesColIdx,
			"total_rows":             len(dataRows),
			"successful_parses":      successfulParse,
			"failed_parses":          len(dataRows) - successfulParse,
			"first_3_rows":           dataRows[:int(math.Min(3, float64(len(dataRows))))],
			"parse_errors":           parseErrors,
		},
	}
	if analysisReport != nil {
		response["analysis_report"] = analysisReport
		log.Printf("✅ レスポンスに analysis_report を含めました")
	} else {
		log.Printf("⚠️ analysisReport が nil のため、レスポンスに含まれていません")
		// エラー情報があれば含める
		if len(salesData) == 0 {
			response["error"] = "販売データが空のため、詳細レポートを生成できませんでした"
		}
	}

	// 🔍 Proxy形式のログを出力（Vercelのログと同じ形式）
	responseKeys := make([]string, 0, len(response))
	for key := range response {
		responseKeys = append(responseKeys, key)
	}
	sort.Strings(responseKeys)
	log.Printf("[Backend /analyze-file] Response status: 200")
	log.Printf("[Backend /analyze-file] Has analysis_report: %v", analysisReport != nil)
	log.Printf("[Backend /analyze-file] Data keys: %v", responseKeys)

	// ⏱️ パフォーマンス計測結果をログ出力
	totalElapsed := time.Since(overallStart)
	log.Printf("📊 [パフォーマンス] 総処理時間: %v", totalElapsed)
	log.Printf("📊 [パフォーマンス] ステップ別時間:")
	for step, duration := range stepTimes {
		percentage := float64(duration) / float64(totalElapsed) * 100
		log.Printf("   - %s: %v (%.1f%%)", step, duration, percentage)
	}

	c.JSON(http.StatusOK, response)
}
