'use client';

import { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { useToast } from '@/components/ui/use-toast';

interface AnomalyDetection {
  date: string;
  actual_value: number;
  expected_value: number;
  deviation: number;
  z_score: number;
  anomaly_type: string;
  severity: string;
  ai_question?: string;
}

interface AnomalyResponse {
  response_id: string;
  anomaly_date: string;
  product_id: string;
  question: string;
  answer: string;
  tags: string[];
  impact: string;
  impact_value: number;
  timestamp: string;
}

interface LearningInsight {
  insight_id: string;
  category: string;
  pattern: string;
  examples: string[];
  average_impact: number;
  confidence: number;
  learned_from: number;
  last_updated: string;
}

export default function LearningPage() {
  const { toast } = useToast();
  const [anomalies, setAnomalies] = useState<AnomalyDetection[]>([]);
  const [responses, setResponses] = useState<AnomalyResponse[]>([]);
  const [insights, setInsights] = useState<LearningInsight[]>([]);
  const [isLoadingAnomalies, setIsLoadingAnomalies] = useState(false);
  const [isLoadingResponses, setIsLoadingResponses] = useState(false);
  const [isLoadingInsights, setIsLoadingInsights] = useState(false);

  // 回答フォーム
  const [selectedAnomaly, setSelectedAnomaly] = useState<AnomalyDetection | null>(null);
  const [answer, setAnswer] = useState('');
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [impact, setImpact] = useState<string>('positive');
  const [impactValue, setImpactValue] = useState<number>(0);
  const [isSaving, setIsSaving] = useState(false);

  const availableTags = [
    'キャンペーン',
    'テレビCM',
    '競合値引き',
    '気象要因',
    'イベント',
    '新製品発売',
    '在庫不足',
    'システム障害',
    '季節要因',
    'その他',
  ];

  // サンプル異常データ
  const sampleSales = [100, 105, 110, 115, 95, 120, 300, 125, 130, 135, 140, 145, 50, 150];
  const sampleDates = [
    '2024-01-01', '2024-01-02', '2024-01-03', '2024-01-04', '2024-01-05',
    '2024-01-06', '2024-01-07', '2024-01-08', '2024-01-09', '2024-01-10',
    '2024-01-11', '2024-01-12', '2024-01-13', '2024-01-14',
  ];

  // 異常検知
  const detectAnomalies = async () => {
    setIsLoadingAnomalies(true);
    try {
      const response = await fetch('/api/proxy/detect-anomalies', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          sales: sampleSales,
          dates: sampleDates,
        }),
      });

      if (!response.ok) throw new Error('異常検知に失敗しました');

      const data = await response.json();
      setAnomalies(data.anomalies || []);
    } catch (error) {
      console.error('Error:', error);
      toast({
        variant: "destructive",
        title: "エラー",
        description: "異常検知中にエラーが発生しました",
      });
    } finally {
      setIsLoadingAnomalies(false);
    }
  };

  // 回答を保存
  const saveResponse = async () => {
    if (!selectedAnomaly || !answer.trim()) {
      toast({
        variant: "destructive",
        title: "入力エラー",
        description: "回答を入力してください",
      });
      return;
    }

    setIsSaving(true);
    try {
      const response = await fetch('/api/proxy/anomaly-response', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          anomaly_date: selectedAnomaly.date,
          product_id: 'P001',
          question: selectedAnomaly.ai_question || '異常の原因を教えてください',
          answer: answer,
          answer_type: 'text',
          tags: selectedTags,
          impact: impact,
          impact_value: impactValue,
        }),
      });

      if (!response.ok) throw new Error('回答の保存に失敗しました');

      const data = await response.json();
      toast({
        variant: "success",
        title: "✅ 保存完了",
        description: data.message || "回答を保存しました。AIが学習データとして活用します。",
      });
      
      // フォームをリセット
      setAnswer('');
      setSelectedTags([]);
      setImpactValue(0);
      setSelectedAnomaly(null);
      
      // 回答履歴を再読み込み
      loadResponses();
      loadInsights();
    } catch (error) {
      console.error('Error:', error);
      toast({
        variant: "destructive",
        title: "エラー",
        description: "回答の保存中にエラーが発生しました",
      });
    } finally {
      setIsSaving(false);
    }
  };

  // 回答履歴を取得
  const loadResponses = async () => {
    setIsLoadingResponses(true);
    try {
      const response = await fetch('/api/proxy/anomaly-responses?limit=50');
      if (!response.ok) throw new Error('回答履歴の取得に失敗しました');

      const data = await response.json();
      setResponses(data.responses || []);
    } catch (error) {
      console.error('Error:', error);
    } finally {
      setIsLoadingResponses(false);
    }
  };

  // 学習洞察を取得
  const loadInsights = async () => {
    setIsLoadingInsights(true);
    try {
      const response = await fetch('/api/proxy/learning-insights');
      if (!response.ok) throw new Error('学習洞察の取得に失敗しました');

      const data = await response.json();
      setInsights(data.insights || []);
    } catch (error) {
      console.error('Error:', error);
    } finally {
      setIsLoadingInsights(false);
    }
  };

  const toggleTag = (tag: string) => {
    if (selectedTags.includes(tag)) {
      setSelectedTags(selectedTags.filter(t => t !== tag));
    } else {
      setSelectedTags([...selectedTags, tag]);
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical': return 'bg-red-500';
      case 'high': return 'bg-orange-500';
      case 'medium': return 'bg-yellow-500';
      default: return 'bg-blue-500';
    }
  };

  const getImpactColor = (impact: string) => {
    switch (impact) {
      case 'positive': return 'text-green-600';
      case 'negative': return 'text-red-600';
      default: return 'text-gray-600';
    }
  };

  useEffect(() => {
    detectAnomalies();
    loadResponses();
    loadInsights();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="min-h-screen bg-gradient-to-br from-purple-50 to-pink-50 p-8">
      <div className="max-w-7xl mx-auto space-y-6">
        {/* ヘッダー */}
        <div>
          <h1 className="text-4xl font-bold text-gray-800 mb-2">🧠 AI学習システム</h1>
          <p className="text-gray-600">異常への回答を通じてAIが学習し、より正確な予測を実現します</p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* 左カラム：異常検知と回答 */}
          <div className="space-y-6">
            {/* 異常検知カード */}
            <Card>
              <CardHeader>
                <CardTitle>🔍 検出された異常</CardTitle>
                <CardDescription>AIが質問を生成します。回答することで学習データになります。</CardDescription>
              </CardHeader>
              <CardContent>
                {isLoadingAnomalies ? (
                  <div className="text-center py-8">読み込み中...</div>
                ) : anomalies.length === 0 ? (
                  <div className="text-center py-8 text-gray-500">異常は検出されませんでした</div>
                ) : (
                  <div className="space-y-3">
                    {anomalies.map((anomaly, index) => (
                      <div
                        key={index}
                        className={`p-4 rounded-lg border-2 cursor-pointer transition-all ${
                          selectedAnomaly?.date === anomaly.date
                            ? 'border-purple-500 bg-purple-50'
                            : 'border-gray-200 hover:border-gray-300'
                        }`}
                        onClick={() => setSelectedAnomaly(anomaly)}
                      >
                        <div className="flex items-start justify-between mb-2">
                          <div className="flex-1">
                            <div className="flex items-center gap-2 mb-1">
                              <span className={`px-2 py-1 rounded text-xs text-white ${getSeverityColor(anomaly.severity)}`}>
                                {anomaly.severity.toUpperCase()}
                              </span>
                              <span className="text-sm text-gray-600">{anomaly.date}</span>
                            </div>
                            <div className="text-sm">
                              <span className="font-semibold">{anomaly.anomaly_type}</span>
                              <span className="text-gray-600 ml-2">
                                実績: {anomaly.actual_value.toFixed(0)} (期待値: {anomaly.expected_value.toFixed(0)})
                              </span>
                            </div>
                          </div>
                        </div>
                        {anomaly.ai_question && (
                          <div className="mt-2 p-3 bg-blue-50 rounded">
                            <div className="text-sm font-medium text-blue-900 mb-1">💬 AIの質問:</div>
                            <div className="text-sm text-blue-800">{anomaly.ai_question}</div>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>

            {/* 回答フォーム */}
            {selectedAnomaly && (
              <Card className="border-2 border-purple-500">
                <CardHeader>
                  <CardTitle>✍️ 回答フォーム</CardTitle>
                  <CardDescription>
                    {selectedAnomaly.date} の異常について教えてください
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <Label htmlFor="answer">回答</Label>
                    <Textarea
                      id="answer"
                      value={answer}
                      onChange={(e) => setAnswer(e.target.value)}
                      placeholder="例: 新春キャンペーンを実施したため、通常より30%売上が増加しました"
                      rows={4}
                      className="mt-1"
                    />
                  </div>

                  <div>
                    <Label>要因タグ（複数選択可）</Label>
                    <div className="flex flex-wrap gap-2 mt-2">
                      {availableTags.map((tag) => (
                        <button
                          key={tag}
                          onClick={() => toggleTag(tag)}
                          className={`px-3 py-1 rounded-full text-sm transition-all ${
                            selectedTags.includes(tag)
                              ? 'bg-purple-500 text-white'
                              : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                          }`}
                        >
                          {tag}
                        </button>
                      ))}
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <Label htmlFor="impact">影響</Label>
                      <select
                        id="impact"
                        value={impact}
                        onChange={(e) => setImpact(e.target.value)}
                        className="w-full p-2 border rounded-lg mt-1"
                      >
                        <option value="positive">プラス影響</option>
                        <option value="negative">マイナス影響</option>
                        <option value="neutral">中立</option>
                      </select>
                    </div>

                    <div>
                      <Label htmlFor="impact-value">影響度（%）</Label>
                      <Input
                        id="impact-value"
                        type="number"
                        value={impactValue}
                        onChange={(e) => setImpactValue(parseFloat(e.target.value))}
                        placeholder="例: 30"
                        className="mt-1"
                      />
                    </div>
                  </div>

                  <Button
                    onClick={saveResponse}
                    disabled={isSaving || !answer.trim()}
                    className="w-full"
                  >
                    {isSaving ? '保存中...' : '💾 回答を保存してAIに学習させる'}
                  </Button>
                </CardContent>
              </Card>
            )}
          </div>

          {/* 右カラム：学習洞察と履歴 */}
          <div className="space-y-6">
            {/* AI学習洞察 */}
            <Card>
              <CardHeader>
                <CardTitle>🎓 AIが学習したパターン</CardTitle>
                <CardDescription>回答から発見された需要変動の法則</CardDescription>
              </CardHeader>
              <CardContent>
                {isLoadingInsights ? (
                  <div className="text-center py-8">読み込み中...</div>
                ) : insights.length === 0 ? (
                  <div className="text-center py-8 text-gray-500">
                    まだ学習データがありません。異常に回答して学習を開始しましょう。
                  </div>
                ) : (
                  <div className="space-y-3">
                    {insights.map((insight) => (
                      <div
                        key={insight.insight_id}
                        className="p-4 bg-gradient-to-r from-purple-50 to-pink-50 rounded-lg"
                      >
                        <div className="flex items-start justify-between mb-2">
                          <span className="px-2 py-1 bg-purple-500 text-white text-xs rounded">
                            {insight.category}
                          </span>
                          <div className="text-right">
                            <div className="text-xs text-gray-600">信頼度</div>
                            <div className="text-sm font-bold text-purple-600">
                              {(insight.confidence * 100).toFixed(0)}%
                            </div>
                          </div>
                        </div>
                        <div className="text-sm text-gray-800 mb-2">{insight.pattern}</div>
                        <div className="flex items-center justify-between text-xs text-gray-600">
                          <span>{insight.learned_from}件の実績から学習</span>
                          <span className={getImpactColor(insight.average_impact > 0 ? 'positive' : 'negative')}>
                            平均影響: {insight.average_impact > 0 ? '+' : ''}
                            {insight.average_impact.toFixed(1)}%
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>

            {/* 回答履歴 */}
            <Card>
              <CardHeader>
                <CardTitle>📝 回答履歴</CardTitle>
                <CardDescription>過去の回答一覧（最新50件）</CardDescription>
              </CardHeader>
              <CardContent>
                {isLoadingResponses ? (
                  <div className="text-center py-8">読み込み中...</div>
                ) : responses.length === 0 ? (
                  <div className="text-center py-8 text-gray-500">回答履歴がありません</div>
                ) : (
                  <div className="space-y-2 max-h-96 overflow-y-auto">
                    {responses.map((response) => (
                      <div
                        key={response.response_id}
                        className="p-3 bg-gray-50 rounded-lg text-sm"
                      >
                        <div className="flex items-center justify-between mb-1">
                          <span className="font-semibold">{response.anomaly_date}</span>
                          <div className="flex gap-1">
                            {response.tags.map((tag) => (
                              <span
                                key={tag}
                                className="px-2 py-0.5 bg-purple-100 text-purple-700 text-xs rounded"
                              >
                                {tag}
                              </span>
                            ))}
                          </div>
                        </div>
                        <div className="text-gray-600 text-xs mb-1">
                          {response.answer}
                        </div>
                        <div className="flex items-center justify-between text-xs text-gray-500">
                          <span>{new Date(response.timestamp).toLocaleDateString('ja-JP')}</span>
                          <span className={getImpactColor(response.impact)}>
                            {response.impact_value > 0 ? '+' : ''}
                            {response.impact_value.toFixed(1)}%
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </div>
  );
}
