# ✨ 機能 ドキュメント

各機能の詳細説明と設計ドキュメントです。

## ドキュメント一覧

### [ANOMALY_DETECTION_WEEKLY_AGGREGATION.md](./ANOMALY_DETECTION_WEEKLY_AGGREGATION.md)
**異常検知の週次集約対応**

週次集約データでの異常検知機能の実装詳細。

**主な内容:**
- 週次データでの3σ法適用
- 集約レベルごとの異常検知アルゴリズム
- 実装例とテストケース

### [CHAT_HISTORY_RAG.md](./CHAT_HISTORY_RAG.md)
**チャット履歴RAG機能**

過去のチャット履歴を活用したRAG（検索拡張生成）機能の詳細。

**主な内容:**
- チャット履歴のベクトル化
- Qdrantでの保存・検索
- 会話コンテキストの構築方法

### [AI_QUESTION_STRATEGY.md](./AI_QUESTION_STRATEGY.md)
**AI質問戦略**

AIが自動で質問を生成し、深掘りする機能の戦略と設計。

**主な内容:**
- 質問生成アルゴリズム
- 評価スコアリング（0-100点）
- 最大2回の深掘り質問
- 選択肢の自動生成

### [ANALYSIS_DATA_FLOW.md](./ANALYSIS_DATA_FLOW.md)
**分析データフロー**

分析結果の保存・活用フローの全体像。

**主な内容:**
- ファイルアップロード → 分析 → 保存 → 活用
- Qdrantへの保存形式
- RAGによる再利用
- チャットでの活用例

## 🔍 機能別の詳細リンク

### 異常検知
- [ANOMALY_DETECTION_WEEKLY_AGGREGATION.md](./ANOMALY_DETECTION_WEEKLY_AGGREGATION.md)
- [../implementation/ANOMALY_DISPLAY_IMPROVEMENT.md](../implementation/ANOMALY_DISPLAY_IMPROVEMENT.md)

### AI学習・質問
- [AI_QUESTION_STRATEGY.md](./AI_QUESTION_STRATEGY.md)
- [../guides/AI_LEARNING_GUIDE.md](../guides/AI_LEARNING_GUIDE.md)
- [../implementation/AI_QUESTION_IMPLEMENTATION.md](../implementation/AI_QUESTION_IMPLEMENTATION.md)

### RAGシステム
- [CHAT_HISTORY_RAG.md](./CHAT_HISTORY_RAG.md)
- [../guides/RAG_SYSTEM_GUIDE.md](../guides/RAG_SYSTEM_GUIDE.md)

### 分析・相関
- [ANALYSIS_DATA_FLOW.md](./ANALYSIS_DATA_FLOW.md)
- [../implementation/ECONOMIC_CORRELATION_IMPLEMENTATION.md](../implementation/ECONOMIC_CORRELATION_IMPLEMENTATION.md)

---

[← ドキュメントTOPへ](../README.md)
