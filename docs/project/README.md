# 📋 プロジェクト管理 ドキュメント

プロジェクトの要件定義、進捗管理、ロードマップなどが含まれています。

## ドキュメント一覧

### 要件・仕様

- [要件定義.md](./要件定義.md) - システム要件定義
- [要求定義.md](./要求定義.md) - 要求定義
- [MVP.md](./MVP.md) - MVP（Minimum Viable Product）定義

### 計画・管理

- [ロードマップ.md](./ロードマップ.md) - 開発ロードマップ
- [優先順位.md](./優先順位.md) - 機能の優先順位
- [ワークフロー.md](./ワークフロー.md) - 開発ワークフロー

### 進捗報告

- [PROGRESS_REPORT.md](./PROGRESS_REPORT.md) - 進捗レポート

### その他

- [就活用自己PR.md](./就活用自己PR.md) - プロジェクト紹介（就活用）
- [DL講座1.md](./DL講座1.md) - 深層学習講座資料

## 📊 プロジェクト概要

### 目的
製造業向けのAI需要予測システムの開発

### 主要機能
- ファイル分析（CSV/Excel）
- 異常検知（3σ法）
- AI学習・深掘り質問
- RAG搭載チャット
- 需要予測

### 技術スタック
- **フロント**: Next.js 14, React 18, TypeScript 5, Tailwind CSS
- **バック**: Go 1.21+, Gin, Azure OpenAI (GPT-4)
- **DB**: Qdrant (ベクトルDB)

### デプロイ
- **フロント**: Vercel
- **バック**: Docker + VPS
- **DB**: Docker Compose

---

[← ドキュメントTOPへ](../README.md)
