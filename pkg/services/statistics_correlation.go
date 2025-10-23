package services

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"hunt-chat-api/pkg/models"
)

// AnalyzeSalesWeatherCorrelation 販売データと気象データの相関を分析（遅れ相関を含む）
func (s *StatisticsService) AnalyzeSalesWeatherCorrelation(
	salesData []models.WeatherSalesData,
	regionCode string,
) ([]models.CorrelationResult, error) {

	if len(salesData) == 0 {
		return nil, fmt.Errorf("販売データが空です")
	}

	// 販売データの日付範囲を特定
	var startDate, endDate time.Time
	for i, data := range salesData {
		t, err := time.Parse("2006-01-02", data.Date)
		if err != nil {
			continue
		}
		if i == 0 || t.Before(startDate) {
			startDate = t
		}
		if i == 0 || t.After(endDate) {
			endDate = t
		}
	}

	// 日付範囲が特定できない場合はデフォルト（過去90日）
	if startDate.IsZero() || endDate.IsZero() {
		endDate = time.Now()
		startDate = endDate.AddDate(0, 0, -90)
	}

	// 気象データを取得（販売データの期間に合わせる）
	weatherData, err := s.weatherService.GetHistoricalWeatherData(regionCode, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("気象データの取得に失敗: %w", err)
	}

	if len(weatherData) == 0 {
		log.Printf("⚠️ 気象データが空です")
		return []models.CorrelationResult{}, nil
	}

	// 販売データの日付と値を抽出
	var salesDates []string
	var salesValues []float64
	for _, sale := range salesData {
		salesDates = append(salesDates, sale.Date)
		salesValues = append(salesValues, sale.Sales)
	}

	// 気象データの日付と値を抽出
	var weatherDates []string
	var tempValues []float64
	var humValues []float64
	for _, w := range weatherData {
		weatherDates = append(weatherDates, w.Date)
		tempValues = append(tempValues, w.Temperature)
		humValues = append(humValues, w.Humidity)
	}

	if len(salesValues) < 5 {
		return nil, fmt.Errorf("販売データが少なすぎます（最低5件必要）")
	}

	// 遅れ相関の最大日数（気象データは短期的な影響が多いため経済データより短く設定）
	maxLagDays := 14 // 最大14日の遅れ相関

	var allResults []models.CorrelationResult

	// 気温との遅れ相関を計算
	tempLaggedCorrs, err := s.CalculateLaggedCorrelations(salesDates, salesValues, weatherDates, tempValues, maxLagDays)
	if err != nil {
		log.Printf("⚠️ 気温の遅れ相関計算エラー: %v", err)
	} else {
		// Factor名に "temperature_" を追加
		for i := range tempLaggedCorrs {
			tempLaggedCorrs[i].Factor = fmt.Sprintf("temperature_%s", tempLaggedCorrs[i].Factor)
		}
		// 統計的に有意な結果のみを追加
		for _, corr := range tempLaggedCorrs {
			if corr.PValue < 0.05 || math.Abs(corr.CorrelationCoef) >= 0.3 {
				allResults = append(allResults, corr)
			}
		}
		log.Printf("✅ 気温の遅れ相関分析完了: %d件の有意な相関を検出", len(tempLaggedCorrs))
	}

	// 湿度との遅れ相関を計算
	humLaggedCorrs, err := s.CalculateLaggedCorrelations(salesDates, salesValues, weatherDates, humValues, maxLagDays)
	if err != nil {
		log.Printf("⚠️ 湿度の遅れ相関計算エラー: %v", err)
	} else {
		// Factor名に "humidity_" を追加
		for i := range humLaggedCorrs {
			humLaggedCorrs[i].Factor = fmt.Sprintf("humidity_%s", humLaggedCorrs[i].Factor)
		}
		// 統計的に有意な結果のみを追加
		for _, corr := range humLaggedCorrs {
			if corr.PValue < 0.05 || math.Abs(corr.CorrelationCoef) >= 0.3 {
				allResults = append(allResults, corr)
			}
		}
		log.Printf("✅ 湿度の遅れ相関分析完了: %d件の有意な相関を検出", len(humLaggedCorrs))
	}

	// 相関係数の絶対値でソート（降順）
	sort.Slice(allResults, func(i, j int) bool {
		return math.Abs(allResults[i].CorrelationCoef) > math.Abs(allResults[j].CorrelationCoef)
	})

	// 上位3件のみを返す（最も有意な相関のみを表示）
	if len(allResults) > 3 {
		allResults = allResults[:3]
		log.Printf("📊 気象データ相関: 上位3件に絞り込みました")
	}

	return allResults, nil
}

// AnalyzeSalesEconomicCorrelation 販売データと経済データの相関を分析（遅れ相関を含む）
func (s *StatisticsService) AnalyzeSalesEconomicCorrelation(
	salesData []models.WeatherSalesData,
	symbols []string,
	maxLagDays int,
) ([]models.CorrelationResult, error) {

	if len(salesData) == 0 {
		return nil, fmt.Errorf("販売データが空です")
	}

	if s.economicService == nil {
		log.Printf("⚠️ EconomicService が初期化されていません")
		return []models.CorrelationResult{}, nil
	}

	// デフォルトのシンボルリスト
	if len(symbols) == 0 {
		symbols = []string{"NIKKEI", "USDJPY", "WTI"}
	}

	// 販売データの日付範囲を特定
	var startDate, endDate time.Time
	for i, data := range salesData {
		t, err := time.Parse("2006-01-02", data.Date)
		if err != nil {
			continue
		}
		if i == 0 || t.Before(startDate) {
			startDate = t
		}
		if i == 0 || t.After(endDate) {
			endDate = t
		}
	}

	// 日付範囲が特定できない場合はデフォルト（過去90日）
	if startDate.IsZero() || endDate.IsZero() {
		endDate = time.Now()
		startDate = endDate.AddDate(0, 0, -90)
	}

	// デフォルトのラグ範囲
	if maxLagDays == 0 {
		maxLagDays = 30 // 最大30日の遅れ相関を調べる
	}

	var allResults []models.CorrelationResult

	// 各経済指標について相関を計算
	for _, symbol := range symbols {
		// 経済データを取得
		economicSeries, err := s.economicService.GetMarketSeries(symbol, startDate, endDate)
		if err != nil {
			log.Printf("⚠️ 経済データ取得エラー (%s): %v", symbol, err)
			continue
		}

		if len(economicSeries) == 0 {
			log.Printf("⚠️ 経済データが空です (%s)", symbol)
			continue
		}

		// 経済データをマップ化
		econMap := make(map[string]float64)
		for _, point := range economicSeries {
			econMap[point.Date.Format("2006-01-02")] = point.Value
		}

		// 販売データの日付と値を抽出
		var salesDates []string
		var salesValues []float64
		for _, sale := range salesData {
			salesDates = append(salesDates, sale.Date)
			salesValues = append(salesValues, sale.Sales)
		}

		// 経済データの日付と値を抽出
		var econDates []string
		var econValues []float64
		for _, point := range economicSeries {
			econDates = append(econDates, point.Date.Format("2006-01-02"))
			econValues = append(econValues, point.Value)
		}

		// 遅れ相関を計算
		laggedCorrs, err := s.CalculateLaggedCorrelations(salesDates, salesValues, econDates, econValues, maxLagDays)
		if err != nil {
			log.Printf("⚠️ 遅れ相関計算エラー (%s): %v", symbol, err)
			continue
		}

		// シンボル名を各相関結果に追加
		for i := range laggedCorrs {
			// Factor名を更新（シンボル名を含める）
			laggedCorrs[i].Factor = fmt.Sprintf("%s_%s", symbol, laggedCorrs[i].Factor)
		}

		// 統計的に有意な結果（p < 0.05）のみを追加
		// または絶対相関係数が0.3以上のものを追加
		for _, corr := range laggedCorrs {
			if corr.PValue < 0.05 || math.Abs(corr.CorrelationCoef) >= 0.3 {
				allResults = append(allResults, corr)
			}
		}

		log.Printf("✅ 経済データ相関分析完了 (%s): %d件の有意な相関を検出", symbol, len(laggedCorrs))
	}

	// 相関係数の絶対値でソート（降順）
	sort.Slice(allResults, func(i, j int) bool {
		return math.Abs(allResults[i].CorrelationCoef) > math.Abs(allResults[j].CorrelationCoef)
	})

	// 上位3件のみを返す（最も有意な相関のみを表示）
	if len(allResults) > 3 {
		allResults = allResults[:3]
		log.Printf("📊 経済データ相関: 上位3件に絞り込みました")
	}

	return allResults, nil
}
