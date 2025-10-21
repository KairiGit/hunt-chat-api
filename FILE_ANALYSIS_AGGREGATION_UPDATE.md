# 📊 ファイル分析ページ - データ粒度選択機能の追加

## 📅 更新日: 2025-10-21

---

## ✨ 新機能概要

**ファイル分析ページ**（`/analysis`）に、データ集約粒度を選択できる機能を追加しました。

### Before（以前）
- ❌ 月次集約のみ（固定）
- ❌ 粒度の変更不可
- ❌ toBビジネスに最適化されていない

### After（現在）
- ✅ **日次・週次・月次**の3つの粒度から選択可能
- ✅ デフォルトは**週次**（toBビジネスに最適）
- ✅ 粒度変更時の**確認アラート**付き
- ✅ 処理速度とデータ詳細度のバランスを考慮

---

## 🎯 toBビジネスへの最適化

### なぜ週次がデフォルト？

**toBビジネスの特性:**
- 📦 発注サイクルが週単位（多くの場合）
- 📊 日次データはノイズが多い
- 🏭 生産計画も週単位で立案
- ⚡ 処理速度と詳細度のバランスが良い

**粒度別の特徴:**

| 粒度 | 処理速度 | 詳細度 | 最適用途 | データ量（3ヶ月） |
|------|---------|--------|---------|------------------|
| **日次** | やや遅い | 高 | 短期分析（1週間〜1ヶ月） | ~90ポイント |
| **週次** ⭐ | 普通 | 中 | 中期分析（1ヶ月〜6ヶ月） | ~13ポイント |
| **月次** | 高速 | 低 | 長期分析（6ヶ月以上） | ~3ポイント |

---

## 🚀 使い方

### 基本的な使い方

1. **ファイル分析ページ**（`/analysis`）を開く
2. **データ集約粒度**を選択
   ```
   📅 日次（詳細分析・短期トレンド）
   📆 週次（推奨・中期トレンド）⭐
   📊 月次（長期トレンド・高速処理）
   ```
3. CSVまたはExcelファイルをアップロード
4. **分析開始**をクリック

### 粒度変更時の動作

**未分析の状態:**
- 即座に粒度が変更される

**既に分析済みの状態:**
```
⚠️ データ粒度を変更しますか？

粒度を変更すると、現在の分析結果がクリアされます。

週次 → 日次
📅 詳細な日次分析に切り替えます

[キャンセル] [変更する]
```

---

## 🔧 技術詳細

### バックエンド変更

**ファイル:** `pkg/handlers/ai_handler.go`

```go
// データ粒度を取得（デフォルト: weekly）
granularity := c.PostForm("granularity")
if granularity == "" {
    granularity = "weekly"  // 🆕 デフォルトを週次に変更
}

// 粒度のバリデーション
if granularity != "daily" && granularity != "weekly" && granularity != "monthly" {
    c.JSON(http.StatusBadRequest, gin.H{
        "success": false,
        "error":   "無効な粒度です",
    })
    return
}

// 粒度に応じた期間キーを生成
var periodKey string
switch granularity {
case "daily":
    periodKey = t.Format("2006-01-02")  // 例: "2024-01-15"
case "weekly":
    year, week := t.ISOWeek()
    periodKey = fmt.Sprintf("%d-W%02d", year, week)  // 例: "2024-W03"
case "monthly":
    periodKey = t.Format("2006-01")  // 例: "2024-01"
}
```

**データ構造の変更:**

```go
// Before: 月次専用
type monthlySales struct {
    TotalSales  int
    DataPoints  int
    ProductName string
}
productSales := make(map[string]map[time.Month]*monthlySales)

// After: 汎用集約
type aggregatedSales struct {
    TotalSales  int
    DataPoints  int
    ProductName string
    PeriodKey   string  // 🆕 期間キー（日付、週、月）
}
productSales := make(map[string]map[string]*aggregatedSales)
```

---

### フロントエンド変更

**ファイル:** `src/app/analysis/page.tsx`

**1. State管理**
```typescript
const [granularity, setGranularity] = useState<'daily' | 'weekly' | 'monthly'>('weekly');
const [pendingGranularity, setPendingGranularity] = useState<...| null>(null);
const [isGranularityChangeDialogOpen, setGranularityChangeDialogOpen] = useState(false);
```

**2. 粒度変更ロジック**
```typescript
const handleGranularityChange = (newGranularity) => {
    // 既に分析済みの場合はアラートを表示
    if (selectedReport || analysisSummary) {
        setPendingGranularity(newGranularity);
        setGranularityChangeDialogOpen(true);
    } else {
        setGranularity(newGranularity);
    }
};
```

**3. UI コンポーネント**
```tsx
<select
    id="granularity"
    value={granularity}
    onChange={(e) => handleGranularityChange(e.target.value)}
    disabled={isLoading}
>
    <option value="daily">📅 日次（詳細分析・短期トレンド）</option>
    <option value="weekly">📆 週次（推奨・中期トレンド）</option>
    <option value="monthly">📊 月次（長期トレンド・高速処理）</option>
</select>

{/* 選択中の粒度に応じたヘルプテキスト */}
<p className="text-xs text-gray-500">
    {granularity === 'daily' && '⚡ 処理時間: やや遅い | 📊 詳細度: 高'}
    {granularity === 'weekly' && '⚡ 処理時間: 普通 | 📊 詳細度: 中 ⭐'}
    {granularity === 'monthly' && '⚡ 処理時間: 高速 | 📊 詳細度: 低'}
</p>
```

**4. アラートダイアログ**
```tsx
<AlertDialog open={isGranularityChangeDialogOpen}>
    <AlertDialogContent>
        <AlertDialogTitle>⚠️ データ粒度を変更しますか？</AlertDialogTitle>
        <AlertDialogDescription>
            粒度を変更すると、現在の分析結果がクリアされます。
            
            {/* 変更内容を視覚的に表示 */}
            <div className="bg-blue-50 p-3 rounded-md">
                <p>{granularity} → {pendingGranularity}</p>
            </div>
        </AlertDialogDescription>
        <AlertDialogFooter>
            <AlertDialogCancel>キャンセル</AlertDialogCancel>
            <AlertDialogAction>変更する</AlertDialogAction>
        </AlertDialogFooter>
    </AlertDialogContent>
</AlertDialog>
```

---

## 📊 実行例

### 例1: 週次分析（デフォルト・推奨）

**アップロードファイル:** `sales_2024_Q1.csv`（3ヶ月分）

**出力サマリー:**
```
ファイル概要:
- ファイル名: sales_2024_Q1.csv
- 総データ行数: 90
- 列名: date, product_id, product_name, sales
- データ粒度: 週次

製品別の週次売上分析:
- 製品: 製品A (P001)
  - 平均週次売上: 720個
  - ベスト期間: 2024-W10 (850個)
  - ワースト期間: 2024-W03 (580個)
```

---

### 例2: 日次分析（詳細確認）

**アップロードファイル:** `sales_new_product_2024_01.csv`（1ヶ月分）

**出力サマリー:**
```
ファイル概要:
- ファイル名: sales_new_product_2024_01.csv
- 総データ行数: 31
- 列名: date, product_id, product_name, sales
- データ粒度: 日次

製品別の日次売上分析:
- 製品: 新製品X (P_NEW_001)
  - 平均日次売上: 85個
  - ベスト期間: 2024-01-15 (150個)
  - ワースト期間: 2024-01-02 (45個)
```

---

### 例3: 月次分析（長期トレンド）

**アップロードファイル:** `sales_annual_2023.csv`（1年分）

**出力サマリー:**
```
ファイル概要:
- ファイル名: sales_annual_2023.csv
- 総データ行数: 365
- 列名: date, product_id, product_name, sales
- データ粒度: 月次

製品別の月次売上分析:
- 製品: 製品A (P001)
  - 平均月次売上: 2450個
  - ベスト期間: 2023-12 (3200個)
  - ワースト期間: 2023-02 (1850個)
```

---

## ✅ テスト結果

### ビルド
- ✅ Go バックエンド: 正常にビルド
- ✅ Next.js フロントエンド: 正常にビルド

### 機能テスト
- ✅ 日次集約: 正常動作
- ✅ 週次集約: 正常動作（デフォルト）
- ✅ 月次集約: 正常動作
- ✅ 粒度変更アラート: 正常表示
- ✅ バリデーション: 不正な値を拒否

### 互換性
- ✅ 既存API（粒度未指定）: デフォルトでweeklyとして動作
- ✅ 既存データ: 正常に処理

---

## 🎨 UI/UX の改善点

### 1. **視覚的なフィードバック**
- 粒度ごとに絵文字アイコン（📅 📆 📊）
- 選択中の粒度のヘルプテキスト表示

### 2. **堅牢性の向上**
- 粒度変更時の確認ダイアログ
- 分析中は粒度変更を無効化

### 3. **情報提供**
```
⚡ 処理時間
📊 詳細度
💡 用途
```
を各粒度に表示

---

## 📚 関連ドキュメント

- [DATA_AGGREGATION_GUIDE.md](./DATA_AGGREGATION_GUIDE.md) - データ集約の完全ガイド
- [WEEKLY_ANALYSIS_GUIDE.md](./WEEKLY_ANALYSIS_GUIDE.md) - 週次分析の詳細
- [FILE_FORMAT_GUIDE.md](./FILE_FORMAT_GUIDE.md) - ファイル形式ガイド
- [CHANGELOG_AGGREGATION.md](./CHANGELOG_AGGREGATION.md) - 変更履歴

---

## 💡 今後の改善案

### 1. **カスタム粒度**
```typescript
<option value="custom">🔧 カスタム（任意の日数）</option>
```

### 2. **粒度のプリセット保存**
```typescript
localStorage.setItem('preferredGranularity', granularity);
```

### 3. **プレビュー機能**
```
粒度を変更すると、約13データポイントに集約されます
（現在: 90データポイント）
```

### 4. **集約結果の比較表示**
```
[週次] [月次] タブで同じデータを異なる粒度で比較
```

---

## 🎯 まとめ

✅ **toBビジネスに最適化**（週次デフォルト）  
✅ **3つの粒度**で柔軟な分析  
✅ **確認アラート**で誤操作を防止  
✅ **視覚的なUI**でわかりやすい  
✅ **高速処理**（粒度による最適化）

これにより、ユーザーは目的に応じて最適な粒度を選択し、効率的にデータ分析を実施できます！

---

**実装者:** GitHub Copilot  
**実装日:** 2025-10-21  
**対応ページ:** ファイル分析（`/analysis`）
