# AI質問力向上戦略ガイド

## 📊 現状の課題分析

### 現在の質問生成の弱点
1. **単発的**: 異常検知時のみ質問を生成
2. **コンテキスト不足**: 過去の学習データや業界知識を活用していない
3. **深さ不足**: Why-Why分析のような段階的な深掘りが弱い
4. **予測精度向上に直結しない**: 質問が学習データとしての価値を最大化していない
5. **シナリオベース質問がない**: 仮説検証型の質問が不足

---

## 🎯 改善戦略（3段階アプローチ）

### **フェーズ1: 異常検知時の質問強化**

#### 1.1 多層質問フレームワーク（5W2H拡張版）
```
Layer 1: WHAT（事実確認）
  └─ 「何が起きたのか」を数値・固有名詞で特定

Layer 2: WHY（原因分析）
  └─ 「なぜそうなったのか」を複数階層で深掘り
  └─ Why-Why-Why（最低3階層）

Layer 3: HOW（プロセス確認）
  └─ 「どのようにして起きたのか」時系列で確認
  └─ 予兆はあったか、段階的変化だったか

Layer 4: WHEN/WHERE（範囲特定）
  └─ 「いつ・どこで」影響範囲を定量化
  └─ 他製品・他地域への波及効果

Layer 5: WHO（関係者確認）
  └─ 「誰が関与したか」意思決定プロセス
  └─ ステークホルダーの行動

Layer 6: HOW MUCH（定量化）
  └─ 「どの程度の影響か」金額・数量で測定
  └─ ROI、売上インパクト

Layer 7: HOW TO PREVENT（対策）
  └─ 「どう対処・予防するか」アクションプラン
  └─ 再発防止策の具体性
```

#### 1.2 コンテキスト活用質問
```javascript
// 質問生成時に以下を参照
{
  "過去の類似異常": "同じ製品・同じ季節の過去データ",
  "業界トレンド": "同業他社の動向、市場全体の傾向",
  "社内イベント": "過去のキャンペーン履歴、価格変更履歴",
  "外部要因": "天候パターン、祝日、大型イベント",
  "ユーザーの回答履歴": "このユーザーが過去に答えた異常原因"
}
```

#### 1.3 質問テンプレート改善例

**❌ 現在（弱い質問）**
```
「2024年3月15日の売上急増について、何か思い当たることはありますか？」

選択肢:
- キャンペーン・販促活動
- 天候の影響
- 競合他社の動き
- 特に思い当たる節はない
- その他
```

**✅ 改善後（強い質問）**
```
「2024年3月15日、製品Aの売上が通常比+45%と大幅に増加しました。

📊 データで見えていること:
- 前週比: +45% (通常の週末効果+10%を大きく超過)
- 同製品の過去3年間で最高記録
- 同日の製品B・Cは通常通り（この製品だけが急増）
- 気温は平年並み（天候要因は低い）

🔍 過去の類似パターン:
- 昨年の同時期に+30%増加 → 原因: 「春の新生活キャンペーン」
- 6ヶ月前に+50%増加 → 原因: 「TVCMの放映開始」

💡 いくつか具体的に教えてください：

1️⃣ この日に特別な販促活動を実施しましたか？
   - 実施した → どんな内容か（例: 30%OFF、ポイント5倍、ノベルティ配布）
   - 実施していない

2️⃣ 社内で新しい取り組みは始めましたか？
   - 広告出稿（媒体名・予算規模）
   - SNS投稿のバズり（いいね数、リーチ数）
   - インフルエンサー起用
   - 新販路開拓（EC・実店舗拡大）
   - 該当なし

3️⃣ 競合や業界で大きな動きはありましたか？
   - 競合A社が○○を発表
   - 業界全体で需要増加のニュース
   - 規制変更・制度改正
   - 該当なし

4️⃣ 顧客から直接聞いた声はありますか？
   - 「○○で見て買いに来た」
   - 「友人・知人から勧められた」
   - 「いつもより安かったから」
   - 「特に理由は聞いていない」
```

---

### **フェーズ2: シナリオベース質問（仮説検証型）**

#### 2.1 仮説駆動型質問フレームワーク
AIが複数の仮説を立て、それを検証する質問を生成

```python
# 仮説生成ロジック（疑似コード）
anomaly = detect_anomaly(data)
hypotheses = [
    {
        "hypothesis": "競合他社の値上げにより相対的に自社製品が割安に見えた",
        "confidence": 0.65,
        "verification_questions": [
            "競合A社・B社の価格変更情報はありますか？",
            "お客様から「他より安い」という声は聞きましたか？",
            "同カテゴリの他製品の売上も増えましたか？"
        ],
        "data_evidence": "同カテゴリ全体で売上+12%（製品Aは+45%で突出）"
    },
    {
        "hypothesis": "SNSでの口コミ拡散（バイラル効果）",
        "confidence": 0.72,
        "verification_questions": [
            "Twitter/Instagram/TikTokでの言及数を確認できますか？",
            "特定のインフルエンサーが投稿していませんか？",
            "Googleトレンドでの検索急増はありますか？"
        ],
        "data_evidence": "新規顧客比率が通常30%→55%に急増（既存顧客以外が購入）"
    },
    {
        "hypothesis": "大型イベント・記念日との関連",
        "confidence": 0.48,
        "verification_questions": [
            "3月15日は何か特別な日ですか？（ホワイトデー翌日）",
            "この製品はギフト需要がありますか？",
            "イベント関連のPOPや陳列変更をしましたか？"
        ],
        "data_evidence": "該当日は大型祝日ではないが、ホワイトデー直後"
    }
]

# 信頼度が高い仮説から質問を生成
for hypothesis in sorted(hypotheses, key=lambda h: h["confidence"], reverse=True):
    ask_verification_questions(hypothesis)
```

#### 2.2 シナリオ質問の例

**製造業向けシナリオ**
```
【シナリオ1: サプライチェーン分析】
「部品Xの発注が急増していますね。

考えられるシナリオ:
A) 顧客からの受注が増えた（需要起因）
B) 部品の不良率が高く、予備を多く発注している（品質起因）
C) サプライヤーのリードタイムが長くなり、在庫を厚めにした（リスク管理起因）

どのシナリオに近いですか？複数該当する場合もお答えください。」
```

**小売業向けシナリオ**
```
【シナリオ2: 顧客行動分析】
「夕方16-18時の来店客数が急増しています。

考えられる顧客行動の変化:
A) 近隣に新しいオフィスビルが開業した
B) 競合店が閉店し、顧客が流入した
C) 通勤・通学ルートが変わった（道路工事、新駅開業など）
D) 時間帯限定セールを始めた

実際に観察した顧客の変化を教えてください（客層、購買パターンなど）。」
```

---

### **フェーズ3: 予測精度向上のための継続質問**

#### 3.1 学習サイクル質問（異常回答後も継続）

```
【タイミング1: 異常発生直後】
→ 初期質問: 「何が起きたか」

【タイミング2: 1週間後】
→ フォローアップ: 「その後の変化」
   - 「先週の売上急増は今週も継続していますか？」
   - 「一過性だった場合、いつ収束しましたか？」

【タイミング3: 1ヶ月後】
→ 効果測定: 「実施した対策の効果」
   - 「新キャンペーンの効果は持続していますか？」
   - 「売上増加は利益にも貢献しましたか？（マージン確認）」

【タイミング4: 3ヶ月後/次の同時期】
→ パターン確認: 「季節性・再現性の検証」
   - 「同じ施策を再度実施する予定はありますか？」
   - 「昨年と比べて今年の売上パターンに変化はありますか？」

【タイミング5: 年次レビュー】
→ 戦略質問: 「来年の計画への反映」
   - 「今年学んだ需要パターンを来年の計画に活かす予定はありますか？」
   - 「最も効果的だった施策TOP3は何ですか？」
```

#### 3.2 予測モデル改善質問

AIが予測精度を自己改善するための質問：

```
【モデル改善質問1: 外れ値の扱い】
「AIの予測: 1000個、実績: 1500個（誤差+50%）でした。

予測が外れた理由として、以下のどれが当てはまりますか？
1. 予測不可能な突発イベント（今後も予測困難）
2. 実は予兆があった（データに含めるべき要素が不足）
3. 定期的に起きるパターン（季節性・曜日効果の未反映）
4. 人的判断で対処可能（AIには学習させなくてOK）

→ 回答に応じてモデルを更新
```

```
【モデル改善質問2: 特徴量の重要性】
「現在のAIは以下の要素を重視して予測しています：
1. 過去の売上トレンド（重要度: 45%）
2. 曜日効果（重要度: 25%）
3. 気温（重要度: 20%）
4. 降水量（重要度: 10%）

あなたの実感として、他に重要な要素はありますか？
例）
- 競合店の出店・閉店情報
- TV・新聞・SNSでの露出
- 原材料価格の変動
- 政治・経済ニュース
```

```
【モデル改善質問3: 粒度の最適化】
「現在は『週次予測』を提供していますが、以下のどの粒度が最も有用ですか？

A) 日次予測（細かいが変動大）
B) 週次予測（現状）
C) 月次予測（大まかだが安定）
D) 製品カテゴリごとに粒度を変えたい

また、予測期間は何日先まで必要ですか？（現在: 7日先）
→ 14日先、30日先、90日先など
```

---

## 🛠️ 実装ガイド

### ステップ1: 質問テンプレートの構造化

```go
// pkg/models/question_strategy.go
package models

type QuestionStrategy struct {
    StrategyType    string   // "anomaly_initial", "scenario_based", "follow_up", "model_improvement"
    ContextLayers   []string // ["past_similar_anomalies", "industry_trends", "user_history"]
    QuestionLayers  []QuestionLayer
    HypothesisList  []Hypothesis
}

type QuestionLayer struct {
    Layer           string   // "WHAT", "WHY", "HOW", "WHEN", "WHERE", "WHO", "HOW_MUCH", "HOW_TO_PREVENT"
    Questions       []string
    Choices         []string
    Priority        int      // 1が最優先
    RequiredContext []string // この質問に必要なコンテキスト
}

type Hypothesis struct {
    HypothesisText       string
    ConfidenceScore      float64  // 0.0 - 1.0
    DataEvidence         string   // この仮説を支持するデータ
    VerificationQuestions []string
    ExpectedAnswerPattern string  // 正規表現やキーワード
}
```

### ステップ2: 強化された質問生成メソッド

```go
// pkg/services/azure_openai_service.go に追加

// GenerateEnhancedQuestion は、コンテキストと仮説を活用した質問を生成
func (aos *AzureOpenAIService) GenerateEnhancedQuestion(
    anomaly models.Anomaly,
    pastSimilarAnomalies []models.AnomalyResponse, // 過去の類似異常
    userHistory []models.AnomalyResponse,           // このユーザーの回答履歴
    weatherContext string,                          // 気象データ
    industryTrends string,                          // 業界トレンド
) (*models.EnhancedQuestion, error) {

    systemPrompt := `あなたは需要予測の専門家です。以下の情報を総合的に分析し、
ユーザーから最も価値のある情報を引き出す質問を生成してください。

【質問設計の原則】
1. 5W2H（What/Why/How/When/Where/Who/How Much）を網羅
2. 複数の仮説を立て、検証する質問を含める
3. 過去の類似ケースと比較し、パターンを特定する質問
4. 具体的な数値・固有名詞を引き出す
5. 予測モデルの改善に直結する情報を得る

【出力形式】
{
  "primary_question": "メインの質問文（2-3文で状況を説明してから質問）",
  "context_summary": "データから見えていること（箇条書き3-5項目）",
  "past_patterns": "過去の類似ケース（該当する場合）",
  "hypotheses": [
    {
      "hypothesis": "仮説1の内容",
      "confidence": 0.75,
      "verification_question": "この仮説を検証する質問",
      "choices": ["選択肢1", "選択肢2", ...]
    }
  ],
  "follow_up_plan": "この回答後に聞くべき次の質問（想定）"
}`

    userPrompt := fmt.Sprintf(`
【異常データ】
- 日付: %s
- 製品ID: %s
- 実績値: %.2f
- 予測値: %.2f
- 偏差: %.1f%%
- 説明: %s

【過去の類似異常】
%s

【このユーザーの回答傾向】
%s

【気象コンテキスト】
%s

【業界トレンド】
%s

上記を分析し、JSON形式で質問を生成してください。`,
        anomaly.Date,
        anomaly.ProductID,
        anomaly.ActualValue,
        anomaly.PredictedValue,
        anomaly.Deviation,
        anomaly.Description,
        formatPastAnomalies(pastSimilarAnomalies),
        formatUserHistory(userHistory),
        weatherContext,
        industryTrends,
    )

    // ... AIに送信してJSON解析 ...
}
```

### ステップ3: シナリオベース質問の実装

```go
// pkg/services/scenario_question_service.go (新規作成)
package services

type ScenarioQuestionService struct {
    azureOpenAIService *AzureOpenAIService
    vectorStoreService *VectorStoreService
}

// GenerateScenarioBasedQuestions は複数の仮説シナリオを生成
func (sqs *ScenarioQuestionService) GenerateScenarioBasedQuestions(
    anomaly models.Anomaly,
    dataContext models.DataContext,
) ([]models.ScenarioQuestion, error) {

    systemPrompt := `あなたは需要予測アナリストです。
異常データから複数の「もっともらしいシナリオ（仮説）」を生成し、
それぞれを検証する質問を作成してください。

【シナリオ生成の原則】
1. 最低3つ、最大5つの仮説を立てる
2. 各仮説に「信頼度スコア」を付与（データの裏付けの強さ）
3. 仮説同士は排他的でなくてOK（複数が同時に成り立つこともある）
4. 各仮説を検証するための具体的質問を2-3個用意

【シナリオの分類】
- 内部要因（自社のアクション: キャンペーン、価格変更、在庫戦略）
- 外部要因（市場・競合: 競合の動き、業界トレンド、規制変更）
- 環境要因（天候、災害、社会イベント）
- 顧客行動（需要の本質的変化、トレンドシフト）

【出力形式】
{
  "scenarios": [
    {
      "scenario_id": "S1",
      "category": "internal",
      "title": "シナリオのタイトル",
      "description": "詳細な説明",
      "confidence": 0.72,
      "data_evidence": "このシナリオを支持するデータ",
      "verification_questions": [
        {
          "question": "検証質問1",
          "choices": ["選択肢1", "選択肢2", ...]
        }
      ]
    }
  ]
}`

    // ... 実装 ...
}
```

### ステップ4: 継続質問スケジューラー

```go
// pkg/services/follow_up_scheduler.go (新規作成)
package services

type FollowUpScheduler struct {
    db *sql.DB
}

// ScheduleFollowUpQuestions は異常回答後のフォローアップを計画
func (fus *FollowUpScheduler) ScheduleFollowUpQuestions(
    anomalyID string,
    initialResponse models.AnomalyResponse,
) error {

    followUpSchedule := []struct {
        delayDays int
        questionType string
    }{
        {7, "short_term_effect"},     // 1週間後
        {30, "medium_term_effect"},   // 1ヶ月後
        {90, "long_term_pattern"},    // 3ヶ月後
        {365, "yearly_review"},       // 1年後
    }

    for _, schedule := range followUpSchedule {
        scheduledDate := time.Now().AddDate(0, 0, schedule.delayDays)
        
        // データベースにスケジュール登録
        _, err := fus.db.Exec(`
            INSERT INTO follow_up_questions (anomaly_id, scheduled_date, question_type, status)
            VALUES (?, ?, ?, 'pending')
        `, anomalyID, scheduledDate, schedule.questionType)
        
        if err != nil {
            return err
        }
    }

    return nil
}

// GetPendingFollowUps は今日実行すべきフォローアップ質問を取得
func (fus *FollowUpScheduler) GetPendingFollowUps() ([]models.FollowUpQuestion, error) {
    // ... 実装 ...
}
```

---

## 📈 効果測定指標

### 質問の質を測る指標
1. **回答率**: 質問に対してユーザーが回答する割合（目標: 80%以上）
2. **回答の具体性スコア**: AIが回答の具体性を0-100で評価（目標: 平均70以上）
3. **情報密度**: 1回答あたりの有用情報数（固有名詞、数値など）
4. **深掘り成功率**: フォローアップ質問で新情報が得られた割合
5. **予測精度改善率**: 質問回答を学習データに加えた後のMAPE改善度

### 予測精度向上への貢献
- **ベースライン**: 質問なしの予測精度
- **質問あり**: 質問回答を学習データに含めた後の精度
- **目標**: MAPE（平均絶対パーセント誤差）を10%以上改善

---

## 🚀 段階的実装ロードマップ

### Week 1-2: 基礎強化
- [ ] 現在の質問生成ロジックを多層フレームワークに更新
- [ ] コンテキスト情報（過去の類似異常、ユーザー履歴）の取得機能実装
- [ ] 質問テンプレートを5W2H構造に再設計

### Week 3-4: 仮説駆動質問
- [ ] 仮説生成ロジックの実装
- [ ] シナリオベース質問サービスの構築
- [ ] 信頼度スコアの計算アルゴリズム

### Week 5-6: 継続質問
- [ ] フォローアップスケジューラーの実装
- [ ] タイミング別質問テンプレートの作成
- [ ] 通知システムとの連携

### Week 7-8: 効果測定
- [ ] 質問品質の評価指標を実装
- [ ] A/Bテスト: 旧質問 vs 新質問
- [ ] 予測精度への影響を測定

---

## 💡 質問例集（業種別）

### 製造業
```
【在庫管理】
「部品Aの発注量が通常の2倍になっています。

🔍 確認事項:
1. 生産計画に変更がありましたか？（製品Xの受注増など）
2. 部品Aの不良率が上がっていませんか？（品質問題）
3. サプライヤーから納期遅延の連絡はありましたか？
4. 年度末の在庫積み増し方針ですか？

具体的な背景を教えてください。」
```

### 小売業
```
【客単価分析】
「先週の客単価が15%上昇しました。

💰 考えられる要因:
1. 高額商品（単価○○円以上）の売上が増えた
2. まとめ買い（購入点数）が増えた
3. クロスセル（関連商品の同時購入）が増えた
4. 富裕層・法人客の来店が増えた

どの要因が大きいと思いますか？また、何か施策を実施しましたか？」
```

### ECサイト
```
【コンバージョン率】
「商品ページのコンバージョン率が2%→5%に急上昇しました。

🖥️ サイト改善チェック:
1. 商品画像・説明文を変更しましたか？
2. 価格を変更しましたか？
3. レビュー・口コミが増えましたか？
4. 検索順位・広告表示が上がりましたか？
5. 配送料無料などの特典を追加しましたか？

実施した施策を具体的に教えてください。」
```

---

## 🎓 まとめ: 良い質問の条件

### ✅ 良い質問の特徴
1. **具体的**: 曖昧な表現ではなく、数値・固有名詞を求める
2. **コンテキストあり**: 「データから見えていること」を先に示す
3. **仮説提示**: 「こういう可能性がありますが」と選択肢を示す
4. **段階的**: 一度に全てを聞かず、回答に応じて深掘り
5. **実用的**: 回答が予測モデルの改善に直結する
6. **学習的**: 過去の回答履歴を活かして質問を進化させる

### ❌ 避けるべき質問
1. 曖昧すぎる: 「何か思い当たることは？」
2. Yes/Noで終わる: 「キャンペーンをしましたか？」
3. 既知の情報を尋ねる: 「売上はいくらでしたか？」（データで分かる）
4. 回答不能: 「今後の景気動向をどう見ますか？」（ユーザーは分からない）
5. 単発で終わる: 深掘りの計画がない

---

## 🔄 継続的改善サイクル

```
1. 質問を生成
   ↓
2. ユーザーが回答
   ↓
3. AIが回答の質を評価（具体性スコア、情報密度）
   ↓
4. 必要なら深掘り質問
   ↓
5. 回答を学習データに追加
   ↓
6. 予測精度を再測定
   ↓
7. 質問テンプレートを改善（精度向上に貢献した質問を強化）
   ↓
（1に戻る）
```

---

**次のアクション**: このガイドに基づいて、まず `GenerateEnhancedQuestion` メソッドを実装してみましょうか？
