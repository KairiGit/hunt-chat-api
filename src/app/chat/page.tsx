'use client';

import { useState, useEffect, useRef } from 'react';
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Textarea } from "@/components/ui/textarea"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Markdown } from "@/components/ui/markdown"
import { Bot, User, MessageSquare, Info, Plus, X, Edit2, Check } from 'lucide-react';
import { useAppContext, type ChatMessage, type ContextSource, type ChatThread } from '@/contexts/AppContext';

export default function ChatPage() {
  const { 
    analysisSummary, 
    chatThreads, 
    setChatThreads, 
    activeThreadId, 
    setActiveThreadId 
  } = useAppContext();
  
  // アクティブなスレッドを取得
  const activeThread = chatThreads.find(t => t.id === activeThreadId) || chatThreads[0];
  const chatMessages = activeThread?.messages || [];

  const [chatInput, setChatInput] = useState('');
  const [chatLoading, setChatLoading] = useState(false);
  const [editingThreadId, setEditingThreadId] = useState<string | null>(null);
  const [editingThreadName, setEditingThreadName] = useState('');
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

  // アクティブスレッドのメッセージを更新するヘルパー関数
  const updateActiveThreadMessages = (updater: (messages: ChatMessage[]) => ChatMessage[]) => {
    setChatThreads(threads => 
      threads.map(thread => 
        thread.id === activeThreadId
          ? { ...thread, messages: updater(thread.messages), updatedAt: new Date().toISOString() }
          : thread
      )
    );
  };

  // 新しいスレッドを作成
  const createNewThread = () => {
    const threadNumber = chatThreads.length + 1;
    const newThread: ChatThread = {
      id: `thread-${Date.now()}`,
      name: `スレッド ${threadNumber}`,
      messages: [],
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    setChatThreads([...chatThreads, newThread]);
    setActiveThreadId(newThread.id);
  };

  // スレッドを削除
  const deleteThread = (threadId: string) => {
    if (chatThreads.length === 1) return; // 最後のスレッドは削除不可
    const newThreads = chatThreads.filter(t => t.id !== threadId);
    setChatThreads(newThreads);
    if (activeThreadId === threadId) {
      setActiveThreadId(newThreads[0].id);
    }
  };

  // スレッド名を編集
  const startEditingThreadName = (thread: ChatThread) => {
    setEditingThreadId(thread.id);
    setEditingThreadName(thread.name);
  };

  const saveThreadName = () => {
    if (editingThreadId && editingThreadName.trim()) {
      setChatThreads(threads =>
        threads.map(t =>
          t.id === editingThreadId ? { ...t, name: editingThreadName.trim() } : t
        )
      );
    }
    setEditingThreadId(null);
    setEditingThreadName('');
  };

  // 通常のチャット送信処理
  const handleChatSubmit = async () => {
    if (!chatInput.trim() || chatLoading) return;

    const userMessage: ChatMessage = { sender: 'user', text: chatInput };
    const aiEmptyMessage: ChatMessage = { sender: 'ai', text: '', contextSources: [] };
    updateActiveThreadMessages(prev => [...prev, userMessage, aiEmptyMessage]);
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
      updateActiveThreadMessages(prev => {
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
      updateActiveThreadMessages(prev => {
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
        <CardHeader className="flex-shrink-0 bg-gradient-to-r from-purple-50 to-indigo-50 dark:from-purple-950/30 dark:to-indigo-950/30 border-b pb-2">
          <div className="flex items-center justify-between gap-3 mb-3">
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
            <Button 
              variant="outline" 
              size="sm"
              onClick={createNewThread}
              className="flex items-center gap-2 border-purple-200 text-purple-700 hover:bg-purple-50"
            >
              <Plus className="h-4 w-4" />
              新規スレッド
            </Button>
          </div>
          
          {/* スレッド切り替えタブ */}
          <Tabs value={activeThreadId} onValueChange={setActiveThreadId} className="w-full">
            <TabsList className="w-full justify-start overflow-x-auto bg-purple-100/50 dark:bg-purple-900/20">
              {chatThreads.map((thread) => (
                <div key={thread.id} className="flex items-center gap-1 group">
                  {editingThreadId === thread.id ? (
                    <div className="flex items-center gap-1 px-2">
                      <input
                        type="text"
                        value={editingThreadName}
                        onChange={(e) => setEditingThreadName(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') saveThreadName();
                          if (e.key === 'Escape') setEditingThreadId(null);
                        }}
                        className="w-24 px-2 py-1 text-sm border rounded"
                        autoFocus
                      />
                      <button onClick={saveThreadName} className="p-1 hover:bg-purple-200 rounded">
                        <Check className="h-3 w-3 text-green-600" />
                      </button>
                    </div>
                  ) : (
                    <>
                      <TabsTrigger value={thread.id} className="relative">
                        {thread.name}
                      </TabsTrigger>
                      <button
                        onClick={() => startEditingThreadName(thread)}
                        className="p-1 opacity-0 group-hover:opacity-100 hover:bg-purple-200 dark:hover:bg-purple-800 rounded transition-opacity"
                      >
                        <Edit2 className="h-3 w-3" />
                      </button>
                      {chatThreads.length > 1 && (
                        <button
                          onClick={() => deleteThread(thread.id)}
                          className="p-1 opacity-0 group-hover:opacity-100 hover:bg-red-200 dark:hover:bg-red-900 rounded transition-opacity"
                        >
                          <X className="h-3 w-3 text-red-600" />
                        </button>
                      )}
                    </>
                  )}
                </div>
              ))}
            </TabsList>
          </Tabs>
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
                    <div className={`rounded-lg px-4 py-3 max-w-[80%] ${
                      msg.sender === 'user' 
                        ? 'bg-blue-500 text-white ml-auto' 
                        : 'bg-gradient-to-br from-purple-50 to-indigo-50 dark:from-purple-900/30 dark:to-indigo-900/30 text-purple-900 dark:text-purple-100 border border-purple-200 dark:border-purple-800'
                    }`}>
                      <div className="flex items-start justify-between gap-2">
                        {msg.sender === 'user' ? (
                          <p className="whitespace-pre-wrap text-sm flex-1">{msg.text}</p>
                        ) : (
                          <div className="flex-1 text-sm">
                            <Markdown>{msg.text}</Markdown>
                          </div>
                        )}
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
