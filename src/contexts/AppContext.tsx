'use client';

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import type { AnomalyDetection } from '@/types/analysis';

export type MessageType = 'normal' | 'anomaly-question';

export interface ChatMessage {
  sender: 'user' | 'ai';
  text: string;
  type?: MessageType;
  anomalyData?: AnomalyDetection;
}

// Contextで共有する値の型定義
interface AppContextType {
  analysisSummary: string;
  setAnalysisSummary: React.Dispatch<React.SetStateAction<string>>;
  chatMessages: ChatMessage[];
  setChatMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>;
}

// Contextの作成（デフォルト値はundefined）
const AppContext = createContext<AppContextType | undefined>(undefined);

// アプリケーション全体をラップするProviderコンポーネント
export function AppProvider({ children }: { children: ReactNode }) {
  // localStorageから初期値を復元
  const [analysisSummary, setAnalysisSummary] = useState<string>('');
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [isHydrated, setIsHydrated] = useState(false);

  // クライアントサイドでのみlocalStorageから復元
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const savedSummary = localStorage.getItem('analysisSummary');
      const savedMessages = localStorage.getItem('chatMessages');
      
      if (savedSummary) {
        setAnalysisSummary(savedSummary);
      }
      
      if (savedMessages) {
        try {
          const parsed = JSON.parse(savedMessages);
          setChatMessages(parsed);
        } catch (e) {
          console.error('Failed to parse chat messages from localStorage:', e);
        }
      }
      
      setIsHydrated(true);
    }
  }, []);

  // analysisSummaryが変更されたらlocalStorageに保存
  useEffect(() => {
    if (isHydrated && typeof window !== 'undefined') {
      if (analysisSummary) {
        localStorage.setItem('analysisSummary', analysisSummary);
      } else {
        localStorage.removeItem('analysisSummary');
      }
    }
  }, [analysisSummary, isHydrated]);

  // chatMessagesが変更されたらlocalStorageに保存
  useEffect(() => {
    if (isHydrated && typeof window !== 'undefined') {
      if (chatMessages.length > 0) {
        localStorage.setItem('chatMessages', JSON.stringify(chatMessages));
      } else {
        localStorage.removeItem('chatMessages');
      }
    }
  }, [chatMessages, isHydrated]);

  const value = {
    analysisSummary,
    setAnalysisSummary,
    chatMessages,
    setChatMessages,
  };

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

// 各コンポーネントでContextを簡単に利用するためのカスタムフック
export function useAppContext() {
  const context = useContext(AppContext);
  if (context === undefined) {
    throw new Error('useAppContext must be used within an AppProvider');
  }
  return context;
}
