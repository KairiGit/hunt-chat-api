# 異常検知の表示改善ガイド

## 概要
異常検知時の質問で、製品IDの代わりに製品名を表示し、日付も読みやすい形式にフォーマットする機能を追加しました。

## 改善内容

### 1. **製品名の表示** 🏷️
従来はProductIDのみ表示されていましたが、ProductNameがある場合はそちらを優先的に表示するようになりました。

#### Before（改善前）
```
続いて、2022-04 の P001 について教えてください。

実績値: 3054.00 予測値: 2337.33

2022年4月にP001の売上が急増した原因について、何か心当たりはありますか？
```

#### After（改善後）
```
続いて、2022年4月 の「製品A」について教えてください。

実績値: 3054.00 予測値: 2337.33

2022年4月に「製品A」の売上が急増した原因について、何か心当たりはありますか？
```

### 2. **日付のフォーマット** 📅

日付形式を自動判別し、読みやすい日本語形式に変換します。

| 元の形式 | 粒度 | 変換後 | 例 |
|---------|-----|--------|-----|
| `YYYY-MM-DD` | 日次 | `YYYY年M月D日` | `2022-03-07` → `2022年3月7日` |
| `YYYY-WWW` | 週次 | `YYYY年 第WW週` | `2022-W10` → `2022年 第10週` |
| `YYYY-MM` | 月次 | `YYYY年M月` | `2022-04` → `2022年4月` |

## 実装詳細

### バックエンド（Go）

#### 1. 日付フォーマット関数の追加
`pkg/services/statistics_service.go`

```go
// formatDateForDisplay 日付を読みやすい形式にフォーマット
func (s *StatisticsService) formatDateForDisplay(date string) string {
    // 月次形式: YYYY-MM
    if len(date) == 7 && date[4] == '-' {
        t, err := time.Parse("2006-01", date)
        if err == nil {
            return t.Format("2006年1月")
        }
    }
    
    // 週次形式: YYYY-WWW
    if len(date) >= 7 && strings.Contains(date, "-W") {
        parts := strings.Split(date, "-W")
        if len(parts) == 2 {
            return fmt.Sprintf("%s年 第%s週", parts[0], parts[1])
        }
    }
    
    // 日次形式: YYYY-MM-DD
    if len(date) == 10 {
        t, err := time.Parse("2006-01-02", date)
        if err == nil {
            return t.Format("2006年1月2日")
        }
    }
    
    return date // パースできない場合はそのまま返す
}
```

#### 2. GenerateAIQuestion関数の改善

```go
func (s *StatisticsService) GenerateAIQuestion(anomaly models.AnomalyDetection) (string, []string) {
    // 製品の表示名を決定（製品名があればそれを使用、なければID）
    displayName := anomaly.ProductName
    if displayName == "" {
        displayName = anomaly.ProductID
    }
    
    // 日付を読みやすい形式にフォーマット
    formattedDate := s.formatDateForDisplay(anomaly.Date)
    
    // テンプレートベースの質問
    question := fmt.Sprintf(
        "📈 %s に「%s」の売上が通常より %.0f 増加しました...",
        formattedDate,  // ← フォーマット済み
        displayName,    // ← 製品名優先
        // ...
    )
    
    return question, choices
}
```

#### 3. AI生成質問への適用

AIに渡すAnomalyオブジェクトにも、フォーマット済みの日付と表示名を使用：

```go
anomalyForAI := models.Anomaly{
    Date:        formattedDate, // フォーマット済みの日付
    ProductID:   displayName,    // 製品名（ProductNameがあればそれ、なければID）
    Description: fmt.Sprintf("売上%s (実績: %.0f, 期待値: %.0f)", ...),
}

result, err := s.azureOpenAIService.GenerateQuestionAndChoicesFromAnomaly(anomalyForAI)
```

## 動作例

### 日次分析の場合

**入力データ:**
```
Date: "2022-03-07"
ProductID: "P001"
ProductName: "ウィジェット A"
```

**生成される質問:**
```
📈 2022年3月7日 に「ウィジェット A」の売上が通常より 150 増加しました
（期待値: 200 → 実績: 350）。この時期に特別なイベント、キャンペーン、
または外的要因はありましたか？
```

### 週次分析の場合

**入力データ:**
```
Date: "2022-W10"
ProductID: "P002"
ProductName: "製品B"
```

**生成される質問:**
```
📉 2022年 第10週 に「製品B」の売上が通常より 500 減少しました
（期待値: 1500 → 実績: 1000）。この時期に売上減少の原因となった要因
（天候、競合、在庫切れなど）はありましたか？
```

### 月次分析の場合

**入力データ:**
```
Date: "2022-04"
ProductID: "P001"
ProductName: "製品A"
```

**生成される質問:**
```
📈 2022年4月 に「製品A」の売上が通常より 717 増加しました
（期待値: 2337 → 実績: 3054）。この時期に特別なイベント、キャンペーン、
または外的要因はありましたか？
```

## 製品名がない場合のフォールバック

ProductNameフィールドが空の場合は、自動的にProductIDを使用します：

```
📈 2022年4月 に「P001」の売上が通常より 717 増加しました...
```

これにより、後方互換性を保ちながら、製品名がある場合はより分かりやすい表示が可能になります。

## 関連ファイル

- `pkg/services/statistics_service.go`: 日付フォーマット関数とGenerateAIQuestion
- `pkg/services/azure_openai_service.go`: AI質問生成サービス
- `pkg/models/types.go`: AnomalyDetection構造体（ProductName含む）

## テスト方法

1. ファイル分析ページでCSVアップロード（ProductNameカラムあり/なし両方）
2. 異常検知後にチャットページで未回答の異常を確認
3. 生成された質問で製品名と日付が読みやすく表示されることを確認

## メリット

✅ **ユーザビリティ向上**: 「P001」より「製品A」の方が直感的  
✅ **日付の可読性**: 「2022-04」より「2022年4月」が読みやすい  
✅ **後方互換性**: 製品名がない場合は従来通りIDを表示  
✅ **粒度対応**: 日次・週次・月次すべての形式に対応

---

**更新日**: 2025年10月21日  
**関連ドキュメント**: [ANOMALY_DETECTION_WEEKLY_AGGREGATION.md](./ANOMALY_DETECTION_WEEKLY_AGGREGATION.md)
