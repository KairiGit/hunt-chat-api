# Handlers パッケージ構成

## ファイル分割について (2025-10-21)

元々 `ai_handler.go` は 2304行の巨大なファイルでしたが、保守性向上のため以下のように分割しました：

### 分割後の構成

1. **ai_handler.go** (1,460行)
   - AIHandler構造体の定義
   - コンストラクタ
   - 共通ヘルパー関数 (findIndex, getStringFromPayload, etc.)
   - 共通型定義 (ChatInputRequest, AnalysisProgress, etc.)
   - その他の小規模なハンドラー関数群

2. **file_analysis.go** (654行)
   - `AnalyzeFile()` メソッド - メインのファイル分析ロジック
   - CSV/Excel解析
   - 統計分析
   - 異常検知
   - 非同期AI処理

3. **chat_handler.go** (229行)
   - `ChatInput()` メソッド - RAGベースのAIチャット
   - チャット履歴検索
   - コンテキスト収集
   - AI応答生成

### その他の既存ファイル

- `chat_handler.go` (既存) - 元々存在していたチャット関連
- `demand_forecast_handler.go` - 需要予測関連
- `weather_handler.go` - 気象データ関連
- `handlers_test.go` - テストコード

### バックアップ

- `ai_handler.go.backup` - 分割前の元ファイル (2,304行)

### 分割の効果

- ✅ ファイルサイズ: 2,304行 → 最大 1,460行に削減
- ✅ 機能ごとに責務が分離
- ✅ 可読性・保守性が向上
- ✅ ビルド成功確認済み
- ✅ 同じpackage内なので、既存のimportやルーティングに影響なし

### 注意事項

- すべてのファイルは同じ `package handlers` に属しています
- 型定義とヘルパー関数は `ai_handler.go` で共有されています
- メソッドはレシーバー `*AIHandler` を通じて他のファイルから自動的にアクセス可能です
