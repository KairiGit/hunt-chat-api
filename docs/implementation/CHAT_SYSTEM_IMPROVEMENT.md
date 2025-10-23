# チャット機能の改善 - システム情報統合

## 🎯 目的
分析結果がなくてもチャット機能を使えるようにし、システム自体について質問できるようにする。

## ✅ 実装した変更

### 1. フロントエンド (`src/app/chat/page.tsx`)
- **分析必須制約を削除**: `analysisSummary`がなくてもチャット可能に
- **動的なプロンプト例**: 分析結果の有無で表示される質問例を切り替え
  - 分析あり: データ傾向、異常値、相関
  - 分析なし: システム機能、予測の仕組み、API利用方法
- **プレースホルダーの変更**: コンテキストに応じた案内

### 2. バックエンド (`pkg/services/azure_openai_service.go`)
- **システムプロンプトの拡張**: `ProcessChatWithHistory`関数に詳細なシステム情報を追加
  - システム概要 (HUNT需要予測システム)
  - 主要機能 (5つ)
  - 技術スタック (フロント・バック・DB)
  - API構成 (8つのエンドポイント)
  - 回答方針 (5つのガイドライン)

### 3. システムドキュメント初期化スクリプト (`scripts/init_system_docs.go`)
- **目的**: システムドキュメント(.md)をベクトルDBに投入
- **対象ドキュメント**:
  - README.md
  - API_MANUAL.md
  - AI_LEARNING_GUIDE.md
  - IMPLEMENTATION_SUMMARY.md
  - CHAT_HISTORY_RAG.md
  - WEEKLY_ANALYSIS_GUIDE.md
  - TROUBLESHOOTING_AND_BEST_PRACTICES.md
  - 要件定義.md
  - ワークフロー.md
  - UML.md
- **カテゴリ分類**: api, ai, design, development, usage, general
- **メタデータ**: file_name, category, description

### 4. Makefile
```bash
make init-docs  # システムドキュメントをベクトルDBに投入
```

## 📊 使い方

### 初回セットアップ
```bash
# 1. バックエンドを起動
make run

# 2. システムドキュメントをベクトルDBに投入
make init-docs

# 3. フロントエンドを起動
npm run dev
```

### チャット機能の使い方
1. **ファイル分析前**:
   - 「このシステムの機能を教えて」
   - 「需要予測の仕組みを教えて」
   - 「APIの使い方を教えて」

2. **ファイル分析後**:
   - 「このデータの傾向を教えて」
   - 「異常値について詳しく教えて」
   - 「相関関係を教えて」

## 🔍 RAG検索の流れ

```
ユーザー質問
    ↓
過去のチャット履歴検索 (3件)
    ↓
システムドキュメント検索 (2件) ← 🆕 追加
    ↓
分析レポート検索 (2件) ※キーワードマッチ時
    ↓
すべてのコンテキストを統合
    ↓
Azure OpenAI (GPT-4) で回答生成
    ↓
ストリーミングレスポンス
```

## 💡 メリット

1. **初心者にやさしい**: システムの使い方から学べる
2. **ドキュメント不要**: チャットでシステム情報を取得
3. **コンテキスト保持**: 分析前後で会話が継続
4. **RAG活用**: 最新のドキュメントを自動参照
5. **スケーラブル**: 新しいドキュメントを追加するだけで知識拡張

## 🚀 今後の拡張案

1. **API仕様書の自動解析**: OpenAPI/Swaggerから自動生成
2. **コード例の埋め込み**: サンプルコードをドキュメントに追加
3. **バージョン管理**: ドキュメントの更新履歴を追跡
4. **多言語対応**: 英語版ドキュメントの追加
5. **FAQ自動生成**: よくある質問を学習して自動回答

## 📝 関連ファイル

- `src/app/chat/page.tsx` - チャットUI
- `pkg/services/azure_openai_service.go` - AIサービス
- `pkg/handlers/ai_handler.go` - チャットハンドラー
- `scripts/init_system_docs.go` - ドキュメント初期化
- `Makefile` - ビルドスクリプト
