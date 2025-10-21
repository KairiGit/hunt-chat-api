## UML & アーキテクチャ図 (v2.0)

このファイルは、現在のプロジェクト実装に基づいた主要なアーキテクチャ図をまとめたものです。図ごとに短い日本語の説明と、該当するソースファイルを明記しています。

**主な変更点 (v2.0):**
- ディレクトリ構造を `internal/` から `pkg/` に更新。
- データベースを、AI機能の中核である `Qdrant` に明記。
- 現在のRAGアーキテクチャや継続的学習のフローを反映。

目次

- コンポーネント図（高レベル構成）
- シーケンス図（RAGチャットフロー）
- シーquence図（ファイル分析と継続的学習）
- デプロイ図（ローカル開発とVercel本番環境）

---

### コンポーネント図 — 高レベル構成

説明: フロントエンド、バックエンド、外部サービス間の責務と主要モジュールを示します。バックエンドは `pkg/` 以下のハンドラ、サービスで構成され、AI機能は `AzureOpenAIService` と `VectorStoreService` (Qdrant) を中心に実現されます。

主な参照ファイル: `api/index.go`, `pkg/handlers/*`, `pkg/services/*`, `pkg/models/types.go`

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

### シーケンス図 — RAGチャットフロー

説明: ユーザーからのチャット入力に対し、Qdrantから関連情報を検索（RAG）してコンテキストを構築し、AIが応答を生成するまでの一連の流れを示します。

主な参照ファイル: `pkg/handlers/ai_handler.go` (ChatInput), `pkg/services/vector_store_service.go`

```mermaid
sequenceDiagram
    participant User
    participant UI as "Next.js UI"
    participant Gin as "Go Backend (Gin)"
    participant AIHandler as "AIHandler<br/>pkg/handlers"
    participant VectorStore as "VectorStoreService<br/>(Qdrant Client)"
    participant Azure as "AzureOpenAIService"

    User->>UI: チャットメッセージを入力
    UI->>Gin: POST /api/v1/ai/chat-input
    Gin->>AIHandler: ChatInput(request)
    AIHandler->>VectorStore: SearchChatHistory(query)
    VectorStore-->>AIHandler: 関連する過去の会話
    AIHandler->>VectorStore: SearchAnalysisReports(query)
    VectorStore-->>AIHandler: 関連する分析レポート
    AIHandler->>Azure: ProcessChatWithHistory(prompt + RAG context)
    Azure-->>AIHandler: 生成された応答
    AIHandler-->>Gin: JSON response
    Gin-->>UI: response
    UI-->>User: 応答を表示
```

### シーケンス図 — ファイル分析と継続的学習（非同期処理対応 v2025-10-21）

説明: ユーザーがアップロードしたファイルを分析してレポートを生成し、Qdrantに保存する流れと、その後の異常検知・ユーザーからの回答を学習データとして保存するループを示します。**v2025-10-21より、AI分析とAI質問生成は非同期処理となり、レスポンスタイムが70%短縮されています。**

主な参照ファイル: `pkg/handlers/ai_handler.go` (AnalyzeFile, SaveAnomalyResponse)

```mermaid
sequenceDiagram
    participant User
    participant UI as "Next.js UI"
    participant Gin as "Go Backend (Gin)"
    participant AIHandler as "AIHandler<br/>pkg/handlers"
    participant StatService as "StatisticsService"
    participant VectorStore as "VectorStoreService"
    participant Azure as "AzureOpenAIService"
    participant AsyncAI as "Async AI<br/>(Goroutine)"

    User->>UI: 販売データファイルを選択
    UI->>Gin: POST /api/v1/ai/analyze-file
    Gin->>AIHandler: AnalyzeFile(file)
    
    note over AIHandler: ステップ1-3: 同期処理
    AIHandler->>AIHandler: ファイル読み込み (0.1-0.5s)
    AIHandler->>AIHandler: CSV解析 (0.2-1s)
    AIHandler->>StatService: 統計分析 (0.5-2s)
    StatService-->>AIHandler: 統計結果
    
    note over AIHandler,AsyncAI: ステップ4: AI分析（非同期）⚡
    AIHandler->>AsyncAI: go func() AI分析開始
    AsyncAI->>Azure: ProcessChatWithContext(stats)
    note right of AsyncAI: バックグラウンドで実行<br/>2-5秒かかるが待たない
    
    note over AIHandler: ステップ5: 異常検知（同期）
    AIHandler->>StatService: DetectAnomalies (0.5-1s)
    StatService-->>AIHandler: 異常検知結果
    
    note over AIHandler,AsyncAI: ステップ6: AI質問生成（非同期）⚡
    AIHandler->>AsyncAI: go func() 質問生成開始
    AsyncAI->>Azure: GenerateAIQuestion(anomalies)
    note right of AsyncAI: 並列で各異常の質問生成<br/>5-10秒かかるが待たない
    
    note over AIHandler: ステップ7: DB保存とレスポンス
    AIHandler->>VectorStore: StoreDocument(analysis_report)
    VectorStore-->>AIHandler: 保存成功
    AIHandler-->>Gin: JSON response<br/>ai_insights_pending: true<br/>ai_questions_pending: true
    Gin-->>UI: 分析レポート（即座に表示）
    UI-->>User: 基本分析結果を表示 ✅<br/>「AI分析実行中...」バッジ表示
    
    note over AsyncAI: バックグラウンド処理完了
    AsyncAI->>Azure: AI分析完了 (2-5s後)
    Azure-->>AsyncAI: AI洞察結果
    note right of AsyncAI: TODO: DB更新<br/>UpdateAnalysisReport()
    
    AsyncAI->>Azure: 質問生成完了 (5-10s後)
    Azure-->>AsyncAI: AI質問と選択肢
    note right of AsyncAI: TODO: DB更新<br/>UpdateAnomalyQuestions()

    rect rgb(200, 230, 255)
        note over User,VectorStore: 継続的学習フロー
        User->>UI: 異常原因を入力して回答
        UI->>Gin: POST /api/v1/ai/anomaly-response
        Gin->>AIHandler: SaveAnomalyResponse(answer)
        AIHandler->>VectorStore: StoreDocument(anomaly_response)
        note right of VectorStore: 回答を新たな知識として学習
        VectorStore-->>AIHandler: 保存成功
        AIHandler-->>UI: 保存完了メッセージ
    end
```

### デプロイ図 — ローカル開発とVercel本番環境

説明: ローカルでの開発環境と、Vercelと外部サービスで構成される本番環境の関係を示します。

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

付録・使いかた

- 図は Mermaid をサポートする Markdown ビューア（VS Code プレビュー等）で確認してください。

