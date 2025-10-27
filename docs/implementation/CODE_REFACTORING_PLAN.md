# コードリファクタリング計画

## 概要
1000行を超える大規模ファイルを機能ごとに分割し、保守性を向上させる。

## 対象ファイル

### 1. statistics_service.go (2421行) ✅進行中
**現状の問題:**
- 50以上の関数が1つのファイルに集約
- 相関分析、異常検知、予測、週次分析など複数の責務が混在

**分割計画:**
- `statistics_core.go` - 基本統計計算（相関、回帰、p値）
- `statistics_math.go` - 数学的ユーティリティ（t分布、ベータ関数） ✅完了
- `statistics_correlation.go` - 相関分析（気象・経済）
- `statistics_anomaly.go` - 異常検知
- `statistics_forecast.go` - 予測・需要予測
- `statistics_weekly.go` - 週次分析

**関数の配置:**

#### statistics_math.go (✅完了)
- `studentTCDF()`
- `regularizedIncompleteBeta()`
- `betacf()`
- `lgamma()`
- `olsRSS()`
- `solveSymmetric()`
- `fDistSurvival()`
- `AdjustPValuesBH()`
- `FirstDifference()`
- `Detrend()`
- `calculateMean()`
- `calculateStandardDeviation()`

#### statistics_core.go (計画中)
- `StatisticsService` struct
- `NewStatisticsService()`
- `CalculateCorrelation()`
- `CalculateLaggedCorrelations()`
- `CalculatePValue()`
- `CalculateLaggedCorrelationsWindowed()`
- `InterpretCorrelation()`
- `PerformLinearRegression()`
- `GenerateStatisticalSummary()`
- `CreateAnalysisReport()`
- `SimpleGrangerCausality()`

#### statistics_correlation.go (計画中)
- `AnalyzeSalesWeatherCorrelation()`
- `AnalyzeSalesEconomicCorrelation()`
- `generateRecommendations()`

#### statistics_anomaly.go (計画中)
- `DetectAnomalies()`
- `DetectAnomaliesWithGranularity()`
- `aggregateDataForAnomalyDetection()`
- `calculateSeverity()`
- `formatDateForDisplay()`
- `GenerateAIQuestion()`

#### statistics_forecast.go (計画中)
- `PredictFutureSales()`
- `ForecastProductDemand()`
- `calculateProductStatistics()`
- `calculateWeekdayEffect()`
- `calculateTrend()`
- `getSeasonalTemperature()`
- `getDayOfWeekJP()`
- `detectSeasonality()`
- `generateForecastRecommendations()`
- `buildFactorsList()`

#### statistics_weekly.go (計画中)
- `AnalyzeWeeklySales()`
- `groupByWeek()`
- `groupByDay()`
- `groupByMonth()`
- `getWeekNumber()`
- `adjustToMonday()`
- `calculateWeeklySummary()`
- `calculateWeeklyOverallStats()`
- `analyzeWeeklyTrends()`
- `generateWeeklyRecommendations()`

---

### 2. vector_store_service.go (1859行)
**現状の問題:**
- ドキュメント保存、セッション管理、コレクション管理が1ファイルに

**分割計画:**
- `vector_store_core.go` - 基本構造とコレクション管理
- `vector_store_document.go` - ドキュメント保存・検索
- `vector_store_session.go` - セッション管理（異常検知レスポンス等）

---

### 3. ai_handler.go (1341行)
**現状の問題:**
- 複数のAI質問生成パターンが1ファイルに

**分割計画:**
- `ai_handler_core.go` - 基本構造とルーティング
- `ai_handler_questions.go` - AI質問生成ロジック
- `ai_handler_analysis.go` - 分析関連処理

---

### 4. weather_service.go (1319行)
**現状の問題:**
- データ取得、変換、分析が混在

**分割計画:**
- `weather_service_core.go` - 基本構造とAPI呼び出し
- `weather_service_data.go` - データ変換・整形
- `weather_service_analysis.go` - 天候データ分析

---

## 実装手順

### フェーズ1: 数学的ユーティリティの分離 ✅
1. `statistics_math.go`作成 ✅
2. テスト実行で動作確認

### フェーズ2: statistics_service.goの完全分割
1. 各ファイルを作成
2. 元のファイルから関数を移動
3. インポート調整
4. テスト実行

### フェーズ3: 他の3ファイルの分割
1. vector_store_service.go
2. ai_handler.go
3. weather_service.go

### フェーズ4: 統合テスト
1. `go test ./...`実行
2. ビルド確認
3. 動作確認

---

## 注意事項
- 各ファイルは500行以下を目標とする
- package は全て`services`を維持
- 公開関数（大文字開始）と非公開関数を明確に区別
- `//go:build ignore`は不要（通常のパッケージファイルのため）

---

## 進捗

### フェーズ1: 完了 ✅
- [x] 計画立案
- [x] statistics_math.go作成 (318行)
- [x] statistics_service.go削減 (2421→2107行, -314行)
- [x] vector_store_aggregation.go作成 (128行)
- [x] vector_store_service.go削減 (1860→1739行, -121行)
- [x] 全テストPASS ✅

### フェーズ2: 保留
- [ ] statistics_core.go作成
- [ ] statistics_correlation.go作成
- [ ] statistics_anomaly.go作成
- [ ] statistics_forecast.go作成
- [ ] statistics_weekly.go作成
- [ ] ai_handler.go分割 (1341行)
- [ ] weather_service.go分割 (1319行)

### 実績サマリー
| ファイル | 変更前 | 変更後 | 削減 | 分離先 |
|---------|--------|--------|------|--------|
| statistics_service.go | 2421行 | 2107行 | -314行 | statistics_math.go (318行) |
| vector_store_service.go | 1860行 | 1739行 | -121行 | vector_store_aggregation.go (128行) |
| **合計削減** | **4281行** | **3846行** | **-435行** | **+446行** |

### 残りの1000行超ファイル
1. statistics_service.go - 2107行 (改善済み)
2. vector_store_service.go - 1739行 (改善済み)
3. ai_handler.go - 1341行
4. weather_service.go - 1319行
