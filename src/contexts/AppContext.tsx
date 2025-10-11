'use client';

import React, { createContext, useContext, useState, ReactNode } from 'react';

export interface ChatMessage {
  sender: 'user' | 'ai';
  text: string;
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
  const [analysisSummary, setAnalysisSummary] = useState<string>('');
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);

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
