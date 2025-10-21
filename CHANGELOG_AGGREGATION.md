# データ集約機能の実装 - 変更履歴

## 📅 2025-10-21 (Update 3)

### 🎨 改善: 異常検知の表示改善（製品名・日付フォーマット）

**概要:**  
異常検知時の質問で、製品IDの代わりに製品名を表示し、日付も読みやすい形式にフォーマットする機能を追加しました。

**変更内容:**

#### バックエンド（Go）

**pkg/services/statistics_service.go:**

1. **新規関数: formatDateForDisplay()**
   - 日付を自動判別して日本語形式に変換
   - `2022-04` → `2022年4月`
   - `2022-W10` → `2022年 第10週`
   - `2022-03-07` → `2022年3月7日`

2. **GenerateAIQuestion() の改善:**
   - 製品名がある場合は優先的に表示（なければProductID）
   - フォーマット済みの日付を使用
   - AI生成質問にもフォーマット適用

**Before（改善前）:**
```
続いて、2022-04 の P001 について教えてください。
実績値: 3054.00 予測値: 2337.33
```

**After（改善後）:**
```
続いて、2022年4月 の「製品A」について教えてください。
実績値: 3054.00 予測値: 2337.33
```

**影響範囲:**
- ✅ チャット画面での異常検知質問
- ✅ 全粒度（日次・週次・月次）に対応
- ✅ 後方互換性あり（製品名がない場合はIDを表示）

**ドキュメント:**
- [ANOMALY_DISPLAY_IMPROVEMENT.md](./ANOMALY_DISPLAY_IMPROVEMENT.md) 🆕

---

## 📅 2025-10-21 (Update 2)

### 🔧 改善: 異常検知の週次集約対応

**概要:**  
異常検知を実行する前に、日次データを週次/月次に集約してから分析を行うように改善しました。これにより、toB企業向けの週次/月次分析で誤検知が減少します。

**変更内容:**

**pkg/services/statistics_service.go:**

1. **DetectAnomaliesWithGranularity()** - 粒度指定可能な異常検知
2. **aggregateDataForAnomalyDetection()** - データ集約関数
3. 粒度別パラメータ調整:
   - 日次: 30日移動平均、閾値50%
   - 週次: 4週移動平均、閾値40%
   - 月次: 3ヶ月移動平均、閾値30%

**ドキュメント:**
- [ANOMALY_DETECTION_WEEKLY_AGGREGATION.md](./ANOMALY_DETECTION_WEEKLY_AGGREGATION.md)

---

## 📅 2025-10-21 (Update 1)

### ✨ 新機能: データ集約粒度の選択

**概要:**  
日次・週次・月次の3つの粒度でデータを集約して分析できる機能を追加しました。

**✅ 実装済み:**
1. **週次分析ページ** - 統計分析での粒度選択
2. **ファイル分析ページ** - ファイルアップロード時の粒度選択 🆕

---

## 🔧 変更内容（Ver 2 - ファイル分析対応）

### バックエンド

#### 1. **pkg/handlers/ai_handler.go** 🆕

**AnalyzeFile() メソッドの改良:**

```go
func (ah *AIHandler) AnalyzeFile(c *gin.Context) {
    // データ粒度を取得（デフォルト: weekly）
    granularity := c.PostForm("granularity")
    if granularity == "" {
        granularity = "weekly"
    }

    // 粒度のバリデーション
    if granularity != "daily" && granularity != "weekly" && granularity != "monthly" {
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false,
            "error":   "無効な粒度です...",
        })
        return
    }
    
    // 粒度に応じた集約処理
    switch granularity {
    case "daily":
        periodKey = t.Format("2006-01-02")
    case "weekly":
        year, week := t.ISOWeek()
        periodKey = fmt.Sprintf("%d-W%02d", year, week)
    case "monthly":
        periodKey = t.Format("2006-01")
    }
    
    // ...
}
```

**変更点:**
- ✅ 粒度パラメータの取得とバリデーション
- ✅ 月次専用から汎用集約ロジックに変更
- ✅ 期間キーを粒度に応じて生成
- ✅ サマリーに粒度情報を含める

---

### フロントエンド

#### 1. **src/app/analysis/page.tsx** 🆕

**新規State:**
```typescript
const [granularity, setGranularity] = useState<'daily' | 'weekly' | 'monthly'>('weekly');
const [pendingGranularity, setPendingGranularity] = useState<'daily' | 'weekly' | 'monthly' | null>(null);
const [isGranularityChangeDialogOpen, setGranularityChangeDialogOpen] = useState(false);
```

**粒度変更ハンドラー:**
```typescript
const handleGranularityChange = (newGranularity: 'daily' | 'weekly' | 'monthly') => {
    // 既に分析済みの場合はアラートを表示
    if (selectedReport || analysisSummary) {
        setPendingGranularity(newGranularity);
        setGranularityChangeDialogOpen(true);
    } else {
        setGranularity(newGranularity);
    }
};
```

**UI の追加:**
```tsx
{/* 粒度選択 */}
<select
    id="granularity"
    value={granularity}
    onChange={(e) => handleGranularityChange(e.target.value as 'daily' | 'weekly' | 'monthly')}
    disabled={isLoading}
>
    <option value="daily">📅 日次（詳細分析・短期トレンド）</option>
    <option value="weekly">📆 週次（推奨・中期トレンド）</option>
    <option value="monthly">📊 月次（長期トレンド・高速処理）</option>
</select>

{/* ヘルプテキスト */}
<p className="text-xs text-gray-500">
    {granularity === 'daily' && '⚡ 処理時間: やや遅い | 📊 詳細度: 高 | 💡 用途: 短期分析（1週間〜1ヶ月）'}
    {granularity === 'weekly' && '⚡ 処理時間: 普通 | 📊 詳細度: 中 | 💡 用途: 中期分析（1ヶ月〜6ヶ月）⭐'}
    {granularity === 'monthly' && '⚡ 処理時間: 高速 | 📊 詳細度: 低 | 💡 用途: 長期分析（6ヶ月以上）'}
</p>
```

**粒度変更確認ダイアログ:**
```tsx
<AlertDialog open={isGranularityChangeDialogOpen}>
    <AlertDialogContent>
        <AlertDialogHeader>
            <AlertDialogTitle>⚠️ データ粒度を変更しますか？</AlertDialogTitle>
            <AlertDialogDescription>
                粒度を変更すると、現在の分析結果がクリアされます。
            </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
            <AlertDialogCancel>キャンセル</AlertDialogCancel>
            <AlertDialogAction>変更する</AlertDialogAction>
        </AlertDialogFooter>
    </AlertDialogContent>
</AlertDialog>
```

**API リクエストの変更:**
```typescript
const formData = new FormData();
formData.append('file', selectedFile);
formData.append('granularity', granularity); // 🆕
```

---

## 🎯 使用例

### ファイル分析での利用

**日次分析（新製品の初動分析）**

**変更:**
```go
// WeeklyAnalysisRequest に granularity フィールドを追加
type WeeklyAnalysisRequest struct {
    ProductID   string           `json:"product_id" binding:"required"`
    StartDate   string           `json:"start_date" binding:"required"`
    EndDate     string           `json:"end_date" binding:"required"`
    SalesData   []SalesDataPoint `json:"sales_data"`
    Granularity string           `json:"granularity"` // "daily", "weekly", "monthly"
}

// WeeklyAnalysisResponse に granularity フィールドを追加
type WeeklyAnalysisResponse struct {
    // ...既存フィールド
    Granularity string `json:"granularity"`
}
```

#### 2. **pkg/handlers/ai_handler.go**

**変更:**
- リクエストから `granularity` を取得
- デフォルト値を "weekly" に設定
- バリデーション追加（"daily", "weekly", "monthly" のみ許可）

```go
// デフォルトの粒度は週次
granularity := req.Granularity
if granularity == "" {
    granularity = "weekly"
}

// 粒度のバリデーション
if granularity != "daily" && granularity != "weekly" && granularity != "monthly" {
    c.JSON(http.StatusBadRequest, gin.H{
        "success": false,
        "error":   "granularityは 'daily', 'weekly', 'monthly' のいずれかを指定してください",
    })
    return
}
```

#### 3. **pkg/services/statistics_service.go**

**新規メソッド:**

##### `groupByDay(data []models.SalesDataPoint) []models.WeeklySummary`
- 日次データをそのまま返す（集約なし）
- 前日比を計算

##### `groupByMonth(data []models.SalesDataPoint, startDate time.Time) []models.WeeklySummary`
- カレンダー月単位でデータを集約
- 月内の統計（合計、平均、標準偏差）を計算
- 前月比を算出

**変更:**
```go
func (s *StatisticsService) AnalyzeWeeklySales(
    productID, productName string, 
    salesData []models.SalesDataPoint, 
    startDate, endDate time.Time,
    granularity string  // 🆕 粒度パラメータを追加
) (*models.WeeklyAnalysisResponse, error) {
    
    // 粒度に応じて処理を分岐
    switch granularity {
    case "daily":
        weeklySummaries = s.groupByDay(salesData)
    case "monthly":
        weeklySummaries = s.groupByMonth(salesData, startDate)
    default: // "weekly"
        // 既存の週次処理
    }
    
    // レスポンスに粒度を含める
    return &models.WeeklyAnalysisResponse{
        // ...
        Granularity: granularity,
    }, nil
}
```

---

### フロントエンド

#### 1. **src/app/weekly-analysis/page.tsx**

**新規State:**
```typescript
const [granularity, setGranularity] = useState<'daily' | 'weekly' | 'monthly'>('weekly');
```

**API リクエストの変更:**
```typescript
const response = await fetch('/api/v1/ai/analyze-weekly', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
        product_id: productId,
        start_date: startDate,
        end_date: endDate,
        granularity: granularity,  // 🆕
    }),
});
```

**UI の追加:**
```tsx
<div>
    <Label htmlFor="granularity">集約粒度</Label>
    <select
        id="granularity"
        value={granularity}
        onChange={(e) => setGranularity(e.target.value as 'daily' | 'weekly' | 'monthly')}
        className="w-full p-2 border rounded-lg"
    >
        <option value="daily">📅 日次</option>
        <option value="weekly">📆 週次</option>
        <option value="monthly">📊 月次</option>
    </select>
</div>
```

**動的ラベル:**
```tsx
{/* ページタイトル */}
<h1>
    📊 {granularity === 'daily' ? '日次' : granularity === 'monthly' ? '月次' : '週次'}売上分析
</h1>

{/* 表のヘッダー */}
<th>前{granularity === 'daily' ? '日' : granularity === 'monthly' ? '月' : '週'}比</th>

{/* カード表示 */}
<div>{analysis.total_weeks}{granularity === 'daily' ? '日間' : granularity === 'monthly' ? 'ヶ月' : '週間'}</div>
```

---

### ドキュメント

#### 1. **DATA_AGGREGATION_GUIDE.md** 🆕

新規作成：データ集約機能の完全ガイド

**内容:**
- 3種類の粒度の説明
- 使い方（Web UI / API）
- 粒度別の使い分け
- 実践例（新製品分析、四半期レビュー、年間計画）
- 技術詳細
- パフォーマンス比較
- よくある質問

#### 2. **README.md**

ドキュメントリストに追加:
```markdown
| [DATA_AGGREGATION_GUIDE.md](./DATA_AGGREGATION_GUIDE.md) | データ集約分析ガイド（日次・週次・月次） ⭐ |
```

---

## 🎯 使用例

### 日次分析（新製品の初動分析）

```bash
curl -X POST http://localhost:8080/api/v1/ai/analyze-weekly \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "P_NEW_001",
    "start_date": "2024-01-01",
    "end_date": "2024-01-31",
    "granularity": "daily"
  }'
```

### 週次分析（デフォルト・四半期レビュー）

```bash
curl -X POST http://localhost:8080/api/v1/ai/analyze-weekly \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "P001",
    "start_date": "2024-01-01",
    "end_date": "2024-03-31",
    "granularity": "weekly"
  }'
```

### 月次分析（年間計画策定）

```bash
curl -X POST http://localhost:8080/api/v1/ai/analyze-weekly \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "P001",
    "start_date": "2023-01-01",
    "end_date": "2023-12-31",
    "granularity": "monthly"
  }'
```

---

## 📊 レスポンス例

```json
{
  "success": true,
  "data": {
    "product_id": "P001",
    "product_name": "製品A",
    "granularity": "monthly",
    "total_weeks": 12,
    "weekly_summary": [
      {
        "week_number": 1,
        "week_start": "2023-01-01",
        "week_end": "2023-01-31",
        "total_sales": 2340,
        "average_sales": 75.5,
        "min_sales": 45,
        "max_sales": 120,
        "business_days": 31,
        "week_over_week": 0,
        "std_dev": 18.3,
        "avg_temperature": 5.2
      }
      // ... 残り11ヶ月
    ],
    "overall_stats": {
      "average_weekly_sales": 2450,
      "median_weekly_sales": 2380,
      "std_dev_weekly_sales": 320,
      "best_week": 12,
      "worst_week": 2,
      "growth_rate": 22.5,
      "volatility": 0.13
    },
    "trends": {
      "direction": "上昇",
      "strength": 0.82,
      "seasonality": "後半期に需要増加傾向",
      "peak_week": 12,
      "low_week": 2,
      "average_growth": 1.9
    },
    "recommendations": [
      "12月の好調要因（年末需要）を次年度に活用してください",
      "2月の低迷は季節要因と考えられます"
    ]
  }
}
```

---

## ✅ テスト結果

### ビルド
- ✅ Go バックエンド: 正常にビルド
- ✅ Next.js フロントエンド: 正常にビルド（軽微な警告のみ）

### 互換性
- ✅ 既存のAPI（粒度未指定）: デフォルトで "weekly" として動作
- ✅ 既存のフロントエンド: 問題なく動作

---

## 🚀 今後の拡張案

### 1. **カスタム粒度**
```json
{
  "granularity": "custom",
  "interval_days": 14  // 2週間ごと
}
```

### 2. **集約関数の選択**
```json
{
  "aggregation_function": "sum"  // "sum", "average", "median"
}
```

### 3. **移動平均**
```json
{
  "moving_average": true,
  "window_size": 7  // 7日移動平均
}
```

### 4. **比較分析**
```json
{
  "compare_with": "previous_period"  // 前期間比較
}
```

---

## 📝 まとめ

✅ **3つの粒度**（日次・週次・月次）で柔軟な分析が可能  
✅ **既存コードとの互換性**を完全に維持  
✅ **動的UI**で粒度に応じた表示を実現  
✅ **包括的なドキュメント**を作成  
✅ **拡張性**を考慮した設計

---

**実装者:** GitHub Copilot  
**実装日:** 2025-10-21  
**影響範囲:** バックエンド（Go）、フロントエンド（Next.js）、ドキュメント
