# HUNT Chat-API 利用マニュアル

`v2.0.0` - 2025-10-16

## 1. はじめに

このドキュメントは、HUNT Chat-APIの利用方法について説明します。
本APIは、販売実績や気象データなどの外部データを取り込み、AIによる分析、チャット対話、需要予測機能を提供することを目的としています。

- **本番環境ベースURL**: `https://hunt-chat-api.vercel.app`
- **プレビュー環境ベースURL**: 各デプロイごとに発行される一時的なURL

## 2. 認証

本APIのすべてのエンドポイント（`/api/v1/...`）は、APIキーによる認証で保護されています。
APIを呼び出す際は、HTTPリクエストのヘッダーにAPIキーを含める必要があります。

- **ヘッダー名**: `X-API-KEY`
- **値**: 事前に発行されたAPIキー

```sh
curl -X POST \
  -H "X-API-KEY: YOUR_SECRET_API_KEY" \
  https://hunt-chat-api.vercel.app/api/v1/...
```
```sh
curl -X POST -H "Content-Type: application/json" -H "X-API-KEY: 
     YOUR_SECRET_API_KEY" -d '{"chat_message": "こんにちは"}'
     'https://hunt-chat-api.vercel.app/api/v1/ai/chat-input'
```

正しいAPIキーが提供されない場合、サーバーは `401 Unauthorized` エラーを返します。

## 3. データ保管のアーキテクチャ

本APIは、データの種類に応じて2つのデータベースを使い分けます。

### 3.1. 構造化データ（生データ）

- **対象**: 販売実績、製品マスター、顧客マスターなど、テーブル形式の元データ。
- **保管場所**: **APIの呼び出し元である業務アプリケーション側のRDB（PostgreSQLなど）**で管理されることを前提とします。
- **本APIの役割**: 本APIは、これらの生データを**直接保管しません**。API経由で一時的に受け取り、分析・要約処理に利用します。

### 3.2. 非構造化データ（意味・文脈）

- **対象**: AIが文脈を理解するためのテキスト情報。
  - ファイルから抽出・要約された「分析レポート」
  - AIとの「チャットの対話履歴」
  - 異常検知への「ユーザーの回答」
- **保管場所**: 本APIに接続された**Qdrant（ベクトルデータベース）**に永続化します。
- **本APIの役割**: テキスト情報をベクトル化し、Qdrantに保存します。これにより、AIは過去の対話や分析結果を「記憶」し、意味的に類似した情報を検索して、より高度な回答を生成できます。

## 4. APIリファレンス

### 4.1. コア機能API

#### ファイル分析とレポート生成

- **エンドポイント**: `POST /api/v1/ai/analyze-file`
- **概要**: アップロードされたファイル（Excel, CSV）を分析し、統計的な相関分析、回帰分析、異常検知を含む詳細な「分析レポート」を生成します。生成されたレポートはベクトルデータベースに自動で保存されます。
- **リクエスト形式**: `multipart/form-data`
  - `file`: 分析対象のファイル（CSV または Excel）
  
**必須列:**
  - **日付列**: `date` または `日付` （形式: `YYYY-MM-DD`, `YYYY/M/D`, `YYYY/MM/DD`）
  - **製品ID列（必須）**: `product_code`, `product_id`, `product_ID`, `製品ID`, `製品id`, `製品コード`, `商品ID`, `商品id`, `商品コード` のいずれか
  
**推奨列:**
  - **製品名列（推奨）**: `product`, `product_name`, `製品`, `製品名`, `商品`, `商品名` のいずれか
    - AIの回答や異常通知で表示されます
    - 製品名がない場合は製品IDが表示に使用されます
    
**必須列:**
  - **販売数列**: `sales`, `quantity`, `販売数`, `数量` のいずれか

**注意**: その他の列（単価、売上金額、曜日、月、年など）は含めてもOKですが、現在は使用されません。

詳細は [FILE_FORMAT_GUIDE.md](./FILE_FORMAT_GUIDE.md) を参照してください。

- **レスポンス形式**: `200 OK`
  ```json
  {
    "success": true,
    "summary": "ファイルの基本的な要約テキスト...",
    "analysis_report": {
      "report_id": "uuid-v4-string",
      "file_name": "sales_data.xlsx",
      "analysis_date": "2025-10-16T10:00:00Z",
      "summary": "AIによる分析サマリー...",
      "correlations": [
        { "factor": "気温", "correlation_coef": 0.85, "interpretation": "強い正の相関" }
      ],
      "anomalies": [
        { "date": "2025-09-20", "actual_value": 500, "expected_value": 250, "severity": "high", "ai_question": "この日の売上急増の要因は何ですか？" }
      ],
      "recommendations": ["気温が高い日の在庫を増やすことを推奨します。"]
    }
  }
  ```

#### AIチャット（RAG対応）

- **エンドポイント**: `POST /api/v1/ai/chat-input`
- **概要**: ユーザーの質問に対し、過去のチャット履歴、ファイル分析レポート、保存されたナレッジを自動で検索（RAG）し、文脈に基づいた応答を生成します。
- **リクエスト形式**: `application/json`
  ```json
  {
    "chat_message": "先月の分析レポートについて教えて。",
    "context": "（オプション：追加のコンテキスト）",
    "session_id": "（オプション：会話を継続するためのセッションID）",
    "user_id": "（オプション：ユーザーごとの履歴管理ID）"
  }
  ```
- **レスポンス形式**: `200 OK`
  ```json
  {
    "success": true,
    "response": {
      "text": "AIによる応答テキスト...",
      "session_id": "new-or-existing-session-id",
      "context_sources": ["分析レポート (sales_data.xlsx)", "過去の会話 (2025-10-15)"]
    }
  }
  ```

---

### 4.2. 異常検知と学習API

#### 売上データの異常を検知

- **エンドポイント**: `POST /api/v1/ai/detect-anomalies`
- **概要**: 提供された時系列の売上データから、統計的に異常な点を検出し、AIによる質問を生成します。
- **リクエスト形式**: `application/json`
  ```json
  {
    "sales": [110, 120, 500, 130],
    "dates": ["2025-09-19", "2025-09-20", "2025-09-21", "2025-09-22"]
  }
  ```
- **レスポンス形式**: `200 OK`
  ```json
  {
    "success": true,
    "anomalies": [
      { "date": "2025-09-21", "actual_value": 500, "expected_value": 125, "severity": "critical", "ai_question": "2025-09-21の売上が予期せず急増した原因は何ですか？特別なイベントがありましたか？" }
    ]
  }
  ```

#### 異常への回答を保存

- **エンドポイント**: `POST /api/v1/ai/anomaly-response`
- **概要**: AIが生成した質問に対し、ユーザーが提供した回答（原因や背景）をベクトルデータベースに保存します。これはAIの継続的な学習データとなります。
- **リクエスト形式**: `application/json`
  ```json
  {
    "anomaly_date": "2025-09-21",
    "product_id": "P001",
    "question": "この日の売上急増の要因は何ですか？",
    "answer": "地域限定の特別セールを実施したため。",
    "tags": ["セール", "キャンペーン"],
    "impact": "positive",
    "impact_value": 250.5
  }
  ```
- **レスポンス形式**: `200 OK`
  ```json
  {
    "success": true,
    "response_id": "new-uuid-v4-string",
    "message": "回答を保存しました。AIが学習データとして活用します。"
  }
  ```

#### 回答履歴を取得

- **エンドポイント**: `GET /api/v1/ai/anomaly-responses`
- **概要**: 保存された異常への回答履歴一覧を取得します。
- **クエリパラメータ**: `product_id` (オプション), `limit` (オプション)
- **レスポンス形式**: `200 OK`
  ```json
  {
    "success": true,
    "responses": [
      { "response_id": "uuid", "anomaly_date": "2025-09-21", "product_id": "P001", "tags": ["セール"] }
    ]
  }
  ```

#### AIの学習による洞察を取得

- **エンドポイント**: `GET /api/v1/ai/learning-insights`
- **概要**: 保存された回答履歴からAIが学習したパターンや洞察（例：「セールを行うと平均30%売上が増加する」など）を取得します。
- **クエリパラメータ**: `category` (オプション, e.g., "セール")
- **レスポンス形式**: `200 OK`
  ```json
  {
    "success": true,
    "insights": [
      {
        "category": "セール",
        "pattern": "「セール」が発生した際、平均+150.0%の需要増加の傾向があります（5件の実績から学習）",
        "average_impact": 150.0,
        "confidence": 0.5,
        "learned_from": 5
      }
    ]
  }
  ```

---

### 4.3. 分析レポートAPI

#### 分析レポートの一覧を取得

- **エンドポイント**: `GET /api/v1/ai/analysis-reports`
- **概要**: 保存されているすべての分析レポートのヘッダー情報（ID、ファイル名、分析日）を取得します。
- **レスポンス形式**: `200 OK`
  ```json
  {
    "success": true,
    "reports": [
      { "report_id": "uuid-1", "file_name": "sales_2025_q3.xlsx", "analysis_date": "2025-10-01" },
      { "report_id": "uuid-2", "file_name": "sales_2025_q2.xlsx", "analysis_date": "2025-07-01" }
    ]
  }
  ```

#### 個別の分析レポートを取得

- **エンドポイント**: `GET /api/v1/ai/analysis-report`
- **概要**: 指定したIDの分析レポートの詳細を取得します。
- **クエリパラメータ**: `id` (必須)
- **レスポンス形式**: `200 OK` (内容は`/analyze-file`の`analysis_report`オブジェクトと同様)

#### 分析レポートを削除

- **エンドポイント**: `DELETE /api/v1/ai/analysis-report`
- **概要**: 指定したIDの分析レポートを削除します。
- **クエリパラメータ**: `id` (必須)
- **レスポンス形式**: `200 OK`
  ```json
  {
    "success": true,
    "message": "レポートが正常に削除されました"
  }
  ```

---

### 4.4. その他のAPI

このセクションには、気象データや需要予測など、より専門的な機能が含まれます。詳細は別途ドキュメントで提供されます。

- `GET /api/v1/weather/...`
- `GET /api/v1/demand/...`
