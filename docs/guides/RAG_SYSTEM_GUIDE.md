# 🔍 RAGシステム（検索拡張生成）ガイド

## 📋 目次
1. [RAGとは](#ragとは)
2. [システムアーキテクチャ](#システムアーキテクチャ)
3. [ソース選定の流れ](#ソース選定の流れ)
4. [Qdrantコレクション構成](#qdrantコレクション構成)
5. [実装詳細](#実装詳細)
6. [カスタマイズ方法](#カスタマイズ方法)

---

## RAGとは

**RAG (Retrieval-Augmented Generation)** = 検索拡張生成

AIの回答生成時に、関連する過去のデータ・ドキュメントを検索して参照することで、より正確で文脈に沿った回答を生成する技術です。

### メリット
- ✅ **正確性向上**: ハルシネーション（事実と異なる情報の生成）を防ぐ
- ✅ **文脈理解**: 過去の会話を踏まえた回答が可能
- ✅ **最新情報**: リアルタイムデータを反映できる
- ✅ **専門性**: ドメイン固有の知識を活用できる

---

## システムアーキテクチャ

```
┌─────────────────────────────────────────────────────────┐
│                   ユーザーの質問                          │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│        HandleChatInput (pkg/handlers/ai_handler.go)     │
│  ┌──────────────────────────────────────────────────┐   │
│  │ 1. ファイル分析結果（req.Context）                │   │
│  │    └─ 直前に分析したCSV/Excelの統計情報          │   │
│  └──────────────────────────────────────────────────┘   │
│                     │                                    │
│                     ▼                                    │
│  ┌──────────────────────────────────────────────────┐   │
│  │ 2. 過去のチャット履歴を検索                       │   │
│  │    └─ SearchChatHistory (Top 3)                  │   │
│  │       Qdrant: chat_history コレクション           │   │
│  │       ベクトル類似度検索（コサイン類似度）         │   │
│  └──────────────────────────────────────────────────┘   │
│                     │                                    │
│                     ▼                                    │
│  ┌──────────────────────────────────────────────────┐   │
│  │ 3. システムドキュメントを検索                     │   │
│  │    └─ Search (Top 2)                             │   │
│  │       Qdrant: hunt_chat_documents コレクション     │   │
│  │       ベクトル類似度検索                          │   │
│  └──────────────────────────────────────────────────┘   │
│                     │                                    │
│                     ▼                                    │
│  ┌──────────────────────────────────────────────────┐   │
│  │ 4. 分析レポートを検索（条件付き）                 │   │
│  │    └─ SearchAnalysisReports (Top 2)              │   │
│  │       Qdrant: analysis_reports コレクション        │   │
│  │       条件: 質問に「分析」「相関」等を含む         │   │
│  └──────────────────────────────────────────────────┘   │
│                     │                                    │
│                     ▼                                    │
│  ┌──────────────────────────────────────────────────┐   │
│  │ 統合コンテキストを構築                            │   │
│  │  - ファイル分析                                   │   │
│  │  - 過去の会話 × 3                                 │   │
│  │  - ドキュメント × 2                               │   │
│  │  - 分析レポート × 2 (条件付き)                    │   │
│  └──────────────────────────────────────────────────┘   │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  ProcessChatWithHistory (azure_openai_service.go)       │
│  ┌──────────────────────────────────────────────────┐   │
│  │ システムプロンプト                                │   │
│  │  - HUNT システムの概要                            │   │
│  │  - 主要機能（8つのAPI）                           │   │
│  │  - 技術スタック                                   │   │
│  │  - 回答方針                                       │   │
│  └──────────────────────────────────────────────────┘   │
│                     │                                    │
│                     ▼                                    │
│  ┌──────────────────────────────────────────────────┐   │
│  │ Azure OpenAI API (GPT-4)                          │   │
│  │  - モデル: gpt-4                                  │   │
│  │  - max_tokens: 2000                               │   │
│  │  - temperature: 0.7                               │   │
│  └──────────────────────────────────────────────────┘   │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                    AI生成の回答                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │ - 統合コンテキストを考慮                          │   │
│  │ - 過去の文脈を理解                                │   │
│  │ - システム情報に基づく回答                        │   │
│  └──────────────────────────────────────────────────┘   │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  回答をチャット履歴として保存                            │
│  └─ SaveChatHistory (非同期)                            │
│     Qdrant: chat_history に追加                          │
└─────────────────────────────────────────────────────────┘
```

---

## ソース選定の流れ

### 🔄 Step 1: ファイル分析結果の取得

```go
// pkg/handlers/ai_handler.go
if req.Context != "" {
    ragContext.WriteString(req.Context)
    contextSources = append(contextSources, "現在のファイル分析")
}
```

**取得内容:**
- CSVファイル名
- データ点数
- 統計サマリー（平均、標準偏差、相関係数など）
- 気象データとの相関
- 回帰分析結果

**優先度:** 🔴 最高（直前の分析結果を優先）

---

### 🔍 Step 2: 過去のチャット履歴検索

```go
// pkg/handlers/ai_handler.go
chatHistory, err := ah.vectorStoreService.SearchChatHistory(
    ctx, 
    req.ChatMessage,  // 質問文
    "",               // セッションID（空=全履歴から検索）
    req.UserID,       // ユーザーID
    3,                // Top 3件
)
```

**検索方法:**
1. 質問文をベクトル化（Azure OpenAI text-embedding-3-small）
2. Qdrantで類似度検索（コサイン類似度）
3. 上位3件を取得

**取得内容:**
```
[2025-01-15 10:30:00] user: このシステムの機能は？
[2025-01-15 10:30:05] assistant: HUNTは需要予測システムで...
```

**優先度:** 🟡 高（文脈理解に重要）

---

### 📚 Step 3: システムドキュメント検索

```go
// pkg/handlers/ai_handler.go
searchResults, err := ah.vectorStoreService.Search(
    ctx, 
    req.ChatMessage,  // 質問文
    2,                // Top 2件
)
```

**コレクション:** `hunt_chat_documents`

**検索対象:**
- README.md
- API_MANUAL.md
- IMPLEMENTATION_SUMMARY.md
- TROUBLESHOOTING_AND_BEST_PRACTICES.md
- その他のMarkdownドキュメント

**優先度:** 🟢 中（システム情報の補完）

---

### 📊 Step 4: 分析レポート検索（条件付き）

```go
// pkg/handlers/ai_handler.go
if strings.Contains(strings.ToLower(req.ChatMessage), "分析") ||
   strings.Contains(strings.ToLower(req.ChatMessage), "相関") ||
   strings.Contains(strings.ToLower(req.ChatMessage), "ファイル") ||
   strings.Contains(strings.ToLower(req.ChatMessage), "レポート") {
    
    analysisResults, err := ah.vectorStoreService.SearchAnalysisReports(
        ctx, 
        req.ChatMessage, 
        2,
    )
}
```

**トリガーキーワード:**
- 「分析」
- 「相関」
- 「ファイル」
- 「レポート」

**取得内容:**
- ファイル名
- 分析日時
- データ点数
- 統計サマリー
- 相関分析結果
- 回帰分析結果

**優先度:** 🟢 中（キーワードがある場合のみ）

---

## Qdrantコレクション構成

### 1. `chat_history`

**用途:** 過去のチャット履歴を保存

**スキーマ:**
```json
{
  "id": "uuid",
  "vector": [0.1, 0.2, ...], // 1536次元
  "payload": {
    "type": "chat_history",
    "session_id": "session-123",
    "user_id": "user-456",
    "role": "user" | "assistant",
    "timestamp": "2025-01-15T10:30:00Z",
    "intent": "system_inquiry",
    "product_id": "P001",
    "tags": "[\"需要予測\",\"統計\"]",
    "keywords": "[\"相関\",\"回帰\"]"
  }
}
```

**インデックス:**
- `type` (Keyword)
- `session_id` (Keyword)
- `user_id` (Keyword)
- `timestamp` (Text)

---

### 2. `hunt_chat_documents`

**用途:** システムドキュメント・ナレッジベース

**スキーマ:**
```json
{
  "id": "doc-uuid",
  "vector": [0.1, 0.2, ...],
  "payload": {
    "type": "documentation",
    "title": "README",
    "category": "system_overview",
    "text": "# HUNT需要予測システム\n..."
  }
}
```

**ドキュメント例:**
- README.md → システム概要
- API_MANUAL.md → API仕様
- TROUBLESHOOTING_AND_BEST_PRACTICES.md → トラブルシューティング

---

### 3. `analysis_reports`

**用途:** ファイル分析レポートの保存

**スキーマ:**
```json
{
  "id": "report-uuid",
  "vector": [0.1, 0.2, ...],
  "payload": {
    "type": "analysis_report",
    "file_name": "sales_2024.csv",
    "analysis_date": "2025-01-15",
    "data_points": 365,
    "text": "{...JSON...}"
  }
}
```

**取得内容:**
- 統計サマリー
- 相関分析
- 回帰分析
- 異常検知結果

---

### 4. `anomaly_responses` / `anomaly_response_sessions`

**用途:** 異常対応の回答履歴（学習データ）

**スキーマ:**
```json
{
  "id": "resp-uuid",
  "vector": [0.1, 0.2, ...],
  "payload": {
    "type": "anomaly_response",
    "anomaly_date": "2025-01-10",
    "product_id": "P001",
    "question": "売上急増の原因は？",
    "answer": "新春キャンペーン実施",
    "tags": "[\"キャンペーン\"]",
    "impact": "positive",
    "impact_value": 30.0
  }
}
```

**活用方法:**
- AIの学習データとして使用
- 類似の異常発生時に過去の対応を参照
- パターン分析・洞察生成

---

## 実装詳細

### ベクトル検索の仕組み

#### 1. **テキストのベクトル化**

```go
// pkg/services/azure_openai_service.go
func (aos *AzureOpenAIService) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
    // Azure OpenAI text-embedding-3-small
    // 出力: 1536次元のベクトル
}
```

**モデル:** `text-embedding-3-small`  
**次元数:** 1536  
**特徴:** 多言語対応、高速、コストパフォーマンス良好

#### 2. **コサイン類似度の計算**

```
similarity = (A · B) / (||A|| × ||B||)

A: 質問文のベクトル
B: 保存されたドキュメントのベクトル

結果: -1 ~ 1 (1に近いほど類似)
```

#### 3. **Top-K検索**

```go
// pkg/services/vector_store_service.go
func (s *VectorStoreService) Search(ctx context.Context, query string, topK uint64) ([]*qdrant.ScoredPoint, error) {
    // 1. クエリをベクトル化
    vector, err := s.CreateEmbedding(ctx, query)
    
    // 2. Qdrantで検索
    result, err := s.qdrantPointsClient.Search(ctx, &qdrant.SearchPoints{
        CollectionName: collectionName,
        Vector:         vector,
        Limit:          topK,
        WithPayload:    &qdrant.WithPayloadSelector{...},
    })
    
    return result.Result, nil
}
```

---

### システムプロンプトの構成

```go
// pkg/services/azure_openai_service.go
systemPrompt := "あなたは、需要予測システム「HUNT」の専門家アシスタントです。\n\n" +
    "## システム概要\n" +
    "- **システム名**: HUNT (需要予測システム)\n" +
    "- **目的**: 製造業向けのAI需要予測・異常検知・学習システム\n" +
    "- **主要機能**:\n" +
    "  1. ファイル分析（CSV/Excelアップロード、相関分析、回帰分析）\n" +
    "  2. 需要予測（気温などの外部要因を考慮、製品別・期間別予測）\n" +
    "  3. 異常検知（3σ法による統計的異常検知、AIによる原因質問生成）\n" +
    "  4. AI学習（ユーザー回答を学習データとして蓄積、継続的な精度向上）\n" +
    "  5. RAG機能（過去の分析レポート、チャット履歴、回答履歴から検索）\n\n" +
    "## 技術スタック\n" +
    "- **フロントエンド**: Next.js (TypeScript, React), Tailwind CSS, Radix UI\n" +
    "- **バックエンド**: Go (Gin), Azure OpenAI API (GPT-4)\n" +
    "- **データベース**: Qdrant (ベクトルDB), PostgreSQL (将来的に)\n" +
    "- **デプロイ**: Vercel (フロント), Docker (バックエンド・Qdrant)\n\n" +
    "## API構成\n" +
    "- /api/v1/ai/analyze-file - ファイル分析\n" +
    "- /api/v1/ai/detect-anomalies - 異常検知\n" +
    "- /api/v1/ai/predict-sales - 売上予測\n" +
    "- /api/v1/ai/forecast-product - 製品別需要予測\n" +
    "- /api/v1/ai/anomaly-response-with-followup - 異常への回答と深掘り質問\n" +
    "- /api/v1/ai/chat-input - AIチャット（このエンドポイント）\n" +
    "- /api/v1/ai/anomaly-responses - 回答履歴取得\n" +
    "- /api/v1/ai/learning-insights - 学習した洞察の取得\n\n" +
    "## 回答方針\n" +
    "1. ユーザーがシステムについて質問した場合は、上記の情報を基に説明してください\n" +
    "2. 分析コンテキストが提供されている場合は、それを優先的に使用してください\n" +
    "3. 過去の会話履歴から学習し、継続的に文脈を把握してください\n" +
    "4. 需要予測に関する専門的な質問には、統計学的・ビジネス的な観点から答えてください\n" +
    "5. わからないことは正直に伝え、ドキュメントやAPIを参照するよう促してください"
```

---

## カスタマイズ方法

### 1. システムドキュメントの追加

**手順:**

```bash
# 1. ドキュメントをプロジェクトに追加
vim docs/new_feature.md

# 2. 初期化スクリプトを実行（開発時のみ）
make init-docs
```

**注意:**
- Markdown形式が推奨
- ファイルサイズは1MB以下
- 日本語・英語対応

---

### 2. 検索精度の調整

#### Top-K値の変更

```go
// pkg/handlers/ai_handler.go

// チャット履歴: 現在 Top 3
chatHistory, err := ah.vectorStoreService.SearchChatHistory(ctx, req.ChatMessage, "", req.UserID, 5) // 5件に増やす

// ドキュメント: 現在 Top 2
searchResults, err := ah.vectorStoreService.Search(ctx, req.ChatMessage, 3) // 3件に増やす
```

**トレードオフ:**
- ✅ 多い → より多くの情報を参照できる
- ❌ 多い → トークン消費増加、応答時間増加

---

### 3. フィルタリング条件の追加

```go
// pkg/services/vector_store_service.go
func (s *VectorStoreService) SearchChatHistory(...) {
    // 例: 特定期間のみ検索
    filterConditions = append(filterConditions, &qdrant.Condition{
        ConditionOneOf: &qdrant.Condition_Range{
            Range: &qdrant.Range{
                Key: "timestamp",
                Gte: &qdrant.Range_DatetimeGte{
                    DatetimeGte: time.Now().Add(-7*24*time.Hour), // 1週間以内
                },
            },
        },
    })
}
```

---

### 4. システムプロンプトの更新

```go
// pkg/services/azure_openai_service.go

// 新しい機能を追加した場合、ここを更新
systemPrompt := "..." +
    "  6. 新機能（説明）\n" +
    ...
```

---

## ベストプラクティス

### ✅ DO

1. **定期的にドキュメントを更新する**
   - 新機能追加時は必ずシステムプロンプトを更新
   - `make init-docs` でベクトルDBに反映

2. **チャット履歴を定期的にクリーンアップ**
   - 古いセッション（例: 30日以上前）は削除
   - ストレージコストの削減

3. **ベクトル検索のパフォーマンスを監視**
   - Qdrantの`/metrics`エンドポイントを活用
   - 検索時間が100ms以上なら要最適化

4. **コンテキスト長を管理**
   - GPT-4のトークン制限: 8,192トークン
   - システムプロンプト + ユーザー入力 + コンテキスト < 6,000トークン

---

### ❌ DON'T

1. **Top-K値を大きくしすぎない**
   - 推奨: 3-5件
   - 10件以上は非推奨

2. **すべての質問で全コレクションを検索しない**
   - 必要に応じて条件分岐
   - パフォーマンス劣化の原因

3. **システムプロンプトに機密情報を含めない**
   - APIキー、パスワード、個人情報は厳禁
   - 環境変数で管理

---

## トラブルシューティング

### 問題: 検索結果が返ってこない

**原因:**
- Qdrantコレクションが空
- ドキュメントが投入されていない

**解決策:**
```bash
make init-docs
```

---

### 問題: 回答が不正確

**原因:**
- コンテキストが不足
- システムプロンプトが古い

**解決策:**
1. Top-K値を増やす
2. システムプロンプトを更新
3. ドキュメントを追加

---

### 問題: 応答が遅い

**原因:**
- ベクトル検索が重い
- コンテキストが大きすぎる

**解決策:**
1. Top-K値を減らす
2. Qdrantのインデックスを最適化
3. 古いデータを削除

---

## 参考リンク

- [Qdrant公式ドキュメント](https://qdrant.tech/documentation/)
- [Azure OpenAI API](https://learn.microsoft.com/ja-jp/azure/ai-services/openai/)
- [RAGについて（日本語解説）](https://qiita.com/tags/rag)

---

**Last Updated:** 2025-10-20  
**Version:** 1.0.0
