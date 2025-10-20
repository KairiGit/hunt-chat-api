'use client';

import { useState, useEffect, useRef } from 'react';
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Textarea } from "@/components/ui/textarea"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { Bot, User, MessageSquare, Info } from 'lucide-react';
import { useAppContext, type ChatMessage, type ContextSource } from '@/contexts/AppContext';

export default function ChatPage() {
  const { analysisSummary, chatMessages, setChatMessages } = useAppContext();

  const [chatInput, setChatInput] = useState('');
  const [chatLoading, setChatLoading] = useState(false);
  const scrollAreaRef = useRef<HTMLDivElement>(null);

  // チャットログが更新されたら一番下にスクロール
  useEffect(() => {
    if (scrollAreaRef.current) {
      const viewport = scrollAreaRef.current.querySelector('[data-radix-scroll-area-viewport]') as HTMLElement;
      if (viewport) {
        viewport.scrollTop = viewport.scrollHeight;
      }
    }
  }, [chatMessages, chatLoading]);

  // 通常のチャット送信処理
  const handleChatSubmit = async () => {
    if (!chatInput.trim() || chatLoading) return;

    const userMessage: ChatMessage = { sender: 'user', text: chatInput };
    const aiEmptyMessage: ChatMessage = { sender: 'ai', text: '', contextSources: [] };
    setChatMessages((prev) => [...prev, userMessage, aiEmptyMessage]);
    setChatLoading(true);
    setChatInput('');

    try {
      const response = await fetch('/api/proxy/chat-input', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          chat_message: chatInput, 
          context: analysisSummary || '' // 分析結果がなくても空文字列で送信
        }),
      });

      if (!response.ok) {
        const errData = await response.json().catch(() => ({ error: `Chat submission failed: ${response.statusText}` }));
        throw new Error(errData.error);
      }

      // まず完全なJSONレスポンスを取得
      const data = await response.json();
      const aiText = data.response?.text || '';
      const contextSources: ContextSource[] = data.response?.context_sources || [];
      
      // AIメッセージを更新
      setChatMessages((prev) => {
        const last = prev[prev.length - 1];
        if (last && last.sender === 'ai') {
          return [...prev.slice(0, -1), { 
            ...last, 
            text: aiText,
            contextSources: contextSources
          }];
        }
        return prev;
      });
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
    if (e.key === 'Enter' && !e.shiftKey && !isComposing) {
      e.preventDefault();
      handleChatSubmit();
    }
  };

  return (
    <div className="container mx-auto p-4 max-w-4xl">
      <Card className="h-[calc(100vh-120px)] flex flex-col">
        <CardHeader className="flex-shrink-0 bg-gradient-to-r from-purple-50 to-indigo-50 dark:from-purple-950/30 dark:to-indigo-950/30 border-b">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-purple-500 rounded-lg">
              <MessageSquare className="h-6 w-6 text-white" />
            </div>
            <div>
              <CardTitle className="text-2xl text-purple-900 dark:text-purple-100">AI分析チャット</CardTitle>
              <CardDescription className="text-purple-700 dark:text-purple-300">
                分析結果について質問したり、過去のデータを検索できます
              </CardDescription>
            </div>
          </div>
        </CardHeader>

        <CardContent className="flex-1 p-0 min-h-0">
          <ScrollArea ref={scrollAreaRef} className="h-full p-4">
            {chatMessages.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-full text-center space-y-4">
                <div className="p-4 bg-purple-100 dark:bg-purple-900/30 rounded-full">
                  <MessageSquare className="h-12 w-12 text-purple-500 dark:text-purple-400" />
                </div>
                <div className="space-y-2">
                  <h3 className="text-lg font-semibold text-purple-900 dark:text-purple-100">
                    {analysisSummary ? '分析結果について質問してください' : 'システムやデータについて質問してください'}
                  </h3>
                  <p className="text-sm text-purple-600 dark:text-purple-400 max-w-md">
                    {analysisSummary 
                      ? 'アップロードしたファイルの分析結果や、過去のデータについてAIに質問できます'
                      : 'システムの使い方、機能、設計について質問したり、データ分析について相談できます'}
                  </p>
                </div>
                <div className="flex flex-wrap gap-2 justify-center">
                  {analysisSummary ? (
                    <>
                      <Button 
                        variant="outline" 
                        size="sm"
                        onClick={() => setChatInput('このデータの傾向を教えて')}
                        className="border-purple-200 text-purple-700 hover:bg-purple-50 dark:border-purple-800 dark:text-purple-300 dark:hover:bg-purple-900/30"
                      >
                        💡 データの傾向を知る
                      </Button>
                      <Button 
                        variant="outline" 
                        size="sm"
                        onClick={() => setChatInput('異常値について詳しく教えて')}
                        className="border-purple-200 text-purple-700 hover:bg-purple-50 dark:border-purple-800 dark:text-purple-300 dark:hover:bg-purple-900/30"
                      >
                        🔍 異常値を調べる
                      </Button>
                      <Button 
                        variant="outline" 
                        size="sm"
                        onClick={() => setChatInput('相関関係を教えて')}
                        className="border-purple-200 text-purple-700 hover:bg-purple-50 dark:border-purple-800 dark:text-purple-300 dark:hover:bg-purple-900/30"
                      >
                        📊 相関を分析
                      </Button>
                    </>
                  ) : (
                    <>
                      <Button 
                        variant="outline" 
                        size="sm"
                        onClick={() => setChatInput('このシステムの機能を教えて')}
                        className="border-purple-200 text-purple-700 hover:bg-purple-50 dark:border-purple-800 dark:text-purple-300 dark:hover:bg-purple-900/30"
                      >
                        🏭 システム機能
                      </Button>
                      <Button 
                        variant="outline" 
                        size="sm"
                        onClick={() => setChatInput('需要予測の仕組みを教えて')}
                        className="border-purple-200 text-purple-700 hover:bg-purple-50 dark:border-purple-800 dark:text-purple-300 dark:hover:bg-purple-900/30"
                      >
                        🔮 予測の仕組み
                      </Button>
                      <Button 
                        variant="outline" 
                        size="sm"
                        onClick={() => setChatInput('APIの使い方を教えて')}
                        className="border-purple-200 text-purple-700 hover:bg-purple-50 dark:border-purple-800 dark:text-purple-300 dark:hover:bg-purple-900/30"
                      >
                        📖 API利用方法
                      </Button>
                    </>
                  )}
                </div>
              </div>
            ) : (
              <div className="space-y-4">
                {chatMessages.map((msg, idx) => (
                  <div key={idx} className={`flex items-start gap-3 ${msg.sender === 'user' ? 'flex-row-reverse' : ''}`}>
                    <Avatar className={msg.sender === 'user' ? 'bg-blue-500' : 'bg-gradient-to-br from-purple-400 to-indigo-500'}>
                      <AvatarFallback className="text-white">
                        {msg.sender === 'user' ? <User className="h-5 w-5" /> : <Bot className="h-5 w-5" />}
                      </AvatarFallback>
                    </Avatar>
                    <div className={`rounded-lg px-4 py-2 max-w-[80%] ${
                      msg.sender === 'user' 
                        ? 'bg-blue-500 text-white ml-auto' 
                        : 'bg-gradient-to-br from-purple-50 to-indigo-50 dark:from-purple-900/30 dark:to-indigo-900/30 text-purple-900 dark:text-purple-100 border border-purple-200 dark:border-purple-800'
                    }`}>
                      <div className="flex items-start justify-between gap-2">
                        <p className="whitespace-pre-wrap text-sm flex-1">{msg.text}</p>
                        {msg.sender === 'ai' && msg.contextSources && msg.contextSources.length > 0 && (
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <button className="flex-shrink-0 text-purple-400 hover:text-purple-600 transition-colors">
                                  <Info className="h-4 w-4" />
                                </button>
                              </TooltipTrigger>
                              <TooltipContent className="max-w-xs">
                                <div className="space-y-1">
                                  <p className="font-semibold text-xs mb-2">📚 参照元ソース</p>
                                  {msg.contextSources.map((source, i) => (
                                    <div key={i} className="text-xs">
                                      <span className="font-medium">
                                        {source.type === 'chat_history' && '💬 '}
                                        {source.type === 'document' && '📄 '}
                                        {source.type === 'analysis_report' && '📊 '}
                                        {source.type === 'file_analysis' && '📁 '}
                                        {source.file_name}
                                      </span>
                                      <span className="text-muted-foreground ml-2">
                                        ({Math.round(source.score * 100)}%)
                                      </span>
                                    </div>
                                  ))}
                                </div>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
                {chatLoading && (
                  <div className="flex items-center gap-2 text-purple-500">
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-purple-500"></div>
                    <span className="text-sm">AIが回答を生成中...</span>
                  </div>
                )}
              </div>
            )}
          </ScrollArea>
        </CardContent>

        <CardFooter className="pt-4 border-t flex-col gap-3 flex-shrink-0">
          <form onSubmit={(e) => { e.preventDefault(); handleChatSubmit(); }} className="flex w-full items-center space-x-2">
            <Textarea
              value={chatInput}
              onChange={(e) => setChatInput(e.target.value)}
              onKeyDown={handleKeyPress}
              onCompositionStart={() => setIsComposing(true)}
              onCompositionEnd={() => setIsComposing(false)}
              placeholder={
                analysisSummary 
                  ? '分析結果について質問... (例: このデータの傾向を教えて)' 
                  : 'システムについて質問... (例: このシステムの機能を教えて)'
              }
              disabled={false}
              className="flex-1 resize-none"
              rows={1}
            />
            <Button 
              type="submit" 
              disabled={!chatInput.trim() || chatLoading}
              className="bg-purple-500 hover:bg-purple-600 text-white"
            >
              送信
            </Button>
          </form>
        </CardFooter>
      </Card>
    </div>
  );
}
