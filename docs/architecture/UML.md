## UML & アーキテクチャ図 (v3.0)

このファイルは、現在のプロジェクト実装に基づいた主要なアーキテクチャ図をまとめたものです。図ごとに短い日本語の説明と、該当するソースファイルを明記しています。

**主な変更点 (v3.0 - 2025-10-21):**
- 処理フローを細かく分割し、個別のシーケンス図として整理
- 非同期処理の詳細フローを追加
- データ集約（日次/週次/月次）フローを追加
- 製品別分析（旧：週次分析）フローを追加
- 異常対応チャットフローを追加
- 各図を見やすいサイズに最適化

## 目次

### 1. システム全体構成
- [1-1. コンポーネント図（高レベル構成）](#1-1-コンポーネント図高レベル構成)
- [1-2. デプロイ図（ローカル開発とVercel本番環境）](#1-2-デプロイ図ローカル開発とvercel本番環境)

### 2. ファイル分析フロー
- [2-1. ファイル分析：全体フロー（概要）](#2-1-ファイル分析全体フロー概要)
- [2-2. ファイル分析：同期処理フェーズ](#2-2-ファイル分析同期処理フェーズ)
- [2-3. ファイル分析：非同期AI処理フェーズ](#2-3-ファイル分析非同期ai処理フェーズ)
- [2-4. データ集約処理（粒度選択）](#2-4-データ集約処理粒度選択)

### 3. AI機能フロー
- [3-1. RAGチャットフロー](#3-1-ragチャットフロー)
- [3-2. 異常対応チャットフロー](#3-2-異常対応チャットフロー)
- [3-3. 継続的学習フロー](#3-3-継続的学習フロー)

### 4. 分析機能フロー
- [4-1. 製品別分析フロー](#4-1-製品別分析フロー)
- [4-2. 需要予測フロー](#4-2-需要予測フロー)

---

## 1. システム全体構成

### 1-1. コンポーネント図（高レベル構成）

**説明:** フロントエンド、バックエンド、外部サービス間の責務と主要モジュールを示します。バックエンドは `pkg/` 以下のハンドラ、サービスで構成され、AI機能は `AzureOpenAIService` と `VectorStoreService` (Qdrant) を中心に実現されます。

**主な参照ファイル:** `api/index.go`, `pkg/handlers/*`, `pkg/services/*`, `pkg/models/types.go`

```mermaid
graph TD
  subgraph "Frontend"
    UI["Next.js UI<br/>(React Components)"]
  end

  subgraph "Backend (Go on Vercel/Local)"
    Proxy["Next.js API Proxy<br/>/src/app/api/proxy"]
    Gin["API Gateway (Gin)<br/>/api/index.go"]
    Handlers["Handlers<br/>/pkg/handlers"]
    Services["Services<br/>/pkg/services"]
  end

  subgraph "External Dependencies"
    Azure["Azure OpenAI Service"]
    QdrantDB["Qdrant<br/>(Vector Database)"]
  end

  UI -->|1 API Call| Proxy
  Proxy -->|2 Forward Request| Gin
  Gin -->|3 Route to Handler| Handlers
  Handlers -->|4 Execute Business Logic| Services
  Services -->|5 Call LLM| Azure
  Services -->|6 Read/Write Vectors| QdrantDB
```

---

### 1-2. デプロイ図（ローカル開発とVercel本番環境）

**説明:** ローカルでの開発環境と、Vercelと外部サービスで構成される本番環境の関係を示します。

**主な参照ファイル:** `cmd/server/main.go`, `api/index.go`, `docker-compose.yml`

```mermaid
graph TD
  subgraph "Local Development"
    DevMachine["Developer Machine"]
    DevMachine -->|runs| GoApp["Go App (Gin)<br/>cmd/server/main.go"]
    DevMachine -->|runs| WebUI["Next.js Dev Server"]
    DevMachine -->|runs via Docker| QdrantLocal["Qdrant (Docker)"]
  end

  subgraph "Cloud Environment"
    subgraph Vercel
        NextApp["Next.js Frontend"]
        GoServerless["Go Serverless Func<br/>api/index.go"]
    end
    subgraph "External Dependencies"
        AzureOpenAI["Azure OpenAI Service"]
        QdrantCloud["Qdrant Cloud / Self-hosted"]
    end
  end

  GoApp --> QdrantLocal
  GoApp --> AzureOpenAI
  WebUI --> GoApp

  NextApp --> GoServerless
  GoServerless --> QdrantCloud
  GoServerless --> AzureOpenAI
```

---

## 2. ファイル分析フロー

### 2-1. ファイル分析：全体フロー（概要）

**説明:** ファイルアップロードから分析完了までの全体像を示します。同期処理（即座にレスポンス）と非同期処理（バックグラウンド実行）の2段階構成になっています。

**主な参照ファイル:** `pkg/handlers/ai_handler.go` (AnalyzeFile)

```mermaid
graph TD
    A[ユーザーがファイルアップロード] --> B[同期処理フェーズ<br/>3-5秒]
    B --> C[即座にレスポンス返却<br/>基本分析結果表示]
    B --> D[非同期処理フェーズ<br/>バックグラウンド]
    D --> E[AI分析完了<br/>2-5秒後]
    D --> F[AI質問生成完了<br/>5-10秒後]
    
    style B fill:#90EE90
    style D fill:#FFB6C1
    style C fill:#87CEEB
```

---

### 2-2. ファイル分析：同期処理フェーズ

**説明:** ユーザーにレスポンスを返すまでの同期処理（ステップ1-7）を詳細に示します。この段階で基本的な統計分析と異常検知が完了します。

**主な参照ファイル:** `pkg/handlers/ai_handler.go`, `pkg/services/statistics_service.go`

```mermaid
sequenceDiagram
    participant User
    participant UI as "Next.js UI"
    participant Handler as "AIHandler"
    participant Stats as "StatisticsService"
    participant VectorStore as "VectorStoreService"

    User->>UI: CSVファイルアップロード
    UI->>Handler: POST /api/v1/ai/analyze-file
    
    rect rgb(144, 238, 144)
        note over Handler: ステップ1-2: ファイル処理
        Handler->>Handler: ReadFile (0.1-0.5s)
        Handler->>Handler: ParseCSV (0.2-1s)
    end
    
    rect rgb(144, 238, 144)
        note over Handler,Stats: ステップ3: 統計分析
        Handler->>Stats: CalculateStatistics
        Stats->>Stats: 基本統計量計算
        Stats->>Stats: データ集約（粒度別）
        Stats-->>Handler: 統計結果
    end
    
    rect rgb(144, 238, 144)
        note over Handler,Stats: ステップ5: 異常検知
        Handler->>Stats: DetectAnomalies
        Stats->>Stats: 閾値ベース検知
        Stats->>Stats: 統計的異常検知
        Stats-->>Handler: 異常リスト
    end
    
    rect rgb(144, 238, 144)
        note over Handler,VectorStore: ステップ7: DB保存
        Handler->>VectorStore: StoreAnalysisReport
        VectorStore-->>Handler: 保存完了
    end
    
    Handler-->>UI: JSON Response<br/>ai_insights_pending: true<br/>ai_questions_pending: true
    UI-->>User: 基本分析結果を即座に表示 ✅
```

---

### 2-3. ファイル分析：非同期AI処理フェーズ

**説明:** バックグラウンドで実行されるAI分析とAI質問生成の処理を示します。これらは並列で実行され、完了後にDBに保存されます（TODO実装）。

**主な参照ファイル:** `pkg/handlers/ai_handler.go`, `pkg/services/azure_openai_service.go`

```mermaid
sequenceDiagram
    participant Handler as "AIHandler"
    participant Goroutine1 as "Goroutine: AI分析"
    participant Goroutine2 as "Goroutine: AI質問生成"
    participant Azure as "AzureOpenAI"
    participant VectorStore as "VectorStore (TODO)"

    rect rgb(255, 182, 193)
        note over Handler,Goroutine1: ステップ4: AI分析（非同期）
        Handler->>Goroutine1: go func() 起動
        Goroutine1->>Azure: ProcessChatWithContext
        note right of Goroutine1: 統計データを基に<br/>AI洞察を生成
        Azure-->>Goroutine1: AI分析結果 (2-5s)
        note right of Goroutine1: TODO: UpdateAnalysisReport()<br/>DBに保存して完了通知
    end
    
    rect rgb(255, 182, 193)
        note over Handler,Goroutine2: ステップ6: AI質問生成（非同期）
        Handler->>Goroutine2: go func() 起動
        
        loop 各異常に対して並列実行
            Goroutine2->>Azure: GenerateAIQuestion
            note right of Goroutine2: 異常の原因を<br/>特定する質問を生成
            Azure-->>Goroutine2: 質問と選択肢
        end
        
        note right of Goroutine2: TODO: UpdateAnomalyQuestions()<br/>DBに保存して完了通知
    end
```

---

### 2-4. データ集約処理（粒度選択）

**説明:** ユーザーが選択した粒度（日次/週次/月次）に応じて、データを集約する処理を示します。

**主な参照ファイル:** `pkg/services/statistics_service.go`

```mermaid
sequenceDiagram
    participant UI as "UI (粒度選択)"
    participant Handler as "AIHandler"
    participant Stats as "StatisticsService"

    UI->>Handler: granularity: "daily/weekly/monthly"
    
    alt 日次集約 (daily)
        Handler->>Stats: AggregateDaily()
        Stats->>Stats: 日ごとにグループ化
        Stats->>Stats: 日次統計計算
        Stats-->>Handler: 日次データ配列
    else 週次集約 (weekly) - デフォルト
        Handler->>Stats: AggregateWeekly()
        Stats->>Stats: 週ごとにグループ化<br/>(月曜始まり)
        Stats->>Stats: 週次統計計算
        Stats->>Stats: 前週比計算
        Stats-->>Handler: 週次データ配列
    else 月次集約 (monthly)
        Handler->>Stats: AggregateMonthly()
        Stats->>Stats: 月ごとにグループ化
        Stats->>Stats: 月次統計計算
        Stats->>Stats: 前月比計算
        Stats-->>Handler: 月次データ配列
    end
    
    Handler-->>UI: 集約済みデータ + 統計サマリー
```

---

## 3. AI機能フロー

### 3-1. RAGチャットフロー

**説明:** ユーザーからのチャット入力に対し、Qdrantから関連情報を検索（RAG）してコンテキストを構築し、AIが応答を生成するまでの一連の流れを示します。

**主な参照ファイル:** `pkg/handlers/ai_handler.go` (ChatInput), `pkg/services/vector_store_service.go`

```mermaid
sequenceDiagram
    participant User
    participant UI as "Next.js UI"
    participant Handler as "AIHandler"
    participant VectorStore as "VectorStoreService"
    participant Azure as "AzureOpenAI"

    User->>UI: チャットメッセージ入力
    UI->>Handler: POST /api/v1/ai/chat-input
    
    rect rgb(230, 230, 250)
        note over Handler,VectorStore: RAGコンテキスト収集
        Handler->>VectorStore: SearchChatHistory(query)
        VectorStore-->>Handler: 関連する過去の会話
        
        Handler->>VectorStore: SearchAnalysisReports(query)
        VectorStore-->>Handler: 関連する分析レポート
        
        Handler->>VectorStore: SearchSystemDocuments(query)
        VectorStore-->>Handler: 関連するシステムドキュメント
    end
    
    rect rgb(255, 228, 196)
        note over Handler,Azure: AI応答生成
        Handler->>Handler: コンテキスト統合<br/>+ システムプロンプト
        Handler->>Azure: ProcessChatWithHistory
        Azure-->>Handler: AI応答
    end
    
    rect rgb(230, 230, 250)
        note over Handler,VectorStore: 会話履歴保存
        Handler->>VectorStore: StoreChatMessage(user + ai)
        VectorStore-->>Handler: 保存完了
    end
    
    Handler-->>UI: JSON Response<br/>+ context_sources
    UI-->>User: AI応答 + 参照元表示
```

---

### 3-2. 異常対応チャットフロー

**説明:** 検出された異常に対してユーザーが原因を回答し、それを学習データとして保存するフローを示します。

**主な参照ファイル:** `pkg/handlers/ai_handler.go` (SaveAnomalyResponse), `src/app/anomaly-response/page.tsx`

```mermaid
sequenceDiagram
    participant User
    participant UI as "異常対応ページ"
    participant Handler as "AIHandler"
    participant VectorStore as "VectorStoreService"
    participant Azure as "AzureOpenAI"

    rect rgb(255, 250, 205)
        note over UI,Handler: 未回答異常の取得
        UI->>Handler: GET /api/v1/ai/unanswered-anomalies
        Handler->>VectorStore: GetUnansweredAnomalies()
        VectorStore-->>Handler: 異常リスト
        Handler-->>UI: 未回答異常一覧
    end
    
    User->>UI: 異常を選択
    UI->>UI: 異常の詳細表示<br/>+ AI生成質問
    
    rect rgb(255, 182, 193)
        note over UI,Azure: 対話的な原因特定
        User->>UI: 初回回答入力
        UI->>Handler: POST /api/v1/ai/anomaly-response
        Handler->>Azure: ProcessAnomalyResponse<br/>(質問+回答+コンテキスト)
        Azure-->>Handler: 追加質問 or 原因特定
        Handler-->>UI: AI応答
        
        opt 追加質問がある場合
            User->>UI: 追加回答入力
            UI->>Handler: POST /api/v1/ai/anomaly-response-followup
            Handler->>Azure: ProcessFollowupResponse
            Azure-->>Handler: さらなる質問 or 最終原因
            Handler-->>UI: AI応答
        end
    end
    
    rect rgb(144, 238, 144)
        note over Handler,VectorStore: 学習データとして保存
        Handler->>VectorStore: StoreAnomalyResponse<br/>(anomaly_responses collection)
        VectorStore-->>Handler: 保存完了
        Handler-->>UI: 保存完了メッセージ
    end
    
    UI-->>User: 「回答を保存しました」表示
```

---

### 3-3. 継続的学習フロー

**説明:** システム全体での継続的学習サイクルを示します。ユーザーの回答や分析結果がQdrantに蓄積され、将来のAI応答に活用されます。

**主な参照ファイル:** `pkg/services/vector_store_service.go`

```mermaid
graph TD
    A[新規ファイル分析] --> B[分析レポート生成]
    B --> C[Qdrant: analysis_reports]
    
    D[異常検知] --> E[ユーザー回答]
    E --> F[Qdrant: anomaly_responses]
    
    G[チャット会話] --> H[Qdrant: chat_history]
    
    I[システムドキュメント] --> J[Qdrant: system_documents]
    
    C --> K[RAG検索]
    F --> K
    H --> K
    J --> K
    
    K --> L[コンテキスト構築]
    L --> M[AI応答生成]
    M --> N[より精度の高い回答]
    
    N --> O[新たな会話データ]
    O --> H
    
    style C fill:#FFE4B5
    style F fill:#FFB6C1
    style H fill:#E0BBE4
    style J fill:#98D8C8
    style K fill:#87CEEB
    style M fill:#90EE90
```

---

## 4. 分析機能フロー

### 4-1. 製品別分析フロー

**説明:** 特定の製品について、日別/週別/月別の粒度で販売実績を分析するフロー（旧：週次分析ページ）。

**主な参照ファイル:** `pkg/handlers/ai_handler.go` (AnalyzeWeekly), `src/app/product-analysis/page.tsx`

```mermaid
sequenceDiagram
    participant User
    participant UI as "製品別分析ページ"
    participant Handler as "AIHandler"
    participant Stats as "StatisticsService"
    participant Azure as "AzureOpenAI"

    User->>UI: 製品選択 + 期間指定<br/>+ 粒度選択
    UI->>Handler: POST /api/v1/ai/analyze-weekly
    
    rect rgb(230, 230, 250)
        note over Handler,Stats: 期間別データ集約
        Handler->>Stats: AggregateByGranularity<br/>(daily/weekly/monthly)
        Stats->>Stats: 期間ごとに集計
        Stats->>Stats: 前期間比計算
        Stats->>Stats: トレンド分析
        Stats-->>Handler: 集約データ + 統計
    end
    
    rect rgb(255, 228, 196)
        note over Handler,Azure: AI推奨事項生成
        Handler->>Azure: GenerateRecommendations<br/>(統計+トレンド)
        Azure-->>Handler: 具体的な推奨アクション
    end
    
    Handler-->>UI: JSON Response<br/>- 期間別内訳<br/>- 統計サマリー<br/>- AI推奨事項
    
    UI->>UI: サマリーカード表示
    UI->>UI: 期間別テーブル表示
    UI->>UI: AI推奨事項表示
    UI-->>User: 詳細な分析結果
```

---

### 4-2. 需要予測フロー

**説明:** 製品の需要予測を実行し、日次内訳と信頼区間を提供するフロー。

**主な参照ファイル:** `pkg/handlers/demand_forecast_handler.go`, `src/app/dashboard/page.tsx`

```mermaid
sequenceDiagram
    participant User
    participant UI as "ダッシュボード"
    participant Handler as "DemandForecastHandler"
    participant Weather as "WeatherService"
    participant Azure as "AzureOpenAI"

    User->>UI: 製品 + 期間選択
    UI->>Handler: POST /api/v1/ai/forecast-product
    
    rect rgb(230, 230, 250)
        note over Handler,Weather: 気象データ取得
        Handler->>Weather: GetWeatherForecast(region, period)
        Weather-->>Handler: 気温・天候予報
    end
    
    rect rgb(255, 228, 196)
        note over Handler,Azure: AI需要予測
        Handler->>Azure: PredictDemand<br/>(product + weather + history)
        Azure->>Azure: トレンド分析
        Azure->>Azure: 季節性考慮
        Azure->>Azure: 曜日効果反映
        Azure-->>Handler: 日次予測 + 信頼区間
    end
    
    Handler-->>UI: JSON Response<br/>- 日次予測値<br/>- 信頼区間<br/>- 推奨事項
    
    UI->>UI: グラフ表示
    UI->>UI: 日次内訳テーブル
    UI-->>User: 予測結果 + 可視化
```

---

## 付録・使いかた

### 図の閲覧方法

- 図は Mermaid をサポートする Markdown ビューア（VS Code プレビュー等）で確認してください。
- GitHub上でも自動的にレンダリングされます。

### 図の更新ルール

1. **新機能追加時**: 該当するセクションにシーケンス図を追加
2. **処理変更時**: 既存の図を更新し、変更日をコメントで記載
3. **複雑な処理**: 概要図と詳細図に分割して可読性を保つ

### 関連ドキュメント

- [API_MANUAL.md](./API_MANUAL.md) - APIエンドポイント詳細
- [RAG_SYSTEM_GUIDE.md](./RAG_SYSTEM_GUIDE.md) - RAGシステムの詳細
- [PERFORMANCE_OPTIMIZATION_GUIDE.md](./PERFORMANCE_OPTIMIZATION_GUIDE.md) - パフォーマンス最適化
- [WEEKLY_ANALYSIS_GUIDE.md](./WEEKLY_ANALYSIS_GUIDE.md) - 製品別分析ガイド

---
