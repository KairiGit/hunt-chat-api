'use client';

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import type { AnomalyDetection } from '@/types/analysis';

export type MessageType = 'normal' | 'anomaly-question';

export interface ContextSource {
  type: string;
  file_name: string;
  score: number;
  date?: string;
}

export interface ChatMessage {
  sender: 'user' | 'ai';
  text: string;
  type?: MessageType;
  anomalyData?: AnomalyDetection;
  contextSources?: ContextSource[]; // 検索ソース情報
}

// 分析チャット用のスレッド型
export interface ChatThread {
  id: string;
  name: string;
  messages: ChatMessage[];
  createdAt: string;
  updatedAt: string;
}

// Contextで共有する値の型定義
interface AppContextType {
  analysisSummary: string;
  setAnalysisSummary: React.Dispatch<React.SetStateAction<string>>;
  
  // 分析チャット用（マルチスレッド対応）
  chatThreads: ChatThread[];
  setChatThreads: React.Dispatch<React.SetStateAction<ChatThread[]>>;
  activeThreadId: string;
  setActiveThreadId: React.Dispatch<React.SetStateAction<string>>;
  
  // 異常対応チャット用（独立したスレッド）
  anomalyChatMessages: ChatMessage[];
  setAnomalyChatMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>;
  
  // 異常対応セッション状態
  anomalyResponseTarget: AnomalyDetection | null;
  setAnomalyResponseTarget: React.Dispatch<React.SetStateAction<AnomalyDetection | null>>;
  anomalyIsWaitingForResponse: boolean;
  setAnomalyIsWaitingForResponse: React.Dispatch<React.SetStateAction<boolean>>;
  anomalyCurrentSessionID: string | null;
  setAnomalyCurrentSessionID: React.Dispatch<React.SetStateAction<string | null>>;
  anomalyUnansweredList: AnomalyDetection[];
  setAnomalyUnansweredList: React.Dispatch<React.SetStateAction<AnomalyDetection[]>>;
  
  // 後方互換性のため残す（使用は非推奨）
  chatMessages: ChatMessage[];
  setChatMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>;
}

// Contextの作成（デフォルト値はundefined）
const AppContext = createContext<AppContextType | undefined>(undefined);

// アプリケーション全体をラップするProviderコンポーネント
export function AppProvider({ children }: { children: ReactNode }) {
  // localStorageから初期値を復元
  const [analysisSummary, setAnalysisSummary] = useState<string>('');
  const [chatThreads, setChatThreads] = useState<ChatThread[]>([]);
  const [activeThreadId, setActiveThreadId] = useState<string>('');
  const [anomalyChatMessages, setAnomalyChatMessages] = useState<ChatMessage[]>([]);
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]); // 後方互換性用
  
  // 異常対応セッション状態
  const [anomalyResponseTarget, setAnomalyResponseTarget] = useState<AnomalyDetection | null>(null);
  const [anomalyIsWaitingForResponse, setAnomalyIsWaitingForResponse] = useState<boolean>(false);
  const [anomalyCurrentSessionID, setAnomalyCurrentSessionID] = useState<string | null>(null);
  const [anomalyUnansweredList, setAnomalyUnansweredList] = useState<AnomalyDetection[]>([]);
  
  const [isHydrated, setIsHydrated] = useState(false);

  // クライアントサイドでのみlocalStorageから復元
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const savedSummary = localStorage.getItem('analysisSummary');
      const savedThreads = localStorage.getItem('analysisChatThreads');
      const savedActiveThreadId = localStorage.getItem('activeThreadId');
      const savedAnomalyMessages = localStorage.getItem('anomalyChatMessages');
      const oldChatMessages = localStorage.getItem('chatMessages');
      
      if (savedSummary) {
        setAnalysisSummary(savedSummary);
      }
      
      // 分析チャットスレッドの復元
      if (savedThreads) {
        try {
          const parsed = JSON.parse(savedThreads);
          setChatThreads(parsed);
          if (savedActiveThreadId) {
            setActiveThreadId(savedActiveThreadId);
          } else if (parsed.length > 0) {
            setActiveThreadId(parsed[0].id);
          }
        } catch (e) {
          console.error('Failed to parse chat threads from localStorage:', e);
          // デフォルトスレッドを作成
          const defaultThread: ChatThread = {
            id: 'thread-1',
            name: 'スレッド 1',
            messages: [],
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
          };
          setChatThreads([defaultThread]);
          setActiveThreadId('thread-1');
        }
      } else if (oldChatMessages) {
        // 既存のchatMessagesからマイグレーション
        try {
          const oldMessages = JSON.parse(oldChatMessages);
          const migratedThread: ChatThread = {
            id: 'thread-1',
            name: 'スレッド 1',
            messages: oldMessages,
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
          };
          setChatThreads([migratedThread]);
          setActiveThreadId('thread-1');
          localStorage.removeItem('chatMessages'); // 旧データを削除
        } catch (e) {
          console.error('Failed to migrate old chat messages:', e);
        }
      } else {
        // 初回起動時：デフォルトスレッドを作成
        const defaultThread: ChatThread = {
          id: 'thread-1',
          name: 'スレッド 1',
          messages: [],
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        };
        setChatThreads([defaultThread]);
        setActiveThreadId('thread-1');
      }
      
      // 異常対応チャットの復元
      if (savedAnomalyMessages) {
        try {
          const parsed = JSON.parse(savedAnomalyMessages);
          setAnomalyChatMessages(parsed);
        } catch (e) {
          console.error('Failed to parse anomaly chat messages from localStorage:', e);
        }
      }
      
      // 異常対応セッション状態の復元
      const savedAnomalySession = localStorage.getItem('anomalySessionState');
      if (savedAnomalySession) {
        try {
          const parsed = JSON.parse(savedAnomalySession);
          if (parsed.responseTarget) setAnomalyResponseTarget(parsed.responseTarget);
          if (parsed.isWaitingForResponse !== undefined) setAnomalyIsWaitingForResponse(parsed.isWaitingForResponse);
          if (parsed.currentSessionID) setAnomalyCurrentSessionID(parsed.currentSessionID);
          if (parsed.unansweredList) setAnomalyUnansweredList(parsed.unansweredList);
        } catch (e) {
          console.error('Failed to parse anomaly session state from localStorage:', e);
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

  // chatThreadsが変更されたらlocalStorageに保存
  useEffect(() => {
    if (isHydrated && typeof window !== 'undefined') {
      if (chatThreads.length > 0) {
        localStorage.setItem('analysisChatThreads', JSON.stringify(chatThreads));
      }
    }
  }, [chatThreads, isHydrated]);

  // activeThreadIdが変更されたらlocalStorageに保存
  useEffect(() => {
    if (isHydrated && typeof window !== 'undefined') {
      if (activeThreadId) {
        localStorage.setItem('activeThreadId', activeThreadId);
      }
    }
  }, [activeThreadId, isHydrated]);

  // anomalyChatMessagesが変更されたらlocalStorageに保存
  useEffect(() => {
    if (isHydrated && typeof window !== 'undefined') {
      if (anomalyChatMessages.length > 0) {
        localStorage.setItem('anomalyChatMessages', JSON.stringify(anomalyChatMessages));
      } else {
        localStorage.removeItem('anomalyChatMessages');
      }
    }
  }, [anomalyChatMessages, isHydrated]);

  // 異常対応セッション状態が変更されたらlocalStorageに保存
  useEffect(() => {
    if (isHydrated && typeof window !== 'undefined') {
      const sessionState = {
        responseTarget: anomalyResponseTarget,
        isWaitingForResponse: anomalyIsWaitingForResponse,
        currentSessionID: anomalyCurrentSessionID,
        unansweredList: anomalyUnansweredList,
      };
      localStorage.setItem('anomalySessionState', JSON.stringify(sessionState));
    }
  }, [anomalyResponseTarget, anomalyIsWaitingForResponse, anomalyCurrentSessionID, anomalyUnansweredList, isHydrated]);

  const value = {
    analysisSummary,
    setAnalysisSummary,
    chatThreads,
    setChatThreads,
    activeThreadId,
    setActiveThreadId,
    anomalyChatMessages,
    setAnomalyChatMessages,
    anomalyResponseTarget,
    setAnomalyResponseTarget,
    anomalyIsWaitingForResponse,
    setAnomalyIsWaitingForResponse,
    anomalyCurrentSessionID,
    setAnomalyCurrentSessionID,
    anomalyUnansweredList,
    setAnomalyUnansweredList,
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
