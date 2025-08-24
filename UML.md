## UML & アーキテクチャ図

このファイルはプロジェクトの主要なアーキテクチャ図をまとめたものです。図ごとに短い日本語の説明と、該当するソースファイルを明記しています。

目次

- シーケンス図（予測リクエスト）
- シーケンス図（AI対話ワークフロー）
- コンポーネント図（高レベル構成）
- データフロー図（DFD レベル1）
- デプロイ図（開発環境と外部サービス）

---

### シーケンス図 — 需要予測リクエスト

説明: クライアントが需要予測をリクエストしてから、ForecastService が Azure OpenAI に頼んで予測を生成し、結果を永続化して返すまでの流れです。

主な参照ファイル: `cmd/server/main.go`, `internal/handlers/demand_forecast_handler.go`, `internal/service/forecast.go`, `pkg/azure/openai.go`

```mermaid
sequenceDiagram
    participant Client
  participant Gin as "Gin (cmd/server/main.go)"
  participant Handler as "demand_forecast_handler.go<br/>(internal/handlers)"
  participant Service as "ForecastService<br/>(internal/service)"
  participant Weather as "WeatherService<br/>(weather_handler.go)"
  participant AzureClient as "pkg/azure/openai.go"
    participant Azure as "Azure OpenAI"
    participant DB as "Storage/DB"

    Client->>Gin: POST /api/v1/forecast (payload)
    Gin->>Handler: route -> DemandForecastHandler
    Handler->>Service: validate + build request
    Service->>Weather: fetch historical & weather data
    Weather-->>Service: data
    Service->>AzureClient: PredictDemand(payload + data)
    AzureClient->>Azure: OpenAI Predict request
    Azure-->>AzureClient: prediction + explanation
    AzureClient-->>Service: prediction result
    Service->>DB: persist forecast, logs
    Service-->>Handler: response DTO
    Handler-->>Gin: JSON response
    Gin-->>Client: 200 OK (forecast)

```

### シーケンス図 — AI 対話（暗黙知取得／調整）

説明: 既存の予測に対して AI とユーザーの対話を行い、暗黙知を取り込んで予測を再計算・保存する流れです。対話は会話履歴を保存しながら行います。

主な参照ファイル: `internal/handlers/ai_handler.go`, `pkg/azure/openai.go`, 永続化コード（DB周り）

```mermaid
sequenceDiagram
    participant User
    participant UI
  participant Gin as "Gin (cmd/server/main.go)"
  participant AIHandler as "ai_handler.go<br/>(internal/handlers)"
    participant DB as "Storage/DB"
    participant AzureClient as "pkg/azure/openai.go"
    participant Azure as "Azure OpenAI"

    User->>UI: start conversation for forecast id
    UI->>Gin: POST /api/v1/ai/converse (start)
    Gin->>AIHandler: handle start
    AIHandler->>DB: load forecast + context (historical, metadata)
    AIHandler->>AzureClient: ChatCompletion(prompt with context)
    AzureClient->>Azure: chat request
    Azure-->>AzureClient: messages (factors/questions)
    AzureClient-->>AIHandler: chat messages
    AIHandler-->>UI: show AI questions/suggestions
    User->>UI: reply (暗黙知を入力)
    UI->>Gin: POST /api/v1/ai/converse (user reply)
    Gin->>AIHandler: append user message
    AIHandler->>AzureClient: ExplainPrediction / PredictDemand with updated context
    AzureClient->>Azure: updated request
    Azure-->>AzureClient: updated prediction + explanation
    AzureClient->>AIHandler: results
    AIHandler->>DB: save conversation + adjusted forecast
    AIHandler-->>UI: updated forecast + explanation
    UI-->>User: show final forecast
```

### コンポーネント図 — 高レベルの責務分離

説明: サービス間の責務と主要モジュールを示します。開発者がどのファイルに関心を持てばよいか分かるようにしました。

主な参照ファイル: `cmd/server/main.go`, `internal/handlers/*`, `internal/service/*`, `pkg/azure/openai.go`, `configs/config.go`, `internal/models/types.go`

```mermaid
graph LR
  Client["Client / UI"]
  Gin["API (Go + Gin)<br/>cmd/server/main.go"]
  Handlers["Handlers<br/>internal/handlers"]
  Services["Services<br/>internal/service"]
  WeatherSvc["WeatherService<br/>internal/handlers/weather_handler.go"]
  ForecastSvc["ForecastService<br/>internal/service/forecast.go"]
  AzureClient["pkg/azure/openai.go<br/>Azure wrapper"]
  DB[("Database / Storage")]
  Config["configs/config.go"]
  Models["internal/models/types.go"]

  Client -->|HTTP| Gin
  Gin --> Handlers
  Handlers --> Services
  Services --> ForecastSvc
  Services --> WeatherSvc
  Services --> AzureClient
  AzureClient -->|API| Azure["Azure OpenAI"]
  Services --> DB
  Gin --> Config
  Services --> Models
```

### データフロー図（DFD レベル1）

説明: 外部データ（販売実績、天気）を取り込んで前処理し、予測モデル（Azure を利用）へ流し、UI と対話を通じてフィードバックを戻す高レベルなデータフローです。

主な参照: データ取り込み・前処理ロジック、`pkg/azure/openai.go` の予測呼び出し

```mermaid
flowchart TD
  A["Historical Sales CSV / ERP"] -->|ingest| Ingest["Ingestion"]
  B["Weather API / External"] -->|fetch| Ingest
  Ingest --> Preproc["Preprocessing & Feature Engineering"]
  Preproc --> ForecastModel["PredictDemand (Azure OpenAI wrapper)"]
  ForecastModel --> Persist["Persist forecast + logs"]
  Persist --> UI["UI shows forecast"]
  UI -->|user implicit knowledge| Conversation["AI Conversation"]
  Conversation --> ForecastModel
  ForecastModel --> FeedbackStorage["Store adjustments & conversation"]
```

### デプロイ図 — 開発環境と外部サービス

説明: ローカル開発と外部依存（DB、Azure）の関係を示します。CI/CD パイプラインについての注記も付けています。

```mermaid
graph TD
  DevMachine["Developer Mac"]
  DevMachine -->|run| App["Go app (Gin)"]
  App -->|connects| DBServer[("Postgres / Storage")]
  App -->|calls| Azure["Azure OpenAI (managed)"]
  App -->|reads| ConfigFile["configs (env)"]
  UI["Web client"] -->|HTTP| App
  Note["Optional: CI/CD -> Docker image -> Azure App Service / AKS"] --- App
```

---

付録・使いかた

- 図は Mermaid をサポートする Markdown ビューア（VS Code プレビュー等）で確認してください。
- 追加で PlantUML 版や PNG/SVG の出力が必要なら教えてください。

