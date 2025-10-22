# 気象データ統合ガイド

## 概要

現在、気象データはCSVファイル（`data/weather/tokyo_weather_2021-2023.csv`）として存在し、`WeatherService`で取得可能ですが、Qdrantには格納されておらず、RAG（検索拡張生成）や需要予測で十分に活用できていない状況です。

このドキュメントでは、気象データを効果的に統合する3つのアプローチと、経済データ（yfinance相当）の統合戦略を提案します。

---

## 現状分析（簡潔）

- 実装済み
  - `pkg/services/weather_service.go`: 気象庁API統合、過去データ（モック生成含む）、各種サマリー/トレンド
  - `data/weather/tokyo_weather_2021-2023.csv`: 過去の気象データ（CSV）
- 課題
  1) Qdrantに気象データ未保存 → RAGで利用不可
  2) 需要予測に気象特徴が未統合 → 精度向上の余地大
  3) 異常検知時の原因分析に気象参照が未連携

---

## 提案1: 需要予測の特徴量として活用（精度向上）🎯

最も直接的で効果的。製品により、気温/湿度/降水/天気コードは需要に強い影響を与えます。

### 実装ポイント

- 追加インターフェイス（案）: `pkg/services/demand_forecast_service.go`

  ```go
  type EnhancedForecastInput struct {
      ProductID   string
      RegionCode  string        // 例: "130000"
      History     []SalesPoint  // 日次の売上系列
      Weather     []WeatherPoint // 同日付の気象系列
      Economic    []EconomicPoint // 任意: 経済系列
  }

  type SalesPoint struct {
      Date  string  // YYYY-MM-DD
      Qty   float64
  }

  type WeatherPoint struct {
      Date          string
      Temperature   float64
      Humidity      float64
      Precipitation float64
      WindSpeed     float64
      WeatherCode   string
  }

  type EconomicPoint struct {
      Date   string
      Nikkei float64
      UsdJpy float64
      OilWTI float64
  }

  // 擬似コード
  func (s *DemandForecastService) ForecastWithWeather(ctx context.Context, in EnhancedForecastInput) (DailyForecast, error) {
      joined := joinByDate(in.History, in.Weather, in.Economic)
      features := buildFeatures(joined) // 移動平均, ラグ, 曜日/祝日, 気温ビン, 降雨ダミー, 変化率 等
      return s.model.Predict(features)
  }
  ```

- 既存フック
  - `weather_service.GetHistoricalWeatherDataByRange(regionCode, days)` で気象系列取得
  - `pkg/handlers/demand_forecast_handler.go` で `regionCode` をリクエストから受け取り、`ForecastWithWeather` に配線
- 欠損処理/スケーリング
  - 欠損は前後日補間、降雨は0優先、極端値はウィンザー化
  - 気温は {<10, 10–20, 20–30, >30} のビンでダミー化し非線形性を吸収
- 効果測定
  - MAPE/RMSEの改善、季節変動商品の外れ予測率の減少
  - 気象あり/なしのA/B比較を最低2週間

---

## 提案2: Qdrantに日次サマリーを保存しRAGで活用 🧠

数値列をそのままベクトル化せず、日次の短い要約文にし、`date`/`region_code` でフィルタ可能に保存。

### コレクション設計

- コレクション: `weather_daily_summaries`
- ベクトル本文（例）: 「2023-07-24 東京: 平均27.8℃、湿度70%、降水1.2mm、南風3.2m/s、晴れ時々曇り。」
- ペイロード
  - `type`: "weather_daily"
  - `date`: "YYYY-MM-DD"
  - `region_code`: "130000"
  - `avg_temp`, `humidity`, `precipitation`, `wind_speed`, `weather_code`

### 実装ポイント

- `VectorStoreService.StoreDocument(ctx, collection, id, text, metadata)` を再利用
- `weather_service` の `DailyWeatherSummary` から本文テキスト生成ユーティリティを追加
- FieldIndex: `date`, `region_code` を Keyword で作成

### 利用例（RAG）

- ユーザー: 「この週の売上下振れ時、関東の天気は？」
  1) `date` 範囲 + `region_code` IN 関東コード群でフィルタ検索
  2) 取得文をコンテキストにして回答（異常の説明を補強）

---

## 提案3: 異常検知時の原因分析へ自動参照（質問の質向上）🔍

売上異常（上振れ/下振れ）で同日の気象を自動参照し、AIの質問/仮説に織り込む。

### 実装ポイント

- `pkg/services/azure_openai_service.go`
  - `GenerateEnhancedQuestion(anomaly, pastResponses, weatherData)` の `weatherData` に `WeatherPoint[]` を渡す
  - プロンプトへ「当日の気温/降雨/天気コード」要約を付与
- `pkg/handlers/ai_handler.go`
  - 異常ごとに `weather_service.GetHistoricalWeatherData(region, date,date)` を呼び、返却の EnhancedQuestion に `ExternalContext.weather` を格納

### 期待効果

- 「その日は最高気温35℃の猛暑。冷却飲料の補充/販促は実施しましたか？」など、具体的で行動可能な問いへ自動シフト

---

## 経済・金融データ（yfinance相当）の統合案 📈

広域の需要変動や景気感度製品向けに、株価指数/為替/コモディティ/景況感指数を外生変数として取り込む。

### データ候補

- 株価指数: 日経平均、TOPIX、S&P500
- 為替: USD/JPY, EUR/JPY
- コモディティ: 原油(WTI/Brent)、小麦、銅
- 景況感: PMI、消費者信頼感指数（入手可なら）

### 実装構成

- 新規: `pkg/services/economic_service.go`
  - `GetMarketSeries(symbol, start, end)` … yfinance相当の取得（本番は公式API/有償API or CSVインポート）
  - 休日補間・正規化・日次リサンプリング
- 需要予測
  - `EnhancedForecastInput` に `EconomicPoint[]` を拡張
  - ラグ（t-1, t-7, t-30）と変化率（pct_change）を特徴化

### Qdrant保存（任意）

- コレクション: `economic_daily_summaries`
- 本文例: 「2024-10-01: 日経+1.2%、USD/JPY 148.3、WTI 82.1」
- フィルタ: `date`

---

## 導入フェーズと運用

- フェーズ導入
  1) オフライン実験（30日）: 気象特徴の精度インパクト測定
  2) シャドーモード（2週間）: 本番予測と並走比較
  3) 段階ロールアウト: メトリクス監視の上で切替
- 運用監視
  - 欠損割合、外れ値率、MAPEを週次レポート
  - API障害時はモック/過去統計にフォールバック

---

## 変更する主なファイル（TODO）

- `pkg/services/demand_forecast_service.go`: ForecastWithWeatherの追加、特徴量生成
- `pkg/handlers/demand_forecast_handler.go`: regionCodeの受け渡し
- `pkg/services/weather_service.go`: 日次サマリー→本文テキスト生成ユーティリティ
- `pkg/services/vector_store_service.go`: weather_daily_summaries保存ヘルパ
- `pkg/services/economic_service.go`: 経済データ取得・正規化（新規）
- `scripts/init_system_docs.go`: 本ドキュメントをQdrantへ登録（任意）

---

## クイック次アクション（推奨）

1) 需要予測リクエストで `regionCode` を受け付ける（handler + service）
2) `weather_service` から期間の `DailyWeatherSummary[]` を取得し、日付ジョイン
3) 最小構成の `ForecastWithWeather` を追加（気温/降雨のみで効果測定）
4) 結果を `PROGRESS_REPORT.md` に追記し、MAPE比較を可視化

---

## 付録: 疑似yfinance（CSV）EconomicServiceの使い方

- ファイル: `pkg/services/economic_service.go`
- 目的: CSVから日次時系列（例: 日経平均、USD/JPY、WTI 等）を読み込み、`GetMarketSeries` で取得
- シンボルとCSVの対応は `NewEconomicService(baseDir, map[string]string{"NIKKEI": "data/econ/nikkei.csv"})` のように登録
- CSVの想定ヘッダ（いずれか）: `Date`, `Adj Close|Close|Price|Value|終値`

簡易サンプル:

```go
svc := services.NewEconomicService("", map[string]string{
    "NIKKEI": "data/econ/nikkei.csv",
    "USDJPY": "data/econ/usdjpy.csv",
})
start, _ := time.Parse("2006-01-02", "2024-01-01")
end, _   := time.Parse("2006-01-02", "2024-03-31")
series, err := svc.GetMarketSeries("NIKKEI", start, end)
if err != nil { /* handle */ }
returns, _ := svc.GetPctChange("NIKKEI", start, end)
```

これを `EnhancedForecastInput.Economic` に変換し、需要予測の外生変数として組み込みます。

