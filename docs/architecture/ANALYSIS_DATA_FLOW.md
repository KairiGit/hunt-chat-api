# 📊 分析結果の保存と活用フロー

## 概要

ファイル分析で得られた相関分析結果は、**Qdrant（ベクトルデータベース）**に保存され、複数の方法で活用されます。

---

## 🔄 データフロー全体像

```
┌─────────────────────────────────────────────────────────────┐
│ 1. ファイルアップロード & 分析                                    │
│    POST /api/v1/ai/analyze-file                              │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. 相関分析実行                                                │
│    ├─ 天気データとの相関（気温、湿度）                            │
│    └─ 経済データとの相関（日経平均、為替、原油価格など）           │
│       ※ 上位3件のみ抽出                                         │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. Qdrant に保存                                              │
│    Collection: hunt_chat_documents                           │
│    Type: analysis_report                                     │
│    │                                                          │
│    ├─ ベクトル化されたサマリー（検索用）                          │
│    └─ メタデータに完全なJSON（取得用）                           │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ↓
┌─────────────────────────────────────────────────────────────┐
│ 4. 活用方法                                                    │
│    ├─ A. レポート一覧表示                                       │
│    ├─ B. レポート詳細表示                                       │
│    ├─ C. チャットでの文脈として利用（RAG）                        │
│    └─ D. 過去の分析パターン学習                                 │
└─────────────────────────────────────────────────────────────┘
```

---

## 💾 保存される情報

### Qdrant に保存されるデータ構造

```json
{
  "id": "report-uuid-12345",
  "vector": [0.123, -0.456, ...],  // ← サマリーをベクトル化
  "payload": {
    "type": "analysis_report",
    "file_name": "sales_data.csv",
    "analysis_date": "2025-10-23T10:30:00Z",
    "full_report_json": "{...}"  // ← 完全なレポートJSON
  }
}
```

### レポートJSON の内容（full_report_json）

```json
{
  "report_id": "abc-123",
  "file_name": "sales_data.csv",
  "analysis_date": "2025-10-23T10:30:00Z",
  "data_points": 365,
  "date_range": "2024-01-01 〜 2024-12-31",
  "weather_matches": 350,
  
  "correlations": [
    {
      "factor": "temperature",
      "correlation_coef": 0.65,
      "p_value": 0.001,
      "sample_size": 350,
      "interpretation": "強い正の相関（統計的に有意）"
    },
    {
      "factor": "NIKKEI_yがxに対して+5日遅れ",
      "correlation_coef": 0.52,
      "p_value": 0.015,
      "sample_size": 340,
      "interpretation": "中程度の正の相関（統計的に有意）"
    },
    {
      "factor": "USDJPY_lag=0",
      "correlation_coef": -0.38,
      "p_value": 0.032,
      "sample_size": 340,
      "interpretation": "弱い負の相関（統計的に有意）"
    }
  ],
  
  "anomalies": [...],
  "regression": {...},
  "recommendations": [...],
  "ai_insights": "..."
}
```

---

## 🎯 活用方法の詳細

### A. レポート一覧表示

**API**: `GET /api/v1/ai/analysis-reports`

```javascript
// フロントエンド: src/app/analysis/page.tsx
const response = await fetch('/api/proxy/analysis-reports');
const { reports } = await response.json();

// 表示内容
// - レポートID
// - ファイル名
// - 分析日時
// - データ点数
// - 異常件数
```

**用途**:
- 過去に実施した分析の履歴を確認
- 再分析が必要な期間の特定
- 分析トレンドの可視化

---

### B. レポート詳細表示

**API**: `GET /api/v1/ai/analysis-report?report_id={id}`

```javascript
// クリックでレポート詳細を表示
const response = await fetch(`/api/proxy/analysis-report?report_id=${reportId}`);
const { report } = await response.json();

// AnalysisReportView コンポーネントで表示
<AnalysisReportView report={report} />
```

**表示内容**:
- ✅ **相関分析結果**（天気・経済の上位3件）
  - アイコン付き表示（🌡️💧📈💱🛢️）
  - 統計的有意性の視覚化
  - タイムラグ情報
- ✅ 異常検知サマリー
- ✅ 回帰分析結果
- ✅ AI による洞察
- ✅ 推奨事項

**用途**:
- 詳細な相関パターンの確認
- 異常発生時の外部要因の特定
- 意思決定のエビデンス

---

### C. チャットでの文脈として利用（RAG: Retrieval Augmented Generation）

**API**: `POST /api/v1/ai/chat-input`

```json
{
  "chat_message": "先月の売上急減の原因は何ですか？"
}
```

**内部処理フロー**:

```go
// 1. ユーザーの質問をベクトル化
queryVector := EmbedText("先月の売上急減の原因は何ですか？")

// 2. Qdrantから類似した分析レポートを検索
searchResults := qdrant.Search(
    collectionName: "hunt_chat_documents",
    vector: queryVector,
    filter: {type: "analysis_report"},
    limit: 3
)

// 3. 検索結果から関連情報を抽出
context := ""
for _, result := range searchResults {
    report := ParseJSON(result.Payload["full_report_json"])
    
    // 相関情報を抽出
    context += "過去の分析結果:\n"
    context += fmt.Sprintf("- 日経平均と5日遅れの相関: %.2f\n", report.Correlations[0].CorrelationCoef)
    context += fmt.Sprintf("- 推奨事項: %s\n", report.Recommendations[0])
}

// 4. 文脈を含めてAIに質問
response := AzureOpenAI.Chat(
    systemPrompt: "あなたは需要予測の専門家です",
    context: context,
    userQuestion: "先月の売上急減の原因は何ですか？"
)
```

**AIの回答例**:
```
先月の売上急減について、過去の分析データから以下の要因が考えられます：

1. **経済指標の影響**: 
   過去の分析では、日経平均が下落すると5日後に売上が減少する
   パターンが確認されています（相関係数: 0.52）。
   先月初旬の株価下落が中旬の売上に影響した可能性があります。

2. **天気の影響**:
   気温との正の相関（0.65）があるため、先月の低温が
   売上減少に寄与した可能性があります。

推奨事項：
- 株価動向を1週間前から監視
- 天気予報を活用した在庫調整
```

**用途**:
- 過去のパターンに基づく原因分析
- データドリブンな意思決定支援
- 自然言語での分析結果の検索

---

### D. 過去の分析パターン学習

**API**: `GET /api/v1/ai/learning-insights`

```javascript
// 複数の分析レポートから学習パターンを抽出
const response = await fetch('/api/proxy/learning-insights');
const { insights } = await response.json();
```

**抽出される学習内容**:

1. **共通の相関パターン**
   ```
   - 気温と売上: 常に正の相関（平均 r=0.63）
   - 日経平均: 5-7日の遅れ相関が頻出
   - 為替レート: 月次では相関強、日次では弱い
   ```

2. **異常の傾向**
   ```
   - 月曜日に異常検出が多い（60%）
   - 連休後の急増パターン（平均+35%）
   - 季節要因: 夏季に天気影響大
   ```

3. **予測精度の改善**
   ```
   - 天気データのみ: R²=0.45
   - 天気+経済データ: R²=0.62 ← 17%改善
   - 遅れ相関を考慮: R²=0.71 ← さらに9%改善
   ```

**用途**:
- 予測モデルの改善
- 重要指標の特定
- 業界ベンチマークとの比較

---

## 🔍 相関結果の絞り込み（上位3件）

### 実装箇所

**バックエンド**: `pkg/services/statistics_service.go`

```go
// AnalyzeSalesEconomicCorrelation
// 相関係数の絶対値でソート（降順）
sort.Slice(allResults, func(i, j int) bool {
    return math.Abs(allResults[i].CorrelationCoef) > math.Abs(allResults[j].CorrelationCoef)
})

// 上位3件のみを返す（最も有意な相関のみを表示）
if len(allResults) > 3 {
    allResults = allResults[:3]
    log.Printf("📊 経済データ相関: 上位3件に絞り込みました")
}
```

### 絞り込みの理由

1. **ノイズの削減**: 弱い相関は誤解を招く可能性
2. **視認性の向上**: 重要な情報に集中できる
3. **処理速度**: データ量を削減
4. **統計的信頼性**: 有意性の高い相関のみを報告

### 表示される相関

- 天気データ: 気温、湿度（最大2件）
- 経済データ: 上位3件
- **合計: 最大5件の相関が表示される**

---

## 📈 データの活用例

### ケース1: 売上予測の改善

```
1. 過去3ヶ月分のファイルを分析
   → 日経平均との+5日遅れ相関を発見

2. 予測モデルに組み込み
   → 日経平均の5日前の値を説明変数に追加

3. 予測精度が15%向上
```

### ケース2: 在庫最適化

```
1. 異常検知で連休後の急増パターンを特定
   
2. チャットで「連休前の在庫戦略は？」と質問
   → RAGが過去の分析を参照
   → 「連休前に通常の1.3倍の在庫を推奨」と回答

3. 実行して在庫切れを回避
```

### ケース3: 原因分析

```
1. 売上急減が発生

2. チャットで「今週の売上減少の原因は？」
   → 過去の相関パターンから分析
   → 「先週の株価下落が影響している可能性（5日遅れ相関）」

3. 外部要因を特定し、対策を実施
```

---

## 🔐 データ管理

### 保存期間
- デフォルト: 無期限
- 削除API: `DELETE /api/v1/ai/analysis-reports`

### プライバシー
- ファイル名とメタデータのみ保存
- 実データは保存されない（集計値のみ）

### パフォーマンス
- ベクトル検索: ~50ms
- レポート取得: ~10ms
- チャット文脈生成: ~200ms

---

## 📊 まとめ

### 保存される情報
✅ 相関分析結果（天気・経済の上位3件）
✅ 異常検知結果
✅ 回帰分析結果
✅ AI洞察と推奨事項

### 活用方法
✅ レポート一覧・詳細表示
✅ チャットでのRAG検索
✅ 過去パターンの学習
✅ 予測モデルの改善

### メリット
✅ データドリブンな意思決定
✅ 過去の知見の再利用
✅ AIによる自然言語での検索
✅ 継続的な精度向上

**→ 分析結果は単なるレポートではなく、組織の知識資産として蓄積・活用されます！**
