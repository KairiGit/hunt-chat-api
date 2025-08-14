# HUNT chat-api

Hidden Understanding & kNowledge Transfer

経験豊富な担当者の暗黙知と AI のデータ分析能力を融合させ、製造業における需要予測の精度と効率を向上させるためのチャット API です。



## 🚀 開発環境

### Dev Container で開発する場合（推奨）

1. **前提条件**:

   - Visual Studio Code
   - Docker Desktop
   - Dev Containers extension for VS Code

2. **開発環境の起動**:

   ```bash
   # リポジトリをクローン
   git clone <repository-url>
   cd hunt-chat-api

   # VS Codeで開く
   code .

   # F1キーを押して "Dev Containers: Reopen in Container" を選択
   ```

3. **開発開始**:
   ```bash
   make dev    # ライブリロード付きで起動
   make run    # 通常起動
   make test   # テスト実行
   ```

### ローカル環境で開発する場合

1. **前提条件**:

   - Go 1.21 以上
   - Azure CLI
   - Azure Developer CLI

2. **セットアップ**:

   ```bash
   # 依存関係のインストール
   go mod tidy

   # 環境変数の設定
   cp .env.example .env
   # .envファイルを編集
   ```

3. **起動**:
   ```bash
   go run cmd/server/main.go
   ```

## 🔧 設定

### 環境変数

`.env`ファイルを作成し、以下の変数を設定してください：

```env
PORT=8080
ENVIRONMENT=development
AZURE_OPENAI_ENDPOINT=https://your-openai-resource.openai.azure.com/
AZURE_OPENAI_MODEL=gpt-4
```

### Azure 認証

```bash
# Azure CLIでログイン
az login

# Azure Developer CLI でログイン
azd auth login
```

## 🎯 API エンドポイント

### ヘルスチェック

```http
GET /health
```

### Hello API

```http
GET /api/v1/hello
```

### チャット API（予定）

```http
POST /api/v1/chat
Content-Type: application/json

{
  "message": "需要予測について教えて",
  "context": "製造業の月次売上データ"
}
```

## 📋 利用可能なコマンド

### 開発用

```bash
make dev      # ライブリロード付きで起動
make run      # 通常起動
make build    # ビルド
make test     # テスト実行
make check    # 全チェック実行（fmt, vet, lint, test）
```

### Azure 用

```bash
make azure-login  # Azureログイン
make azure-init   # Azureリソース初期化
make deploy       # Azureにデプロイ
```

### Docker 用

```bash
make docker-build  # Dockerイメージビルド
make docker-run    # Dockerコンテナ実行
```

## 🏗️ アーキテクチャ

```
hunt-chat-api/
├── cmd/server/           # メインアプリケーション
├── internal/
│   ├── handlers/         # HTTPハンドラー
│   ├── services/         # ビジネスロジック
│   └── models/           # データモデル
├── pkg/azure/            # Azure SDK ラッパー
├── configs/              # 設定管理
└── .devcontainer/        # Dev Container設定
```

## 🔄 開発フロー

1. **機能開発**: `internal/` 以下で実装
2. **テスト**: `make test` でテスト実行
3. **コード品質**: `make check` で品質チェック
4. **ローカル確認**: `make dev` でライブリロード
5. **デプロイ**: `make deploy` で Azure へデプロイ

## 📚 技術スタック

- **言語**: Go 1.21
- **Web フレームワーク**: Gin
- **AI**: Azure OpenAI Service
- **認証**: Azure Identity
- **コンテナ**: Docker
- **開発環境**: VS Code Dev Containers

## 🤝 コントリビューション

1. Dev Container で開発環境を起動
2. 機能ブランチを作成
3. コードを実装
4. `make check` でテスト・品質チェック
5. プルリクエストを作成

## 📝 ライセンス

このプロジェクトは [MIT License](LICENSE) の下で公開されています。

## 🌐 How to use this API

### API基本情報

- **ベースURL**: `https://api.hunt-chat.example.com`（または実際のデプロイ先URL）
- **APIバージョン**: v1
- **データ形式**: JSON
- **認証方式**: Bearer Token（Azureの認証情報）

### APIエンドポイント一覧

#### 🔍 健康状態確認

```http
GET /health
```

**レスポンス例**:
```json
{
  "status": "healthy",
  "service": "HUNT Chat-API"
}
```

#### 👋 Hello API（疎通確認用）

```http
GET /api/v1/hello
```

**レスポンス例**:
```json
{
  "message": "Hello from HUNT Chat-API!"
}
```

#### 🌤️ 気象データAPI

##### 地域コード一覧取得

```http
GET /api/v1/weather/regions
```

##### 気象予報データ取得

```http
GET /api/v1/weather/forecast/{regionCode}
```

**パラメータ**:
- `regionCode`: 地域コード（例: `130000`は東京都）。省略時は東京都が使用されます。

**レスポンス例**:
```json
{
  "success": true,
  "data": [
    {
      "date": "2025-08-01",
      "weather": "晴れ",
      "temperature": {
        "max": 32,
        "min": 24
      }
    }
  ],
  "count": 1
}
```

##### 過去の気象データ取得

```http
GET /api/v1/weather/historical/{regionCode}
```

**パラメータ**:
- `regionCode`: 地域コード。省略時は東京都が使用されます。

#### 📊 需要予測API

##### 需要予測実行

```http
POST /api/v1/demand/forecast
Content-Type: application/json

{
  "region_code": "240000",
  "product_category": "飲料",
  "forecast_days": 7,
  "historical_days": 30,
  "tacit_knowledge": [
    {
      "type": "seasonal",
      "description": "夏季は冷たい飲料の需要が増加",
      "weight": 0.3,
      "condition": "hot_day"
    }
  ]
}
```

**パラメータ**:
- `region_code`: 地域コード（省略可、デフォルト: `240000`（三重県））
- `product_category`: 製品カテゴリ（省略可、デフォルト: `飲料`）
- `forecast_days`: 予測日数（省略可、デフォルト: `7`）
- `historical_days`: 過去データ参照日数（省略可、デフォルト: `30`）
- `tacit_knowledge`: 暗黙知データ（省略可）

**レスポンス例**:
```json
{
  "success": true,
  "data": {
    "region_code": "240000",
    "product_category": "飲料",
    "forecast_period": "2025-08-01 to 2025-08-07",
    "daily_forecast": [
      {
        "date": "2025-08-01",
        "demand_index": 120,
        "confidence": 0.85,
        "factors": [
          {
            "name": "temperature",
            "impact": 0.6
          },
          {
            "name": "day_of_week",
            "impact": 0.2
          }
        ]
      }
    ]
  }
}
```

##### 簡易需要予測（三重県鈴鹿市）

```http
GET /api/v1/demand/forecast/suzuka?product_category=飲料&forecast_days=7
```

**クエリパラメータ**:
- `product_category`: 製品カテゴリ（省略可、デフォルト: `飲料`）
- `forecast_days`: 予測日数（省略可、デフォルト: `7`）
- `historical_days`: 過去データ参照日数（省略可、デフォルト: `30`）

##### 需要異常検知

```http
GET /api/v1/demand/anomalies
```

販売実績と気象データを分析し、需要の異常値を検出します。AIがユーザーに対話を開始するきっかけとして利用されます。

**クエリパラメータ**:
- `region_code`: 地域コード（省略可、デフォルト: `240000`（三重県））
- `days`: 分析対象の日数（省略可、デフォルト: `30`）

**レスポンス例**:
```json
{
  "success": true,
  "data": [
    {
      "date": "2025-08-08",
      "product_id": "P001",
      "description": "猛暑日（32.5℃）に売上が平均（110個）を大幅に上回りました（300個）。",
      "impact_score": 2.95,
      "trigger": "weather_sales_high",
      "weather": "晴れ",
      "temperature": 32.5
    }
  ],
  "count": 1
}
```

#### 🧠 AI統合API

##### 気象データAI分析

```http
POST /api/v1/ai/analyze-weather
Content-Type: application/json

{
  "region_code": "240000",
  "days": 30
}
```

**パラメータ**:
- `region_code`: 地域コード（省略可、デフォルト: `240000`（三重県））
- `days`: 分析対象日数（省略可、デフォルト: `30`）

##### 需要予測AI説明

```http
POST /api/v1/ai/explain-forecast
Content-Type: application/json

{
  "forecast_data": { /* 需要予測データ */ },
  "detail_level": "detailed"
}
```

##### 異常検知に基づく質問生成

```http
GET /api/v1/ai/generate-question
```

システムが自動で検知した需要の異常データに基づき、原因究明のための質問をAIが生成します。

**クエリパラメータ**:
- `region_code`: 地域コード（省略可、デフォルト: `240000`（三重県））
- `days`: 分析対象の日数（省略可、デフォルト: `30`）

**レスポンス例**:
```json
{
  "success": true,
  "message": "異常を検知し、質問を生成しました。",
  "question": "8月8日のミネラルウォーターの売上が特に高かったようですが、この日は何か特別な販促活動やイベントがありましたか？",
  "source_anomaly": {
      "date": "2025-08-08",
      "product_id": "P001",
      "description": "猛暑日（32.5℃）に売上が平均（110個）を大幅に上回りました（300個）。",
      "impact_score": 2.95,
      "trigger": "weather_sales_high",
      "weather": "晴れ",
      "temperature": 32.5
    }
}
```

### APIの利用例（curl）

#### ヘルスチェック

```bash
curl -X GET "https://api.hunt-chat.example.com/health"
```

```bash
curl http://localhost:8080/api/your-endpoint # ローカルでの確認
```
#### 需要予測取得

```bash
curl -X POST "https://api.hunt-chat.example.com/api/v1/demand/forecast" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "region_code": "240000",
    "product_category": "飲料",
    "forecast_days": 7
  }'
```

### エラーレスポンス

APIはエラー時に適切なHTTPステータスコードと共に以下の形式のレスポンスを返します：

```json
{
  "error": "エラーメッセージ"
}
```

**一般的なHTTPステータスコード**:
- `200 OK`: リクエスト成功
- `400 Bad Request`: リクエスト形式が不正
- `401 Unauthorized`: 認証エラー
- `404 Not Found`: リソースが見つからない
- `500 Internal Server Error`: サーバー内部エラー

### API利用時の注意点

1. **認証情報の管理**: APIキーやトークンは安全に管理し、公開リポジトリにコミットしないでください
2. **リクエスト制限**: 大量のリクエストを短時間に送信しないでください
3. **データキャッシュ**: 頻繁に変更されないデータはクライアント側でキャッシュすることを推奨します
4. **エラーハンドリング**: 適切なエラーハンドリングを実装し、リトライロジックを考慮してください
