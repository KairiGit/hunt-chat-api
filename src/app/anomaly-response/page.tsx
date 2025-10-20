'use client';

import { useState, useEffect, useRef } from 'react';
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Textarea } from "@/components/ui/textarea"
import { AlertCircle, Bot, User } from 'lucide-react';
import { useAppContext, type ChatMessage } from '@/contexts/AppContext';
import type { AnomalyDetection } from '@/types/analysis';

export default function AnomalyResponsePage() {
  const { chatMessages, setChatMessages } = useAppContext();

  const [chatInput, setChatInput] = useState('');
  const scrollAreaRef = useRef<HTMLDivElement>(null);

  // 異常対応専用のstate
  const [unansweredAnomalies, setUnansweredAnomalies] = useState<AnomalyDetection[]>([]);
  const [responseTarget, setResponseTarget] = useState<AnomalyDetection | null>(null);
  const [isWaitingForResponse, setIsWaitingForResponse] = useState(false);
  const [currentSessionID, setCurrentSessionID] = useState<string | null>(null);

  // チャットログが更新されたら一番下にスクロール
  useEffect(() => {
    if (scrollAreaRef.current) {
      const viewport = scrollAreaRef.current.querySelector('[data-radix-scroll-area-viewport]') as HTMLElement;
      if (viewport) {
        viewport.scrollTop = viewport.scrollHeight;
      }
    }
  }, [chatMessages]);

  // 回答を開始する関数
  const startAnswering = () => {
    if (unansweredAnomalies.length === 0) return;
    
    const firstAnomaly = unansweredAnomalies[0];
    setResponseTarget(firstAnomaly);
    setIsWaitingForResponse(true);

    const questionMessage: ChatMessage = {
      sender: 'ai',
      text: `${firstAnomaly.date} に ${firstAnomaly.product_id} で異常な売上変動を検出しました。\n\n実績値: ${firstAnomaly.actual_value.toFixed(2)}\n予測値: ${firstAnomaly.expected_value.toFixed(2)}\n偏差: ${firstAnomaly.deviation.toFixed(2)}%\n\n${firstAnomaly.ai_question || 'この原因は何だと思いますか？'}`,
      type: 'anomaly-question',
      anomalyData: firstAnomaly,
    };
    setChatMessages((prev) => [...prev, questionMessage]);
  };

  // 選択肢ボタンをクリックした時の処理
  const handleChoiceClick = async (choice: string) => {
    if (!responseTarget) return;
    await sendAnswer(choice);
  };

  // 回答を送信する共通関数（深掘り質問対応版）
  const sendAnswer = async (answer: string) => {
    if (!responseTarget) return;

    const userMessage: ChatMessage = {
      sender: 'user',
      text: answer,
    };
    setChatMessages((prev) => [...prev, userMessage]);

    const responsePayload = {
      session_id: currentSessionID || undefined,
      anomaly_date: responseTarget.date,
      product_id: responseTarget.product_id,
      question: responseTarget.ai_question || '原因は何だと思いますか？',
      answer: answer,
      answer_type: answer.length > 20 ? 'free_text' : 'choice',
    };

    try {
      const response = await fetch('/api/proxy/anomaly-response-with-followup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(responsePayload),
      });

      if (!response.ok) {
        const errData = await response.json().catch(() => ({ error: 'Failed to save anomaly response' }));
        throw new Error(errData.error);
      }

      const data = await response.json();

      if (data.session_id) {
        setCurrentSessionID(data.session_id);
      }

      // 深掘り質問が必要か確認
      if (data.needs_follow_up && data.follow_up_question) {
        const followUpMessage: ChatMessage = {
          sender: 'ai',
          text: `${data.message}\n\n${data.follow_up_question}`,
          type: 'anomaly-question',
          anomalyData: {
            ...responseTarget,
            ai_question: data.follow_up_question,
            question_choices: data.follow_up_choices || [],
          },
        };
        setChatMessages((prev) => [...prev, followUpMessage]);

        setResponseTarget({
          ...responseTarget,
          ai_question: data.follow_up_question,
          question_choices: data.follow_up_choices || [],
        });

        setIsWaitingForResponse(true);

      } else {
        const thankYouMessage: ChatMessage = {
          sender: 'ai',
          text: data.message || `ご回答ありがとうございます！「${answer}」という情報を学習しました。`,
        };
        setChatMessages((prev) => [...prev, thankYouMessage]);

        setCurrentSessionID(null);

        const updatedAnomalies = unansweredAnomalies.filter(
          anomaly => anomaly.date !== responseTarget.date || anomaly.product_id !== responseTarget.product_id
        );
        setUnansweredAnomalies(updatedAnomalies);

        if (updatedAnomalies.length > 0) {
          const nextAnomaly = updatedAnomalies[0];
          setResponseTarget(nextAnomaly);
          
          const nextQuestionMessage: ChatMessage = {
            sender: 'ai',
            text: `続いて、${nextAnomaly.date} の ${nextAnomaly.product_id} について教えてください。\n\n実績値: ${nextAnomaly.actual_value.toFixed(2)}\n予測値: ${nextAnomaly.expected_value.toFixed(2)}\n\n${nextAnomaly.ai_question || 'この原因は何だと思いますか？'}`,
            type: 'anomaly-question',
            anomalyData: nextAnomaly,
          };
          setChatMessages((prev) => [...prev, nextQuestionMessage]);
        } else {
          setResponseTarget(null);
          setIsWaitingForResponse(false);
          
          const completionMessage: ChatMessage = {
            sender: 'ai',
            text: 'すべての異常について回答いただきました。ありがとうございました！学習したデータは今後の分析に活用されます。',
          };
          setChatMessages((prev) => [...prev, completionMessage]);
        }
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

  // コンポーネントのマウント時に未回答の異常を取得
  useEffect(() => {
    const fetchUnansweredAnomalies = async () => {
      try {
        const response = await fetch('/api/proxy/unanswered-anomalies');
        if (!response.ok) {
          throw new Error('Failed to fetch unanswered anomalies');
        }
        const data = await response.json();
        
        if (data.success && data.anomalies) {
          setUnansweredAnomalies(data.anomalies);
        }
      } catch (error) {
        console.error('[AnomalyResponsePage] Failed to fetch unanswered anomalies:', error);
      }
    };

    fetchUnansweredAnomalies();
  }, []);

  // 通常のチャット送信
  const handleChatSubmit = async () => {
    if (!chatInput.trim() || !responseTarget) return;
    await sendAnswer(chatInput.trim());
    setChatInput('');
  };

  // IME（日本語入力）の状態を追跡
  const [isComposing, setIsComposing] = useState(false);

  const handleKeyPress = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey && !isComposing) {
      e.preventDefault();
      handleChatSubmit();
    }
  };

  return (
    <div className="container mx-auto p-4 max-w-4xl">
      <Card className="h-[calc(100vh-120px)] flex flex-col">
        <CardHeader className="flex-shrink-0 bg-gradient-to-r from-amber-50 to-orange-50 dark:from-amber-950/30 dark:to-orange-950/30 border-b">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-amber-500 rounded-lg">
              <AlertCircle className="h-6 w-6 text-white" />
            </div>
            <div>
              <CardTitle className="text-2xl text-amber-900 dark:text-amber-100">異常対応チャット</CardTitle>
              <CardDescription className="text-amber-700 dark:text-amber-300">
                検出された異常について原因を教えてください
              </CardDescription>
            </div>
          </div>
          
          {unansweredAnomalies.length > 0 && !isWaitingForResponse && (
            <div className="mt-4">
              <Button 
                onClick={startAnswering}
                className="w-full bg-amber-500 hover:bg-amber-600 text-white"
              >
                <AlertCircle className="mr-2 h-4 w-4" />
                異常に回答する ({unansweredAnomalies.length}件)
              </Button>
            </div>
          )}
        </CardHeader>

        <CardContent className="flex-1 p-0 min-h-0">
          <ScrollArea ref={scrollAreaRef} className="h-full p-4">
            <div className="space-y-4">
              {chatMessages.map((msg, idx) => (
                <div key={idx} className={`flex items-start gap-3 ${msg.sender === 'user' ? 'flex-row-reverse' : ''}`}>
                  <Avatar className={msg.sender === 'user' ? 'bg-blue-500' : 'bg-gradient-to-br from-amber-400 to-orange-500'}>
                    <AvatarFallback className="text-white">
                      {msg.sender === 'user' ? <User className="h-5 w-5" /> : <Bot className="h-5 w-5" />}
                    </AvatarFallback>
                  </Avatar>
                  <div className={`rounded-lg px-4 py-2 max-w-[80%] ${
                    msg.sender === 'user' 
                      ? 'bg-blue-500 text-white ml-auto' 
                      : 'bg-gradient-to-br from-amber-50 to-orange-50 dark:from-amber-900/30 dark:to-orange-900/30 text-amber-900 dark:text-amber-100 border border-amber-200 dark:border-amber-800'
                  }`}>
                    <p className="whitespace-pre-wrap text-sm">{msg.text}</p>
                  </div>
                </div>
              ))}
            </div>
          </ScrollArea>
        </CardContent>

        <CardFooter className="pt-4 border-t flex-col gap-3 flex-shrink-0">
          {responseTarget && responseTarget.question_choices && (
            <div className="w-full flex flex-wrap gap-2 mb-2">
              {responseTarget.question_choices
                .filter(choice => !choice.includes('その他') && !choice.includes('自由記述'))
                .map((choice, index) => (
                  <Button 
                    key={index} 
                    variant="secondary"
                    onClick={() => handleChoiceClick(choice)}
                    className="flex-1 min-w-[120px] bg-amber-100 hover:bg-amber-200 text-amber-900 dark:bg-amber-900/30 dark:hover:bg-amber-900/50 dark:text-amber-100"
                  >
                    {choice === 'キャンペーン・販促活動' && '🎯 '}
                    {choice === '天候・気温の影響' && '☀️ '}
                    {choice === 'イベント・行事' && '🎉 '}
                    {choice}
                  </Button>
                ))}
            </div>
          )}
          
          <form onSubmit={(e) => { e.preventDefault(); handleChatSubmit(); }} className="flex w-full items-center space-x-2">
            <Textarea
              value={chatInput}
              onChange={(e) => setChatInput(e.target.value)}
              onKeyDown={handleKeyPress}
              onCompositionStart={() => setIsComposing(true)}
              onCompositionEnd={() => setIsComposing(false)}
              placeholder={
                responseTarget 
                  ? '💡 上記以外の場合は、こちらに詳しく記述してください...' 
                  : '異常への回答を開始してください'
              }
              disabled={!responseTarget}
              className="flex-1 resize-none"
              rows={1}
            />
            <Button 
              type="submit" 
              disabled={!chatInput.trim() || !responseTarget}
              className="bg-amber-500 hover:bg-amber-600 text-white"
            >
              送信
            </Button>
          </form>
        </CardFooter>
      </Card>
    </div>
  );
}
