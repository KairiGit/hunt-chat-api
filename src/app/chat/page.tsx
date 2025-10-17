'use client';

import { useState, useEffect, useRef } from 'react';
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Textarea } from "@/components/ui/textarea"
import { Bot, User } from 'lucide-react';
import { useAppContext, type ChatMessage } from '@/contexts/AppContext'; // ★ AppContextと型をインポート

// AnomalyDetectionの型をインポート
import type { AnomalyDetection } from '@/types/analysis';

// ChatMessageの型定義はAppContextで共有されているので不要

export default function ChatPage() {
  // ★ useAppContextから共有のstateと更新関数を取得
  const { analysisSummary, chatMessages, setChatMessages } = useAppContext();

  const [chatInput, setChatInput] = useState('');
  const [chatLoading, setChatLoading] = useState(false);
  const scrollAreaRef = useRef<HTMLDivElement>(null);

  // ★ 未回答の異常と回答モードのstateを追加
  const [unansweredAnomalies, setUnansweredAnomalies] = useState<AnomalyDetection[]>([]);
  const [isAnomalyResponseMode, setIsAnomalyResponseMode] = useState(false);
  const [currentAnomaly, setCurrentAnomaly] = useState<AnomalyDetection | null>(null);

  // チャットログが更新されたら一番下にスクロール
  useEffect(() => {
    if (scrollAreaRef.current) {
      scrollAreaRef.current.scrollTop = scrollAreaRef.current.scrollHeight;
    }
  }, [chatMessages, chatLoading]);

  // ★ コンポーネントのマウント時に未回答の異常を取得
  useEffect(() => {
    const fetchUnansweredAnomalies = async () => {
      try {
        const response = await fetch('/api/proxy/unanswered-anomalies');
        if (!response.ok) {
          throw new Error('Failed to fetch unanswered anomalies');
        }
        const data = await response.json();
        console.log("[ChatPage] Fetched unanswered anomalies:", data);
        if (data.success && data.anomalies) {
          setUnansweredAnomalies(data.anomalies);
        } else {
          // データがnullの場合でも空の配列をセットしてエラーを防ぐ
          setUnansweredAnomalies([]);
        }
      } catch (error) {
        console.error('Error fetching unanswered anomalies:', error);
        setUnansweredAnomalies([]);
      }
    };

    fetchUnansweredAnomalies();
  }, [analysisSummary]); // ★ analysisSummaryを依存配列に追加

  const handleChatSubmit = async () => {
    if (!chatInput.trim() || chatLoading) return;

    const userMessage: ChatMessage = { sender: 'user', text: chatInput };
    const aiEmptyMessage: ChatMessage = { sender: 'ai', text: '' };
    setChatMessages((prev) => [...prev, userMessage, aiEmptyMessage]);
    setChatLoading(true);
    setChatInput('');

    try {
      const response = await fetch('/api/proxy/chat-input', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ chat_message: chatInput, context: analysisSummary }),
      });

      if (!response.ok) {
        const errData = await response.json().catch(() => ({ error: `Chat submission failed: ${response.statusText}` }));
        throw new Error(errData.error);
      }

      const reader = response.body?.getReader();
      if (!reader) throw new Error('Failed to get response reader');
      
      const decoder = new TextDecoder();
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        const chunk = decoder.decode(value, { stream: true });
        setChatMessages((prev) => {
          const last = prev[prev.length - 1];
          if (last && last.sender === 'ai') {
            return [...prev.slice(0, -1), { ...last, text: last.text + chunk }];
          }
          return prev; // 予期しないケースでは状態を更新しない
        });
      }
    } catch (e) {
      const errorMessage = e instanceof Error ? e.message : 'An unknown error occurred';
      setChatMessages((prev) => {
        const last = prev[prev.length - 1];
        if (last && last.sender === 'ai' && last.text === '') {
          return [...prev.slice(0, -1), { sender: 'ai', text: `エラー: ${errorMessage}` }];
        }
        return [...prev, { sender: 'ai', text: `エラー: ${errorMessage}` }];
      });
    } finally {
      setChatLoading(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleChatSubmit();
    }
  };

  // ★ 異常への回答を処理する関数
  const handleAnomalyResponse = async (choice: string) => {
    if (!currentAnomaly) return;

    const responsePayload = {
      anomaly_date: currentAnomaly.date,
      product_id: currentAnomaly.product_id,
      question: currentAnomaly.ai_question,
      answer: choice,
      answer_type: 'multiple_choice',
      tags: [choice], // 回答をタグとして保存
      impact: 'unknown', // 現時点では影響不明
      impact_value: 0,
    };

    try {
      const response = await fetch('/api/proxy/anomaly-response', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(responsePayload),
      });

      if (!response.ok) {
        const errData = await response.json().catch(() => ({ error: 'Failed to save anomaly response' }));
        throw new Error(errData.error);
      }

      const result = await response.json();

      // AIからの感謝メッセージをチャットに追加
      const thankYouMessage: ChatMessage = {
        sender: 'ai',
        text: `ご回答ありがとうございます！「${choice}」という情報を学習しました。今後の分析精度が向上します。`,
      };
      setChatMessages((prev) => [...prev, thankYouMessage]);

      // 回答済みの異常をリストから削除
      const updatedAnomalies = unansweredAnomalies.filter(anomaly => anomaly.date !== currentAnomaly.date || anomaly.product_id !== currentAnomaly.product_id);
      setUnansweredAnomalies(updatedAnomalies);

      // 次の未回答の異常があれば表示、なければモード終了
      if (updatedAnomalies.length > 0) {
        setCurrentAnomaly(updatedAnomalies[0]);
      } else {
        setIsAnomalyResponseMode(false);
        setCurrentAnomaly(null);
      }

    } catch (e) {
      const errorMessage = e instanceof Error ? e.message : 'An unknown error occurred';
      const errorResponseMessage: ChatMessage = {
        sender: 'ai',
        text: `エラー: 回答の保存中に問題が発生しました。(${errorMessage})`,
      };
      setChatMessages((prev) => [...prev, errorResponseMessage]);
    }
  };

  return (
    <div className="flex flex-col h-[calc(100vh-4rem)]">
      <h1 className="text-2xl font-bold mb-4">AIチャット</h1>

      {/* 未回答の異常通知エリア */}
      {unansweredAnomalies.length > 0 && !isAnomalyResponseMode && (
        <Card className="mb-4 border-yellow-500 bg-yellow-50">
          <CardHeader>
            <CardTitle className="text-yellow-800">未回答の異常があります</CardTitle>
            <CardDescription className="text-yellow-700">
              AIが検出した {unansweredAnomalies.length} 件の異常について、原因の特定にご協力ください。
            </CardDescription>
          </CardHeader>
          <CardFooter>
            <Button onClick={() => {
              console.log("Entering anomaly response mode with:", unansweredAnomalies[0]);
              setCurrentAnomaly(unansweredAnomalies[0]);
              setIsAnomalyResponseMode(true);
            }} className="bg-yellow-600 hover:bg-yellow-700">
              回答を開始する
            </Button>
          </CardFooter>
        </Card>
      )}

      {/* 異常回答モードのUI */}
      {isAnomalyResponseMode && currentAnomaly ? (
        <Card className="mb-4 border-blue-500">
          <CardHeader>
            <CardTitle>異常の原因分析</CardTitle>
            <CardDescription>
              {currentAnomaly.date} の {currentAnomaly.product_id} の売上 (実績: {currentAnomaly.actual_value.toFixed(2)}, 予測: {currentAnomaly.expected_value.toFixed(2)}) の異常について、最も可能性の高い原因を選択してください。
            </CardDescription>
          </CardHeader>
          <CardContent>
            <p className="font-semibold mb-4">{currentAnomaly.ai_question}</p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
              {currentAnomaly.question_choices?.map((choice, index) => (
                <Button key={index} variant="outline" onClick={() => handleAnomalyResponse(choice)}>
                  {choice}
                </Button>
              ))}
               <Button variant="destructive" onClick={() => setIsAnomalyResponseMode(false)}>
                キャンセル
              </Button>
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card className="flex-1 flex flex-col">
          <CardHeader>
            <CardTitle>会話</CardTitle>
            <CardDescription>
              {analysisSummary 
                ? 'ファイル分析の結果を元に対話できます。' 
                : '先に「ファイル分析」ページでデータを分析してください。'}
            </CardDescription>
          </CardHeader>
          <CardContent className="flex-1 overflow-hidden">
            <ScrollArea className="h-full" ref={scrollAreaRef}>
              <div className="space-y-6 pr-4">
                {chatMessages.map((msg, index) => (
                  <div key={index} className={`flex items-start gap-4 ${msg.sender === 'user' ? 'justify-end' : ''}`}>
                    {msg.sender === 'ai' && (
                      <Avatar className="w-8 h-8">
                        <AvatarFallback><Bot size={16} /></AvatarFallback>
                      </Avatar>
                    )}
                    <div className={`rounded-lg px-4 py-2 max-w-[80%] whitespace-pre-wrap ${msg.sender === 'user' ? 'bg-blue-600 text-white' : 'bg-gray-100 dark:bg-gray-800'}`}>
                      {msg.text}
                    </div>
                    {msg.sender === 'user' && (
                       <Avatar className="w-8 h-8">
                        <AvatarFallback><User size={16} /></AvatarFallback>
                      </Avatar>
                    )}
                  </div>
                ))}
                {chatLoading && chatMessages[chatMessages.length - 1]?.sender === 'user' && (
                   <div className="flex items-start gap-4">
                      <Avatar className="w-8 h-8">
                        <AvatarFallback><Bot size={16} /></AvatarFallback>
                      </Avatar>
                      <div className="rounded-lg px-4 py-2 max-w-[80%] bg-gray-100 dark:bg-gray-800 animate-pulse">
                        ... 
                      </div>
                  </div>
                )}
              </div>
            </ScrollArea>
          </CardContent>
          <CardFooter className="pt-4 border-t">
            <form onSubmit={(e) => { e.preventDefault(); handleChatSubmit(); }} className="flex w-full items-center space-x-2">
              <Textarea
                value={chatInput}
                onChange={(e) => setChatInput(e.target.value)}
                onKeyDown={handleKeyPress}
                placeholder={analysisSummary ? '分析結果について質問... (例: このデータの傾向を教えて)' : '先にファイルを分析してください'}
                disabled={!analysisSummary}
                className="flex-1 resize-none"
                rows={1}
              />
              <Button type="submit" disabled={chatLoading || !analysisSummary || !chatInput.trim()}>
                送信
              </Button>
            </form>
          </CardFooter>
        </Card>
      )}
    </div>
  );
}
