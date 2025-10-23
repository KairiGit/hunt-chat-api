# 経済データとの相関分析実装ガイド

## 概要

ファイル分析時に、天気データとの相関分析に加えて、**経済データとの相関分析（遅れ相関を含む）** を実装しました。これにより、売上データと経済指標（日経平均、為替レート、原油価格など）の関係性を分析できます。

## 実装内容

### 1. StatisticsServiceの拡張

#### 1.1 EconomicServiceの統合

**変更ファイル**: `pkg/services/statistics_service.go`

```go
type StatisticsService struct {
    weatherService     *WeatherService
    economicService    *EconomicService  // ← 追加
    azureOpenAIService *AzureOpenAIService
}
```

#### 1.2 新しいメソッド: `AnalyzeSalesEconomicCorrelation`

経済データとの相関を分析する新しいメソッドを追加しました。

**主な機能**:
- 複数の経済指標（NIKKEI、USDJPY、WTI等）との相関を計算
- **遅れ相関の検出**: 最大30日のラグを考慮した相関分析
- 統計的有意性の判定（p値 < 0.05）
- 相関係数の絶対値が0.3以上のものを抽出
- 上位10件の有意な相関を返す

**使用例**:
```go
economicCorrelations, err := s.AnalyzeSalesEconomicCorrelation(
    salesData, 
    []string{"NIKKEI", "USDJPY", "WTI"}, 
    30  // 最大30日の遅れ相関
)
```

**遅れ相関の意味**:
- `lag=0`: 同じ日の相関
- `lag=+7`: 経済指標が7日後の売上に影響
- `lag=-7`: 経済指標が7日前の売上に影響（先行指標）

### 2. CreateAnalysisReportの更新

**変更内容**:
```go
// 天気データとの相関
weatherCorrelations, err := s.AnalyzeSalesWeatherCorrelation(salesData, regionCode)

// 経済データとの相関（遅れ相関を含む）
economicCorrelations, err := s.AnalyzeSalesEconomicCorrelation(
    salesData, 
    []string{"NIKKEI", "USDJPY", "WTI"}, 
    30
)

// 両方を結合
correlations := append(weatherCorrelations, economicCorrelations...)
```

### 3. レコメンデーション生成の拡張

**変更ファイル**: `pkg/services/statistics_service.go` の `generateRecommendations`

経済データの相関に基づいて、以下のような推奨事項を生成します:

- **日経平均との相関**:
  - 正の相関: 「株価動向を需要予測に活用できる可能性があります」
  - 負の相関: 「景気後退期に需要が増加する製品特性が示唆されます」

- **為替レート（USD/JPY）との相関**:
  - 「輸入原材料や外国人観光客需要の影響を考慮してください」

- **原油価格（WTI）との相関**:
  - 「輸送コストや消費者心理への影響を監視してください」

- **遅れ相関の検出**:
  - 「⏱️ タイムラグが検出されました。先行指標として活用できます」

### 4. AIHandlerの更新

**変更ファイル**: `pkg/handlers/ai_handler.go`

```go
type AIHandler struct {
    azureOpenAIService    *services.AzureOpenAIService
    weatherService        *services.WeatherService
    economicService       *services.EconomicService  // ← 追加
    demandForecastService *services.DemandForecastService
    vectorStoreService    *services.VectorStoreService
    statisticsService     *services.StatisticsService
}
```

**コンストラクタの更新**:
```go
func NewAIHandler(
    azureOpenAIService *services.AzureOpenAIService, 
    weatherService *services.WeatherService, 
    economicService *services.EconomicService,  // ← 追加
    demandForecastService *services.DemandForecastService, 
    vectorStoreService *services.VectorStoreService
) *AIHandler
```

### 5. main.goでの初期化

**変更ファイル**: `cmd/server/main.go`

```go
// 経済データサービスの初期化
economicSymbolMapping := map[string]string{
    "NIKKEI": "moc/nikkei_daily.csv",
    // 追加の経済指標はここに追加
    // "USDJPY": "data/econ/usdjpy.csv",
    // "WTI": "data/econ/wti.csv",
}
economicService := services.NewEconomicService("", economicSymbolMapping)

// AIHandlerの初期化（economicServiceを渡す）
aiHandler := handlers.NewAIHandler(
    azureOpenAIService, 
    weatherHandler.GetWeatherService(), 
    economicService,  // ← 追加
    demandForecastHandler.GetDemandForecastService(), 
    vectorStoreService
)
```

## フロントエンドの対応

### AnalysisReportView コンポーネントの更新

**変更ファイル**: `src/components/analysis/AnalysisReportView.tsx`

相関分析セクションを拡張し、天気データと経済データの両方を適切に表示できるようにしました。

#### 主な変更点

1. **タイトルの変更**
   - 変更前: `🌤️ 天気との相関分析`
   - 変更後: `📊 相関分析` - より汎用的なタイトル

2. **説明文の更新**
   - 「売上と外部要因（天気、経済指標）の相関関係を分析しました」

3. **動的なアイコン表示**
   ```typescript
   // 要因に応じて適切なアイコンを表示
   🌡️ 気温
   💧 湿度
   📈 日経平均
   💱 USD/JPY
   🛢️ 原油価格
   ```

4. **遅れ相関の視覚化**
   - タイムラグがある相関には「⏱️ タイムラグあり」のバッジを表示
   - 先行/遅行指標として活用可能であることを明示

5. **統計的有意性の表示**
   - p値 < 0.05 の場合、「✓ 有意」マークを表示

#### 表示例

```
📊 相関分析
売上と外部要因（天気、経済指標）の相関関係を分析しました

┌─────────────────────────────────────┐
│ 🌡️ 気温                    65.0%  │
│ 強い正の相関（統計的に有意）        │
│ P値: 0.001 ✓ 有意  サンプル数: 90  │
│ ████████████████████░░░░░░░░░░     │
└─────────────────────────────────────┘

┌─────────────────────────────────────┐
│ 📈 日経平均 - yがxに対して+5日遅れ │
│                             52.0%  │
│ 中程度の正の相関（統計的に有意）    │
│ ⏱️ タイムラグあり（先行/遅行指標） │
│ P値: 0.015 ✓ 有意  サンプル数: 85  │
│ ██████████████░░░░░░░░░░░░░░░░     │
└─────────────────────────────────────┘
```

### 表示される情報

各相関について、以下の情報が表示されます:

1. **アイコンと名前**: 要因の種類を視覚的に識別
2. **相関係数**: パーセンテージ表示（大きく強調）
3. **解釈**: 統計的な意味の説明
4. **タイムラグ情報**: 遅れ相関がある場合のみ表示
5. **P値**: 統計的有意性（0.05未満の場合「✓ 有意」マーク）
6. **サンプル数**: データ点の数
7. **視覚的プログレスバー**: 相関の強さを色分けして表示
   - 緑: 強い相関（|r| > 0.5）
   - 黄: 中程度（|r| > 0.3）
   - 灰: 弱い相関

## 使い方

### ファイルアップロード時の自動分析

ファイル分析API（`POST /api/v1/ai/analyze-file`）にファイルをアップロードすると、自動的に以下の分析が実行されます:

1. 天気データとの相関分析
2. **経済データとの相関分析（遅れ相関を含む）** ← 新機能
3. 異常検知
4. 推奨事項の生成

### レスポンス例

```json
{
  "success": true,
  "analysis_report": {
    "report_id": "abc-123",
    "file_name": "sales_data.csv",
    "correlations": [
      {
        "factor": "temperature",
        "correlation_coef": 0.65,
        "p_value": 0.001,
        "interpretation": "強い正の相関（統計的に有意）"
      },
      {
        "factor": "NIKKEI_lag=0",
        "correlation_coef": 0.48,
        "p_value": 0.023,
        "interpretation": "中程度の正の相関（統計的に有意）"
      },
      {
        "factor": "NIKKEI_yがxに対して+5日遅れ",
        "correlation_coef": 0.52,
        "p_value": 0.015,
        "interpretation": "中程度の正の相関（統計的に有意）"
      }
    ],
    "recommendations": [
      "気温が高いほど売上が増加する傾向があります。夏季の在庫を強化することを推奨します。",
      "日経平均との正の相関が検出されました（相関係数: 0.52）。株価動向を需要予測に活用できる可能性があります。",
      "⏱️ タイムラグが検出されました: NIKKEI_yがxに対して+5日遅れ（相関係数: 0.52）。先行指標として活用できます。"
    ]
  }
}
```

## 経済指標の追加方法

新しい経済指標を追加するには、`cmd/server/main.go`の初期化部分を更新します:

```go
economicSymbolMapping := map[string]string{
    "NIKKEI": "moc/nikkei_daily.csv",
    "USDJPY": "data/econ/usdjpy.csv",      // 為替レート
    "WTI": "data/econ/wti_crude_oil.csv",  // 原油価格
    "GOLD": "data/econ/gold_price.csv",    // 金価格
    "CPI": "data/econ/consumer_price_index.csv", // 消費者物価指数
}
```

### CSVファイル形式

経済データのCSVファイルは以下の形式である必要があります:

```csv
Date,Close
2021-01-01,28000.50
2021-01-02,28150.75
2021-01-03,27980.25
...
```

または

```csv
Date,Adj Close
2021-01-01,28000.50
2021-01-02,28150.75
...
```

**対応するカラム名**:
- `Date`, `date`, `年月日`, `日付`, `データ日付`
- `Close`, `Adj Close`, `Adj_Close`, `AdjClose`, `Price`, `Value`, `終値`

## 遅れ相関の解釈

### 正の遅れ（例: +5日）
- 経済指標が5日後の売上に影響する
- 「日経平均が上がると、5日後に売上が増加する傾向」

### 負の遅れ（例: -3日）
- 売上が3日前に経済指標に先行する
- 「売上が増えると、3日後に日経平均が上がる傾向（先行指標）」

### 活用方法
- **先行指標**: 負のラグで有意な相関がある場合、売上データを株価予測に使える
- **遅行指標**: 正のラグで有意な相関がある場合、経済指標を売上予測に使える

## テクニカルノート

### 統計的手法

1. **ピアソン相関係数**: 線形相関の強さを測定
2. **p値計算**: Student's t分布を使用した統計的有意性の検証
3. **遅れ相関**: -30日から+30日のラグを走査
4. **多重検定補正**: Benjamini-Hochberg法によるFDR補正（オプション）

### パフォーマンス最適化

- 相関計算は並列処理せず、順次実行（データ依存性のため）
- 統計的に有意な結果のみを抽出（p < 0.05 または |r| >= 0.3）
- 上位10件のみを返すことで、レスポンスサイズを削減

## まとめ

ファイル分析機能が以下の点で強化されました:

✅ **天気データとの相関分析** (既存)
✅ **経済データとの相関分析** (新規)
✅ **遅れ相関の検出** (新規) - 先行指標・遅行指標の特定
✅ **統計的有意性の検証** (強化)
✅ **具体的な推奨事項の生成** (強化)

これにより、より多角的な需要予測と在庫管理の意思決定が可能になります。
