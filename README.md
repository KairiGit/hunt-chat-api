# 🏭 HUNT - AI需要予測システム
![logo](/public/img/HUNT-logo.jpeg)

**製造業向けの次世代需要予測・異常検知・学習システム**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)
[![Next.js](https://img.shields.io/badge/Next.js-14+-black?logo=next.js)](https://nextjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.0+-3178C6?logo=typescript)](https://www.typescriptlang.org/)

## 📋 目次

- [概要](#概要)
- [主要機能](#主要機能)
- [アーキテクチャ & 技術スタック](#アーキテクチャ--技術スタック)
- [セットアップ](#セットアップ)
- [使い方](#使い方)
- [API仕様](#api仕様)
- [開発](#開発)
- [ドキュメント](#ドキュメント)

---

## 概要

**HUNT（Highly Unified Needs Tracker）** は、製造業のtoB需要予測に特化したAI搭載システムです。

### 🎯 解決する課題

- **需要予測の属人化**: ベテラン社員の勘に依存
- **異常値の見逃し**: 統計的な異常検知が不十分
- **学習データの欠如**: 過去の知見が蓄積されない
- **分析の時間コスト**: 手動での相関分析・回帰分析

### 💡 特徴

- **RAG搭載**: 過去のデータ・会話履歴から最適な回答を生成
- **深掘り質問**: AIが不足情報を自動で質問（最大2回）
- **リアルタイム分析**: CSV/Excelアップロードで即座に分析
- **異常検知+学習**: 3σ法で異常を検出し、原因を学習データ化

---

## 主要機能

### 1. 📊 ファイル分析

CSV/Excelファイルをアップロードすると、自動で以下を実行：

- **基本統計**: 平均、標準偏差、最大/最小値
- **相関分析**: 気温・湿度・降水量との相関係数（Pearson）
- **回帰分析**: 線形回帰による予測モデル構築
- **異常検知**: 3σ法による統計的異常値の検出
- **AIレポート**: Azure OpenAI (GPT-4) による洞察生成

**対応フォーマット:**
- CSV（カンマ区切り）
- Excel（.xlsx）

**必須カラム:**
- `date`: 日付（YYYY-MM-DD）
- `product_id`: 製品ID（システム内部での識別用）
- `sales`: 売上数量

**推奨カラム:**
- `product_name`: 製品名（AIの回答や異常通知での表示用）

### 2. 🔮 需要予測

#### 製品別・期間別予測

```
製品A × 1週間 → 日別予測 + 信頼区間
製品B × 1ヶ月 → 週次平均 + 季節性分析
```

**予測精度:**
- 信頼区間: 90% / 95% / 99%
- 考慮要因: 気温、曜日、過去の異常対応データ

#### 気温ベース予測

```
気温30°C → 製品A: 150個 (信頼区間: 120-180)
```

**回帰式:**
```
sales = α + β × temperature + ε
```

### 3. 🚨 異常検知

**3σ法（標準偏差3倍）:**

```
μ ± 3σ の範囲外 → 異常と判定
```

**重症度判定:**
- **Critical**: |Z| > 4.0
- **High**: 3.0 < |Z| ≤ 4.0
- **Medium**: 2.5 < |Z| ≤ 3.0

**AI質問生成:**
異常検出時、AIが自動で原因を質問：

```
「2024-01-07に製品Aで売上が急増しました。
 実績値: 300個、予測値: 120個
 → この原因は何だと思いますか？」
```

**選択肢自動生成:**
- キャンペーン・販促活動
- 天候・気温の影響
- イベント・行事
- 品切れ・欠品
- （その他：自由記述）

### 4. 🧠 AI学習

**深掘り質問機能:**

ユーザーの回答をAIが評価（0-100点）し、情報不足なら追加質問：

```
User: "キャンペーンを実施したため"
AI: (評価: 60点) → 追加質問
AI: "どのようなキャンペーンでしたか？期間や内容を教えてください"
User: "新春キャンペーン、1/5-1/10、20%OFF"
AI: (評価: 85点) → 完了
```

**学習データ活用:**
- Qdrantベクトルデータベースに保存
- 類似異常発生時に過去の対応を参照
- パターン分析で洞察を生成

### 5. 💬 AIチャット

**RAG（検索拡張生成）機能:**

質問に応じて自動で関連情報を検索：

```
ユーザー: "このシステムの機能は？"
  ↓
検索: README, API_MANUAL, 過去の会話
  ↓
回答: システム情報を統合して説明
```

**検索対象:**
1. 過去のチャット履歴（Top 3）
2. システムドキュメント（Top 2）
3. 分析レポート（キーワード検索）
4. 異常対応履歴（学習データ）

---

## アーキテクチャ & 技術スタック

本システムは、Next.jsによるUIとGoによるAPIサーバーで構成されています。

### Primary API (Go)
メインのAPIサーバーです。統計処理やAIとの連携など、コアなビジネスロジックを担当します。

- **言語**: Go (1.21+)
- **フレームワーク**: Gin (1.10+)
- **デプロイ**: Vercel (Serverless Function)

### Frontend & BFF (Next.js)
ユーザーが操作するUIと、Go APIへのプロキシとして機能するBFF（Backend for Frontend）です。

- **フレームワーク**: Next.js (15.5+, App Router)
- **UI**: React (18+), TypeScript, Tailwind CSS, shadcn/ui
- **デプロイ**: Vercel

### AI & Vector Database
AI機能とベクトル検索を支えるサービス群です。

- **AI**: Azure OpenAI (GPT-4, text-embedding-3-small)
- **ベクトルDB**: Qdrant (クラウドサービス)
- **ローカル開発**: Docker


---

## セットアップ

### 必要な環境

- **Go**: 1.21以上
- **Node.js**: 18以上
- **Docker**: 24以上
- **Azure OpenAI**: API キー

### 1. リポジトリのクローン

```bash
git clone https://github.com/YourOrg/hunt-chat-api.git
cd hunt-chat-api
```

### 2. 環境変数の設定

```bash
cp .env.example .env
```

`.env`を編集：

```env
# Azure OpenAI
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com/
AZURE_OPENAI_API_KEY=your-api-key-here
AZURE_OPENAI_DEPLOYMENT_NAME=gpt-4
AZURE_OPENAI_EMBEDDING_DEPLOYMENT=text-embedding-3-small
AZURE_OPENAI_API_VERSION=2024-02-15-preview

# Qdrant
QDRANT_URL=http://localhost:6333

# 気象API（オプション）
OPENWEATHERMAP_API_KEY=your-api-key-here
```

### 3. Qdrantの起動

```bash
docker run -d \
  --name qdrant_db \
  -p 6333:6333 \
  -p 6334:6334 \
  -v "$(pwd)/qdrant_storage:/qdrant/storage" \
  qdrant/qdrant
```

### 4. システムドキュメントの投入

```bash
# ローカル環境のQdrantに投入
make init-docs

# 本番環境のQdrant Cloudに投入（手動実行のみ・要確認プロンプト）
make init-docs-prod

# CI/CD環境での自動実行（環境変数 ENABLE_INIT_DOCS=true が必要）
make init-docs-auto
```

このコマンドで以下がQdrantに投入されます：
- README.md
- API_MANUAL.md
- IMPLEMENTATION_SUMMARY.md
- その他の.mdファイル

**注意:**
- `make init-docs` は環境変数`QDRANT_URL`で接続先が決まります
- 未設定の場合はローカル（`127.0.0.1:6334`）に接続
- 本番環境では環境変数で **Qdrant Cloud** に自動接続します
- `make init-docs-prod` はCI環境では自動的にスキップされます（誤実行防止）
- Vercelなどで自動実行したい場合は `ENABLE_INIT_DOCS=true` を設定してください

### 5. バックエンドの起動

```bash
# 依存関係のインストール
go mod download

# 開発サーバー起動
make run

# または
go run cmd/server/main.go
```

サーバーは `http://localhost:8080` で起動します。

### 6. フロントエンドの起動

```bash
# 依存関係のインストール
npm install

# 開発サーバー起動
npm run dev
```

フロントエンドは `http://localhost:3000` で起動します。

---

## 使い方

### 1. ファイル分析

1. **ダッシュボード** → **ファイル分析** に移動
2. CSV/Excelファイルをドラッグ&ドロップ
3. 自動で分析結果が表示されます

**サポートされているファイル形式:**
- CSV (.csv)
- Excel (.xlsx)

**必須列:**
- `date` または `日付` (日付列)
- `product_code`, `product_id`, `製品ID` など (製品ID列 - 必須)
- `product_name`, `製品名`, `product` など (製品名列 - 推奨)
- `sales`, `quantity`, `販売数` など (販売数列)

詳細は [FILE_FORMAT_GUIDE.md](./FILE_FORMAT_GUIDE.md) を参照してください。

**最小限のサンプルデータ（製品ID のみ）:**
```csv
date,product_id,sales
2024-01-01,P001,100
2024-01-02,P001,105
2024-01-03,P001,300  ← 異常値として検出
```

**推奨形式（製品名を含む）:**
```csv
date,product_id,product_name,sales
2024-01-01,P001,コーヒー豆（ブレンド）,100
2024-01-02,P001,コーヒー豆（ブレンド）,105
2024-01-03,P001,コーヒー豆（ブレンド）,300
```

**推奨フル形式（moc/moc.csvと同様）:**
```csv
日付,製品コード,製品,販売数,単価,売上金額,曜日,月,年
2024-01-01,P001,製品A,100,1000,100000,月,1,2024
```
※ 製品名列は推奨ですが必須ではありません。製品名がない場合は製品IDが表示に使用されます。

### 2. 異常対応

1. **異常対応** ページに移動
2. 未回答の異常がある場合、AIが質問を表示
3. 選択肢をクリック、または自由記述で回答
4. AIが追加質問する場合があります（最大2回）

### 3. AIチャット

1. **分析チャット** ページに移動
2. 質問を入力（例: 「このシステムの機能は？」）
3. AIが過去のデータ・ドキュメントを検索して回答

**サジェスト例:**
- 🤖 システムの機能
- 📈 予測の仕組み
- 🚨 異常検知について

**回答の出典表示:**
AIの回答には出典が明記されます：
- 📄 **システムドキュメントより**: 検索されたドキュメントからの情報
- 💡 **一般的な知識**: 統計学やビジネスの一般論
- 📊 **分析レポートより**: 過去の分析結果
- 🗣️ **過去の対話より**: 以前の会話履歴

これにより、どの情報がシステム固有で、どの情報が一般的な知識なのか明確に区別できます。

---

## API仕様

詳細は [API_MANUAL.md](./API_MANUAL.md) を参照してください。

### エンドポイント一覧

| エンドポイント | メソッド | 説明 |
|---------------|---------|------|
| `/api/v1/ai/analyze-file` | POST | ファイル分析 |
| `/api/v1/ai/detect-anomalies` | POST | 異常検知 |
| `/api/v1/ai/predict-sales` | POST | 売上予測 |
| `/api/v1/ai/forecast-product` | POST | 製品別需要予測 |
| `/api/v1/ai/anomaly-response-with-followup` | POST | 異常回答+深掘り |
| `/api/v1/ai/chat-input` | POST | AIチャット |
| `/api/v1/ai/anomaly-responses` | GET | 回答履歴取得 |
| `/api/v1/ai/learning-insights` | GET | 学習洞察取得 |

### リクエスト例

```bash
# ファイル分析
curl -X POST http://localhost:8080/api/v1/ai/analyze-file \
  -F "file=@sales_data.csv" \
  -F "region_code=240000"

# AIチャット
curl -X POST http://localhost:8080/api/v1/ai/chat-input \
  -H "Content-Type: application/json" \
  -d '{
    "chat_message": "このシステムの主要機能を教えてください",
    "session_id": "session-123",
    "user_id": "user-456"
  }'
```

---

## 開発

### ディレクトリ構成

```
hunt-chat-api/
├── cmd/
│   └── server/
│       └── main.go          # エントリーポイント
├── pkg/
│   ├── handlers/            # HTTPハンドラー
│   ├── services/            # ビジネスロジック
│   └── models/              # データモデル
├── configs/
│   ├── config.go            # 設定管理
│   └── system_prompt.yaml   # AIプロンプト設定
├── src/                     # Next.jsフロントエンド
│   ├── app/                 # ページコンポーネント
│   ├── components/          # UIコンポーネント
│   └── contexts/            # Reactコンテキスト
├── scripts/
│   └── init_docs.go         # ドキュメント投入スクリプト
└── qdrant_storage/          # Qdrantデータ
```

### Makefileコマンド

```bash
make run          # バックエンド起動
make test         # テスト実行
make build        # ビルド
make init-docs    # ドキュメント投入
make clean        # クリーンアップ
```

### テスト

```bash
# Go（バックエンド）
go test ./...

# TypeScript（フロントエンド）
npm test
```

---

## ドキュメント

📚 **[完全なドキュメント一覧はこちら](./docs/README.md)**

### クイックリンク

#### 📖 はじめに
- [API仕様書](./docs/api/API_MANUAL.md) - エンドポイント一覧、リクエスト/レスポンス形式
- [アーキテクチャ図](./docs/architecture/UML.md) - UML図・シーケンス図
- [ファイル形式ガイド](./docs/guides/FILE_FORMAT_GUIDE.md) - CSV/Excel形式の詳細

#### 🚀 主要機能
- [RAGシステムガイド](./docs/guides/RAG_SYSTEM_GUIDE.md) - 検索拡張生成の仕組み
- [AI学習ガイド](./docs/guides/AI_LEARNING_GUIDE.md) - AI深掘り質問機能
- [データ集約分析](./docs/guides/DATA_AGGREGATION_GUIDE.md) - 日次・週次・月次分析
- [経済データ相関分析](./docs/implementation/ECONOMIC_CORRELATION_IMPLEMENTATION.md) - 日経平均、為替、原油価格

#### 🔧 実装・最適化
- [最終実装サマリー](./docs/implementation/FINAL_IMPLEMENTATION_SUMMARY.md) - 相関分析最適化（上位3件表示）
- [非同期化完了](./docs/implementation/ASYNC_IMPLEMENTATION_COMPLETE.md) - 70%高速化達成 🆕
- [パフォーマンス最適化](./docs/guides/PERFORMANCE_OPTIMIZATION_GUIDE.md) - Phase 1完了 ✅

#### 🆘 トラブルシューティング
- [トラブルシューティング](./docs/guides/TROUBLESHOOTING_AND_BEST_PRACTICES.md) - よくある問題と解決策
- [Vercelデバッグ](./docs/guides/VERCEL_DEBUGGING_GUIDE.md) - デプロイ時の問題

### ドキュメントカテゴリ

| カテゴリ | 説明 | リンク |
|---------|------|--------|
| 📡 **API** | API仕様、エンドポイント | [docs/api/](./docs/api/) |
| 🏗️ **アーキテクチャ** | 設計図、UML、診断 | [docs/architecture/](./docs/architecture/) |
| 🔧 **実装** | 実装詳細、変更履歴 | [docs/implementation/](./docs/implementation/) |
| 📖 **ガイド** | 使い方、ベストプラクティス | [docs/guides/](./docs/guides/) |
| ✨ **機能** | 各機能の詳細設計 | [docs/features/](./docs/features/) |
| 📋 **プロジェクト** | 要件定義、進捗管理 | [docs/project/](./docs/project/) |

---

## ライセンス

MIT License

---

## 貢献

プルリクエストを歓迎します！

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## サポート

問題が発生した場合は [Issues](https://github.com/YourOrg/hunt-chat-api/issues) を作成してください。

---

**Last Updated:** 2025-10-20  
**Version:** 1.0.0
