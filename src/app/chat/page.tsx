'use client';

import { useState, useEffect, useRef } from 'react';
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Textarea } from "@/components/ui/textarea"
import { Bot, User } from 'lucide-react';
import { useAppContext, type ChatMessage } from '@/contexts/AppContext'; // ★ AppContextと型をインポート

// ChatMessageの型定義はAppContextで共有されているので不要

export default function ChatPage() {
  // ★ useAppContextから共有のstateと更新関数を取得
  const { analysisSummary, chatMessages, setChatMessages } = useAppContext();

  const [chatInput, setChatInput] = useState('');
  const [chatLoading, setChatLoading] = useState(false);
  const scrollAreaRef = useRef<HTMLDivElement>(null);

  // チャットログが更新されたら一番下にスクロール
  useEffect(() => {
    if (scrollAreaRef.current) {
      scrollAreaRef.current.scrollTop = scrollAreaRef.current.scrollHeight;
    }
  }, [chatMessages, chatLoading]);

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

  return (
    <div className="flex flex-col h-[calc(100vh-4rem)]">
      <h1 className="text-2xl font-bold mb-4">AIチャット</h1>
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
    </div>
  );
}
