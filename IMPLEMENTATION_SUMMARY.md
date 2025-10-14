# チャット履歴統合とRAG強化 - 実装サマリー

## 📅 実装日
2025年10月13日

## 🎯 実装目標

1. **チャット履歴の統合**: 過去の会話履歴からも学習させる
2. **RAG（Retrieval-Augmented Generation）の強化**: より文脈を理解した回答を生成

## ✅ 実装完了項目

### 1. データモデルの拡張 (`pkg/models/types.go`)

#### 新規追加
- `ChatHistoryEntry`: チャット履歴エントリー（ID, SessionID, UserID, Role, Message, etc.）
- `Metadata`: メタデータ構造（Intent, ProductID, TopicKeywords, RelevanceScore）
- `ChatHistorySaveRequest`: 履歴保存リクエスト
- `ChatHistorySearchRequest`: 履歴検索リクエスト
- `ChatHistorySearchResponse`: 履歴検索レスポンス

#### 既存型の拡張
- `ChatRequest`: SessionID, UserID フィールド追加
- `ChatResponse`: RelevantHistory, ContextSources, ConversationCount フィールド追加

### 2. ベクトルストアサービスの強化 (`pkg/services/vector_store_service.go`)

#### 新機能
- **`SaveChatHistory`**: チャット履歴をQdrantに保存
  - エントリーをベクトル化して保存
  - メタデータ（session_id, user_id, role, intent, tags）を含める
  - コレクション: `chat_history`

- **`SearchChatHistory`**: チャット履歴を検索（RAG機能の中核）
  - ベクトル類似度検索
  - フィルタ条件: type, session_id, user_id
  - 関連性スコア付きで結果を返す

- **`GetRecentChatHistory`**: 最近の履歴を取得

- **`getStringFromPayload`**: ヘルパー関数（Qdrantペイロードから文字列取得）

### 3. Azure OpenAI サービスの拡張 (`pkg/services/azure_openai_service.go`)

#### 新機能
- **`ProcessChatWithHistory`**: 過去の履歴を活用したAI応答生成
  - システムプロンプトに「過去の会話履歴から学習」を明記
  - 関連する過去の会話をコンテキストに含める
  - より文脈を理解した回答を生成

- **`ExtractMetadataFromMessage`**: メッセージからメタデータを抽出
  - 意図（intent）の抽出
  - キーワードの抽出
  - AI分析により自動タグ付け

### 4. AIハンドラーの強化 (`pkg/handlers/ai_handler.go`)

#### `ChatInput` メソッドの改良

**追加機能:**

1. **セッション管理**
   - SessionID でチャットの継続性を管理
   - UserID でユーザー別の履歴を分離

2. **メタデータ自動抽出**
   - メッセージから意図とキーワードを抽出
   - チャット履歴に自動的にタグ付け

3. **履歴の自動保存**
   - ユーザーメッセージを `ChatHistoryEntry` として保存
   - AI応答も同様に保存
   - 非同期処理でパフォーマンスを維持

4. **RAG検索の多層化**
   ```
   ┌─────────────────────────────────────┐
   │   ユーザーのメッセージ              │
   └──────────────┬──────────────────────┘
                  │
                  ▼
   ┌─────────────────────────────────────┐
   │   過去のチャット履歴から検索        │
   │   (chat_history コレクション)       │
   └──────────────┬──────────────────────┘
                  │
                  ▼
   ┌─────────────────────────────────────┐
   │   一般ドキュメントから検索          │
   │   (hunt_chat_documents)             │
   └──────────────┬──────────────────────┘
                  │
                  ▼
   ┌─────────────────────────────────────┐
   │   分析レポートから検索              │
   │   (analysis_report)                 │
   └──────────────┬──────────────────────┘
                  │
                  ▼
   ┌─────────────────────────────────────┐
   │   すべてのコンテキストを統合        │
   └──────────────┬──────────────────────┘
                  │
                  ▼
   ┌─────────────────────────────────────┐
   │   AI が過去の履歴を踏まえて回答     │
   └─────────────────────────────────────┘
   ```

5. **レスポンスの拡張**
   - `session_id`: 会話の継続に使用
   - `relevant_history`: 参照した過去の会話
   - `context_sources`: コンテキストのソース情報
   - `conversation_count`: 使用した会話数

## 🔧 技術的な詳細

### Qdrant コレクション構造

**コレクション名**: `chat_history`

**ベクトル設定**:
- サイズ: 1536 (text-embedding-3-small)
- 距離メトリック: Cosine

**メタデータフィールド**:
```json
{
  "type": "chat_history",
  "session_id": "session-xxx",
  "user_id": "user-xxx",
  "role": "user" | "assistant",
  "timestamp": "2025-10-13T10:30:00Z",
  "intent": "需要予測" | "異常分析" | "データ分析" | "質問" | "その他",
  "product_id": "P001",
  "date_range": "2024-01-01:2024-12-31",
  "tags": "[\"需要予測\",\"分析\"]",
  "keywords": "[\"売上\",\"気温\",\"相関\"]"
}
```

### RAG検索フロー

1. **ユーザーメッセージ受信**
2. **メタデータ抽出** (AI により意図とキーワードを抽出)
3. **マルチソース検索**:
   - チャット履歴検索 (3件)
   - 一般ドキュメント検索 (2件)
   - 分析レポート検索 (2件、条件付き)
4. **コンテキスト統合**
5. **AI応答生成** (過去の履歴を考慮)
6. **履歴保存** (ユーザーメッセージ + AI応答)
7. **レスポンス返却**

### パフォーマンス最適化

- **非同期保存**: チャット履歴の保存は非同期で実行（レスポンス時間に影響なし）
- **適切なtopK**: 検索件数を制限（デフォルト: 3件）
- **フィルタリング**: session_id, user_id でフィルタして検索範囲を絞る
- **並列処理**: 複数の検索を並行して実行

## 📊 使用例

### 初回の会話

**リクエスト**:
```bash
curl -X POST http://localhost:8080/api/v1/ai/chat \
  -H "Content-Type: application/json" \
  -d '{
    "chat_message": "需要予測について教えてください",
    "user_id": "user-001"
  }'
```

**レスポンス**:
```json
{
  "success": true,
  "response": {
    "text": "需要予測とは...",
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "relevant_history": [],
    "context_sources": ["ナレッジベース"],
    "conversation_count": 0
  }
}
```

### 続きの会話（履歴を活用）

**リクエスト**:
```bash
curl -X POST http://localhost:8080/api/v1/ai/chat \
  -H "Content-Type: application/json" \
  -d '{
    "chat_message": "具体的な方法を教えてください",
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": "user-001"
  }'
```

**レスポンス**:
```json
{
  "success": true,
  "response": {
    "text": "先ほどお話しした需要予測について、具体的な方法は...",
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "relevant_history": [
      "[2025-10-13T10:30:00Z] user: 需要予測について教えてください",
      "[2025-10-13T10:30:15Z] assistant: 需要予測とは..."
    ],
    "context_sources": [
      "過去の会話 (2025-10-13T10:30:00Z)",
      "過去の会話 (2025-10-13T10:30:15Z)",
      "ナレッジベース"
    ],
    "conversation_count": 2
  }
}
```

## 🎁 メリット

### 1. ユーザー体験の向上
- AIが過去の会話を覚えている
- 同じ説明を繰り返さなくてよい
- 文脈を理解した回答を得られる

### 2. 精度の向上
- 過去の質問と回答から学習
- ユーザーの意図をより正確に理解
- 関連性の高い情報を提供

### 3. パーソナライゼーション
- ユーザーごとの履歴を管理
- 個別のニーズに対応
- セッションごとの文脈保持

### 4. データの蓄積と活用
- 会話データが資産として蓄積
- パターン分析が可能
- サービス改善に活用

## 📈 今後の拡張案

1. **時系列フィルタ**: 期間を指定した履歴検索
2. **重要度スコアリング**: 会話の重要度に基づく優先順位付け
3. **トピック分類**: より高度な意図分類とカテゴリ化
4. **会話要約**: 長い会話履歴の自動要約
5. **感情分析**: ユーザーの感情を考慮した応答
6. **マルチモーダル**: 画像やファイルを含む履歴管理
7. **会話の分岐管理**: 複数のトピックを並行処理
8. **推薦システム**: 過去の履歴に基づく提案

## 🧪 テスト方法

### 1. ビルド確認
```bash
cd /Users/kairemix/dev/mci/hunt-chat-api
go build -o main cmd/server/main.go
```

### 2. サーバー起動
```bash
make run
```

### 3. 初回チャット
```bash
curl -X POST http://localhost:8080/api/v1/ai/chat \
  -H "Content-Type: application/json" \
  -d '{"chat_message": "需要予測について教えてください", "user_id": "test-user"}'
```

### 4. 続きのチャット
```bash
# 前のレスポンスから session_id を取得して使用
curl -X POST http://localhost:8080/api/v1/ai/chat \
  -H "Content-Type: application/json" \
  -d '{
    "chat_message": "もっと詳しく教えてください",
    "session_id": "<取得したsession_id>",
    "user_id": "test-user"
  }'
```

### 5. 結果確認
- `relevant_history` に過去の会話が含まれているか確認
- `conversation_count` が増えているか確認
- AI応答が文脈を理解しているか確認

## 📝 ログ出力

実装により以下のログが出力されます:

```
✅ ユーザーメッセージを履歴に保存: SessionID=xxx
📚 N件の関連する過去の会話を取得しました
チャット履歴検索: N 件の関連する会話を取得しました
✅ AI応答を履歴に保存: SessionID=xxx
```

## 🔒 セキュリティ考慮

- **ユーザー隔離**: user_id でデータを分離
- **セッション管理**: session_id で会話をグルーピング
- **メタデータ検証**: 入力値のバリデーション
- **アクセス制御**: 将来的な認証・認可の基盤

## 📚 関連ドキュメント

- [CHAT_HISTORY_RAG.md](./CHAT_HISTORY_RAG.md): 詳細な技術ドキュメント
- [API_MANUAL.md](./API_MANUAL.md): API仕様書
- [README.md](./README.md): プロジェクト概要

## 🎉 まとめ

チャット履歴の統合とRAG強化により、AIアシスタントは以下が可能になりました:

✅ 過去の会話を記憶して文脈を理解  
✅ ユーザーの意図をより正確に把握  
✅ 関連する情報を複数ソースから取得  
✅ パーソナライズされた応答を提供  
✅ 会話データを資産として蓄積  

これにより、需要予測システムとしての実用性が大幅に向上しました！
