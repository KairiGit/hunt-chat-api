'use client';

import { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { useToast } from '@/components/ui/use-toast';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { AnomalyDetection } from '@/types/analysis';

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
  const [responses, setResponses] = useState<AnomalyResponse[]>([]);
  const [insights, setInsights] = useState<LearningInsight[]>([]);
  const [isLoadingResponses, setIsLoadingResponses] = useState(false);
  const [isLoadingInsights, setIsLoadingInsights] = useState(false);

  // 削除確認ダイアログ用のState
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [isDeleteAllDialogOpen, setDeleteAllDialogOpen] = useState(false);
  const [responseIdToDelete, setResponseIdToDelete] = useState<string | null>(null);

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

  // 削除ダイアログを開く
  const openDeleteDialog = (responseId: string) => {
    setResponseIdToDelete(responseId);
    setIsDeleteDialogOpen(true);
  };

  // 回答を削除
  const deleteResponse = async () => {
    if (!responseIdToDelete) return;

    try {
      const response = await fetch(`/api/proxy/anomaly-responses?id=${responseIdToDelete}`, {
        method: 'DELETE',
      });

      if (!response.ok) throw new Error('削除に失敗しました');

      toast({
        variant: "success",
        title: "✅ 削除完了",
        description: "回答を削除しました",
      });

      // リストを再読み込み
      loadResponses();
      loadInsights();
    } catch (error) {
      console.error('Error:', error);
      toast({
        variant: "destructive",
        title: "エラー",
        description: "削除中にエラーが発生しました",
      });
    } finally {
      setIsDeleteDialogOpen(false);
      setResponseIdToDelete(null);
    }
  };

  // すべての回答を削除
  const deleteAllResponses = async () => {
    try {
      const response = await fetch('/api/proxy/anomaly-responses', {
        method: 'DELETE',
      });

      if (!response.ok) throw new Error('削除に失敗しました');

      toast({
        variant: "success",
        title: "✅ 削除完了",
        description: "すべての学習データを削除しました",
      });

      // リストをクリア
      setResponses([]);
      setInsights([]);
    } catch (error) {
      console.error('Error:', error);
      toast({
        variant: "destructive",
        title: "エラー",
        description: "削除中にエラーが発生しました",
      });
    } finally {
      setDeleteAllDialogOpen(false);
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

  const getImpactColor = (impact: string) => {
    switch (impact) {
      case 'positive': return 'text-green-600';
      case 'negative': return 'text-red-600';
      default: return 'text-gray-600';
    }
  };

  useEffect(() => {
    loadResponses();
    loadInsights();
  }, []);

  return (
    <div className="min-h-screen bg-gradient-to-br from-purple-50 to-pink-50 p-8">
      <div className="max-w-4xl mx-auto space-y-6">
        {/* ヘッダー */}
        <div>
          <h1 className="text-4xl font-bold text-gray-800 mb-2">🧠 AI学習システム</h1>
          <p className="text-gray-600">過去の回答履歴と、それに基づいてAIが学習したパターン（洞察）を確認できます。</p>
        </div>

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
                    まだ学習データがありません。分析ページで異常に回答して学習を開始しましょう。
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
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>📝 回答履歴</CardTitle>
                    <CardDescription>過去の回答一覧（最新50件）</CardDescription>
                  </div>
                  {responses.length > 0 && (
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => setDeleteAllDialogOpen(true)}
                    >
                      🗑️ すべて削除
                    </Button>
                  )}
                </div>
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
                        className="p-3 bg-gray-50 rounded-lg text-sm hover:bg-gray-100 transition-colors"
                      >
                        <div className="flex items-center justify-between mb-1">
                          <span className="font-semibold">{response.anomaly_date}</span>
                          <div className="flex items-center gap-2">
                            <div className="flex gap-1">
                              {response.tags && response.tags.map((tag) => (
                                <span
                                  key={tag}
                                  className="px-2 py-0.5 bg-purple-100 text-purple-700 text-xs rounded"
                                >
                                  {tag}
                                </span>
                              ))}
                            </div>
                            <button
                              onClick={() => openDeleteDialog(response.response_id)}
                              className="text-red-500 hover:text-red-700 hover:bg-red-50 p-1 rounded transition-colors"
                              title="削除"
                            >
                              🗑️
                            </button>
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

        {/* 個別削除ダイアログ */}
        <AlertDialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>本当に削除しますか？</AlertDialogTitle>
              <AlertDialogDescription>
                この操作は取り消せません。この回答を完全に削除します。
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>キャンセル</AlertDialogCancel>
              <AlertDialogAction onClick={deleteResponse} className="bg-red-500 hover:bg-red-600">
                削除
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        {/* 全件削除ダイアログ */}
        <AlertDialog open={isDeleteAllDialogOpen} onOpenChange={setDeleteAllDialogOpen}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>本当によろしいですか？</AlertDialogTitle>
              <AlertDialogDescription>
                すべての学習データが完全に削除されます。この操作は取り消せません。
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>キャンセル</AlertDialogCancel>
              <AlertDialogAction onClick={deleteAllResponses} className="bg-red-500 hover:bg-red-600">
                すべて削除
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

      </div>
    </div>
  );
}