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

  // ★ 未回答の異常と回答ターゲットのstate
  const [unansweredAnomalies, setUnansweredAnomalies] = useState<AnomalyDetection[]>([]);
  const [responseTarget, setResponseTarget] = useState<AnomalyDetection | null>(null);
  const [isWaitingForResponse, setIsWaitingForResponse] = useState(false);

  // チャットログが更新されたら一番下にスクロール
  useEffect(() => {
    if (scrollAreaRef.current) {
      // ScrollArea内のビューポート要素を取得してスクロール
      const viewport = scrollAreaRef.current.querySelector('[data-radix-scroll-area-viewport]') as HTMLElement;
      if (viewport) {
        viewport.scrollTop = viewport.scrollHeight;
      }
    }
  }, [chatMessages, chatLoading]);

  // ★ 回答を開始する関数
  const startAnswering = () => {
    if (unansweredAnomalies.length === 0) return;
    
    const firstAnomaly = unansweredAnomalies[0];
    setResponseTarget(firstAnomaly);
    setIsWaitingForResponse(true);

    // AIの質問メッセージをチャットに追加
    const questionMessage: ChatMessage = {
      sender: 'ai',
      text: `${firstAnomaly.date} に ${firstAnomaly.product_id} で異常な売上変動を検出しました。\n\n実績値: ${firstAnomaly.actual_value.toFixed(2)}\n予測値: ${firstAnomaly.expected_value.toFixed(2)}\n偏差: ${firstAnomaly.deviation.toFixed(2)}%\n\n${firstAnomaly.ai_question || 'この原因は何だと思いますか？'}`,
      type: 'anomaly-question',
      anomalyData: firstAnomaly,
    };
    setChatMessages((prev) => [...prev, questionMessage]);
  };

  // ★ 選択肢ボタンをクリックした時の処理
  const handleChoiceClick = async (choice: string) => {
    if (!responseTarget) return;

    // 「その他」が選択された場合は、入力欄にフォーカス
    if (choice === 'その他') {
      setChatInput('');
      setIsWaitingForResponse(true);
      
      // プレースホルダーを変更して自由記述を促す
      const textarea = document.querySelector('textarea');
      if (textarea) {
        textarea.placeholder = '異常の原因を詳しく記述してください...';
        textarea.focus();
      }
      return;
    }

    // 定型回答として即座に送信
    await sendAnswer(choice);
  };

  // ★ 回答を送信する共通関数（深掘り質問対応版）
  const [currentSessionID, setCurrentSessionID] = useState<string | null>(null);

  const sendAnswer = async (answer: string) => {
    if (!responseTarget) return;

    // ユーザーメッセージをチャットに追加
    const userMessage: ChatMessage = {
      sender: 'user',
      text: answer,
    };
    setChatMessages((prev) => [...prev, userMessage]);

    const responsePayload = {
      session_id: currentSessionID || undefined, // セッションIDがあれば送信
      anomaly_date: responseTarget.date,
      product_id: responseTarget.product_id,
      question: responseTarget.ai_question || '原因は何だと思いますか？',
      answer: answer,
      answer_type: answer.length > 20 ? 'free_text' : 'choice',
    };

    try {
      // 深掘り対応版のエンドポイントを使用
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
      console.log("[ChatPage] Response from backend:", data);

      // セッションIDを保存
      if (data.session_id) {
        setCurrentSessionID(data.session_id);
      }

      // 深掘り質問が必要か確認
      if (data.needs_follow_up && data.follow_up_question) {
        // 深掘り質問をチャットに追加
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

        // 質問内容を更新（次の回答で使用）
        setResponseTarget({
          ...responseTarget,
          ai_question: data.follow_up_question,
          question_choices: data.follow_up_choices || [],
        });

        // まだ回答待ちを継続
        setIsWaitingForResponse(true);

      } else {
        // 深掘り不要 → この異常の対話は完了
        const thankYouMessage: ChatMessage = {
          sender: 'ai',
          text: data.message || `ご回答ありがとうございます！「${answer}」という情報を学習しました。`,
        };
        setChatMessages((prev) => [...prev, thankYouMessage]);

        // セッションIDをリセット
        setCurrentSessionID(null);

        // 回答済みの異常をリストから削除
        const updatedAnomalies = unansweredAnomalies.filter(
          anomaly => anomaly.date !== responseTarget.date || anomaly.product_id !== responseTarget.product_id
        );
        setUnansweredAnomalies(updatedAnomalies);

        // 次の未回答の異常があれば表示、なければモード終了
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

  // ★ 通常のチャット送信処理
  const handleChatSubmit = async () => {
    if (!chatInput.trim() || chatLoading) return;

    // 回答待ち状態の場合は、回答として処理
    if (isWaitingForResponse && responseTarget) {
      await sendAnswer(chatInput);
      setChatInput('');
      setIsWaitingForResponse(false);
      return;
    }

    // 通常のチャット処理
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
          return prev;
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

  // IME（日本語入力）の状態を追跡
  const [isComposing, setIsComposing] = useState(false);

  const handleKeyPress = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // IME変換中はEnterキーを無視
    if (e.key === 'Enter' && !e.shiftKey && !isComposing) {
      e.preventDefault();
      handleChatSubmit();
    }
  };

  return (
    <div className="flex flex-col h-[calc(100vh-4rem)]">
      <h1 className="text-2xl font-bold mb-4">AIチャット</h1>

      {/* 未回答の異常通知エリア */}
      {unansweredAnomalies.length > 0 && !responseTarget && (
        <Card className="mb-4 border-yellow-500 bg-yellow-50">
          <CardHeader>
            <CardTitle className="text-yellow-800">未回答の異常があります</CardTitle>
            <CardDescription className="text-yellow-700">
              AIが検出した {unansweredAnomalies.length} 件の異常について、原因の特定にご協力ください。
            </CardDescription>
          </CardHeader>
          <CardFooter>
            <Button onClick={startAnswering} className="bg-yellow-600 hover:bg-yellow-700">
              回答を開始する
            </Button>
          </CardFooter>
        </Card>
      )}

      {/* メインのチャットエリア */}
      <Card className="flex-1 flex flex-col min-h-0">
        <CardHeader className="flex-shrink-0">
          <CardTitle>会話</CardTitle>
          <CardDescription>
            {responseTarget 
              ? '異常の原因について回答してください' 
              : analysisSummary 
                ? 'ファイル分析の結果を元に対話できます。' 
                : '先に「ファイル分析」ページでデータを分析してください。'}
          </CardDescription>
        </CardHeader>
          <CardContent className="flex-1 overflow-hidden min-h-0">
            <ScrollArea className="h-full w-full" ref={scrollAreaRef}>
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
        <CardFooter className="pt-4 border-t flex-col gap-3 flex-shrink-0">
          {/* ★ 選択肢ボタンエリア（回答モード時のみ表示） */}
          {responseTarget && responseTarget.question_choices && (
            <div className="w-full flex flex-wrap gap-2 mb-2">
              {responseTarget.question_choices.map((choice, index) => (
                <Button 
                  key={index} 
                  variant={choice === 'その他' ? 'outline' : 'secondary'}
                  onClick={() => handleChoiceClick(choice)}
                  className="flex-1 min-w-[120px]"
                >
                  {choice === 'キャンペーン・販促活動' && '🎯 '}
                  {choice === '天候・気温の影響' && '☀️ '}
                  {choice === 'イベント・行事' && '🎉 '}
                  {choice === 'その他' && '✏️ '}
                  {choice}
                </Button>
              ))}
            </div>
          )}
          
          {/* 入力欄 */}
          <form onSubmit={(e) => { e.preventDefault(); handleChatSubmit(); }} className="flex w-full items-center space-x-2">
            <Textarea
              value={chatInput}
              onChange={(e) => setChatInput(e.target.value)}
              onKeyDown={handleKeyPress}
              onCompositionStart={() => setIsComposing(true)}
              onCompositionEnd={() => setIsComposing(false)}
              placeholder={
                responseTarget 
                  ? '「その他」を選択した場合は、詳しい原因を記述してください...' 
                  : analysisSummary 
                    ? '分析結果について質問... (例: このデータの傾向を教えて)' 
                    : '先にファイルを分析してください'
              }
              disabled={!analysisSummary && !responseTarget}
              className="flex-1 resize-none"
              rows={1}
            />
            <Button type="submit" disabled={chatLoading || (!analysisSummary && !responseTarget) || !chatInput.trim()}>
              送信
            </Button>
          </form>
        </CardFooter>
      </Card>
    </div>
  );
}