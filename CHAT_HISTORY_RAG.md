# チャット履歴統合とRAG強化機能

## 概要
このドキュメントでは、チャット履歴を活用したRAG (Retrieval-Augmented Generation) 機能の実装について説明します。

## 実装した機能

### 1. チャット履歴モデル (`pkg/models/types.go`)

#### 新しい型の追加
- **`ChatHistoryEntry`**: チャット履歴の1エントリー
  - ID, SessionID, UserID, Role, Message, Context, Timestamp, Tags, Metadata, CreatedAt
- **`Metadata`**: チャット履歴のメタデータ
  - Intent（意図）, ProductID, DateRange, TopicKeywords, SentimentScore, RelevanceScore
- **`ChatHistorySearchRequest`**: チャット履歴検索リクエスト
- **`ChatHistorySearchResponse`**: チャット履歴検索レスポンス

#### 既存型の拡張
- **`ChatRequest`**: SessionID, UserID フィールドを追加
- **`ChatResponse`**: RelevantHistory, ContextSources, ConversationCount フィールドを追加

### 2. ベクトルストアサービス (`pkg/services/vector_store_service.go`)

#### 新しいメソッド

##### `SaveChatHistory(ctx, entry)`
- チャット履歴をQdrantに保存
- エントリーをテキストベクトル化して保存
- メタデータ（session_id, user_id, role, intent, tags など）を含める
- コレクション: `chat_history`

##### `SearchChatHistory(ctx, query, sessionID, userID, topK)`
- チャット履歴をベクトル検索
- フィルタ条件: type, session_id, user_id
- 関連性スコア付きで結果を返す
- RAG機能の中核

##### `GetRecentChatHistory(ctx, sessionID, limit)`
- 特定セッションの最近のチャット履歴を取得

### 3. Azure OpenAI サービス (`pkg/services/azure_openai_service.go`)

#### 新しいメソッド

##### `ProcessChatWithHistory(chatMessage, context, relevantHistory)`
- 過去のチャット履歴を活用してAI応答を生成
- システムプロンプトに「過去の会話履歴から学習」を明記
- 関連する過去の会話をコンテキストに含める

##### `ExtractMetadataFromMessage(message)`
- メッセージから意図（intent）とキーワードを抽出
- AI分析により自動的にタグ付け

### 4. AIハンドラー (`pkg/handlers/ai_handler.go`)

#### `ChatInput` の強化

**変更点:**

1. **セッション管理**
   - `SessionID` と `UserID` をリクエストから受け取る
   - セッションIDがない場合は自動生成

2. **メタデータ抽出**
   - メッセージから意図とキーワードを自動抽出
   - チャット履歴に紐付け

3. **チャット履歴の保存**
   - ユーザーメッセージを `ChatHistoryEntry` として保存
   - AI応答も同様に保存
   - 非同期処理でパフォーマンスを維持

4. **RAG検索の強化**
   - 過去のチャット履歴から関連する会話を検索
   - 検索結果をコンテキストに含める
   - 一般ドキュメント、分析レポートと併用

5. **レスポンスの拡張**
   - `session_id`: セッションID
   - `relevant_history`: 関連する過去の会話
   - `context_sources`: コンテキストのソース情報
   - `conversation_count`: 使用した過去の会話数

## 使用方法

### チャットAPIの呼び出し

```bash
curl -X POST http://localhost:8080/api/v1/ai/chat \
  -H "Content-Type: application/json" \
  -d '{
    "chat_message": "先月の売上分析結果を教えてください",
    "session_id": "session-12345",
    "user_id": "user-001",
    "context": "過去のファイル分析コンテキスト"
  }'
```

### レスポンス例

```json
{
  "success": true,
  "response": {
    "text": "先月の売上分析結果について...",
    "session_id": "session-12345",
    "relevant_history": [
      "[2025-10-13T10:30:00Z] user: 売上データをアップロードしました",
      "[2025-10-13T10:31:00Z] assistant: データを分析しました..."
    ],
    "context_sources": [
      "現在のファイル分析",
      "過去の会話 (2025-10-13T10:30:00Z)",
      "分析レポート (sales_data.csv)"
    ],
    "conversation_count": 2
  }
}
```

## データベース構造

### Qdrantコレクション: `chat_history`

**ベクトルサイズ**: 1536 (text-embedding-3-small)

**メタデータフィールド**:
- `type`: "chat_history"
- `session_id`: セッションID
- `user_id`: ユーザーID
- `role`: "user" または "assistant"
- `timestamp`: タイムスタンプ (RFC3339)
- `intent`: 意図（"需要予測", "異常分析", etc.）
- `product_id`: 関連商品ID
- `date_range`: 関連期間
- `tags`: タグ配列（JSON文字列）
- `keywords`: キーワード配列（JSON文字列）

## RAG検索のフロー

```
1. ユーザーがメッセージを送信
   ↓
2. メッセージから意図とキーワードを抽出
   ↓
3. チャット履歴を検索
   - ベクトル類似度検索
   - session_id, user_id でフィルタ
   ↓
4. 関連するドキュメントを検索
   - 一般ドキュメント
   - 分析レポート
   ↓
5. すべてのコンテキストを統合
   ↓
6. AI応答を生成
   ↓
7. ユーザーメッセージとAI応答を履歴に保存
   ↓
8. レスポンスを返す（履歴情報を含む）
```

## パフォーマンス最適化

1. **非同期保存**: チャット履歴の保存は非同期で実行
2. **適切なtopK**: 検索件数を制限（デフォルト: 3件）
3. **フィルタリング**: session_id, user_id でフィルタして検索範囲を絞る

## セキュリティ考慮事項

1. **ユーザー隔離**: user_id でデータを分離
2. **セッション管理**: session_id で会話をグルーピング
3. **メタデータ検証**: 入力値のバリデーション

## 今後の改善案

1. **時系列フィルタ**: timestampを使った期間フィルタリング
2. **重要度スコアリング**: 会話の重要度に基づく検索
3. **トピック分類**: より高度な意図分類
4. **会話要約**: 長い会話履歴の要約機能
5. **感情分析**: SentimentScoreの活用
6. **マルチモーダル**: 画像やファイルを含む会話履歴

## テスト手順

1. **サーバー起動**
   ```bash
   make run
   ```

2. **初回メッセージ送信**
   ```bash
   curl -X POST http://localhost:8080/api/v1/ai/chat \
     -H "Content-Type: application/json" \
     -d '{"chat_message": "需要予測について教えてください", "user_id": "test-user"}'
   ```

3. **セッションIDを使って続きの会話**
   ```bash
   curl -X POST http://localhost:8080/api/v1/ai/chat \
     -H "Content-Type: application/json" \
     -d '{
       "chat_message": "具体的な方法を教えてください",
       "session_id": "<前回のレスポンスのsession_id>",
       "user_id": "test-user"
     }'
   ```

4. **レスポンスの確認**
   - `relevant_history` に過去の会話が含まれているか確認
   - `conversation_count` が増えているか確認

## ログ出力

実装では以下のログが出力されます:

- ✅ ユーザーメッセージを履歴に保存: SessionID=xxx
- ✅ AI応答を履歴に保存: SessionID=xxx
- 📚 N件の関連する過去の会話を取得しました
- チャット履歴検索: N 件の関連する会話を取得しました

## トラブルシューティング

### チャット履歴が保存されない
- Qdrantが起動しているか確認
- `chat_history` コレクションが作成されているか確認
- ログでエラーメッセージを確認

### 過去の会話が検索されない
- `session_id` または `user_id` が一致しているか確認
- ベクトル検索のクエリが適切か確認
- topK パラメータを増やしてみる

### パフォーマンスが遅い
- 非同期処理が正常に動作しているか確認
- topK を減らして検索件数を制限
- Qdrantのインデックスを最適化

## 関連ファイル

- `pkg/models/types.go`: データモデル定義
- `pkg/services/vector_store_service.go`: ベクトルストア操作
- `pkg/services/azure_openai_service.go`: AI処理
- `pkg/handlers/ai_handler.go`: APIハンドラー
- `qdrant_storage/`: Qdrantローカルストレージ（Gitignore推奨）
