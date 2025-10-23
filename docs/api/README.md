# 📡 API ドキュメント

HUNTシステムのAPI仕様書です。

## ドキュメント一覧

### [API_MANUAL.md](./API_MANUAL.md)
**API仕様書**

すべてのAPIエンドポイント、リクエスト/レスポンス形式、認証方法、エラーハンドリングについて記載されています。

**主な内容:**
- エンドポイント一覧
- リクエスト形式（JSON、FormData）
- レスポンス形式
- エラーコード
- 認証（X-API-KEY）
- 使用例（curl、JavaScript）

**主要なエンドポイント:**
- `POST /api/v1/ai/analyze-file` - ファイル分析
- `POST /api/v1/ai/chat-input` - AIチャット
- `POST /api/v1/ai/predict-sales` - 売上予測
- `POST /api/v1/ai/anomaly-response-with-followup` - 異常回答+深掘り

---

[← ドキュメントTOPへ](../README.md)
