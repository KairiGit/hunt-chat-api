"use client";

import { useState, useEffect, useRef } from 'react';
import Image from "next/image";

// --- 型定義 ---
// interface DemandForecastSettings { available_regions: { [key: string]: string }; available_products: string[]; forecast_range: { min_days: number; max_days: number }; }
// interface DemandForecastItem { date: string; predicted_demand: number; confidence_level: number; weather_impact: number; seasonal_impact: number; }
// interface DemandForecastResponse { success: boolean; data?: { forecasts: DemandForecastItem[]; region_name: string; product_category: string; }; error?: string; }
interface ChatMessage { sender: 'user' | 'ai'; text: string; }

// --- コンポーネント ---
export default function Home() {
  // --- State定義 ---
  // const [settings, setSettings] = useState<DemandForecastSettings | null>(null);
  // const [forecastResult, setForecastResult] = useState<DemandForecastResponse['data'] | null>(null);
  // const [regionCode, setRegionCode] = useState<string>('240000');
  // const [productCategory, setProductCategory] = useState<string>('飲料');
  // const [forecastDays, setForecastDays] = useState<number>(7);
  // const [loading, setLoading] = useState(false);
  // const [error, setError] = useState<string | null>(null);

  // File Analysis State
  const [selectedFileForAnalysis, setSelectedFileForAnalysis] = useState<File | null>(null);
  const [analysisSummary, setAnalysisSummary] = useState<string>('');
  const [analysisLoading, setAnalysisLoading] = useState(false);
  const [analysisError, setAnalysisError] = useState<string | null>(null);
  const [analysisWarning, setAnalysisWarning] = useState<string | null>(null);
  const analysisFileRef = useRef<HTMLInputElement>(null);

  // Chat State
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [chatInput, setChatInput] = useState('');
  const [chatLoading, setChatLoading] = useState(false);

  // --- データ取得 ---
  useEffect(() => {
    // const fetchSettings = async () => {
    //   try {
    //     const response = await fetch('/api/proxy/demand/settings');
    //     if (!response.ok) throw new Error('Failed to fetch settings.');
    //     const result = await response.json();
    //     if (result.success) {
    //       setSettings(result.data);
    //       if (result.data.available_regions) setRegionCode(Object.keys(result.data.available_regions)[0] || '240000');
    //       if (result.data.available_products) setProductCategory(result.data.available_products[0] || '飲料');
    //     } else {
    //       throw new Error(result.error || 'Failed to parse settings.');
    //     }
    //   } catch (e) {
    //     setError(e instanceof Error ? e.message : 'An unknown error occurred while fetching settings.');
    //   }
    // };
    // fetchSettings();
  }, []);

  // --- ハンドラ関数 ---
  // const handleForecast = async (e: React.FormEvent) => {
  //   e.preventDefault();
  //   setLoading(true);
  //   setError(null);
  //   setForecastResult(null);
  //   try {
  //     const response = await fetch('/api/proxy/demand/forecast', {
  //       method: 'POST',
  //       headers: { 'Content-Type': 'application/json' },
  //       body: JSON.stringify({ region_code: regionCode, product_category: productCategory, forecast_days: forecastDays }),
  //     });
  //     if (!response.ok) {
  //       const errData = await response.json();
  //       throw new Error(errData.error || `HTTP error! status: ${response.status}`);
  //     }
  //     const result: DemandForecastResponse = await response.json();
  //     if (result.success && result.data) setForecastResult(result.data);
  //     else throw new Error(result.error || 'Failed to get forecast data from API');
  //   } catch (e) {
  //     setError(e instanceof Error ? e.message : 'An unknown error occurred');
  //   } finally {
  //     setLoading(false);
  //   }
  // };

  const handleFileAnalysis = async () => {
    if (!selectedFileForAnalysis) return;
    console.log('🔵 [Client] ファイル分析開始:', selectedFileForAnalysis.name);
    setAnalysisLoading(true);
    setAnalysisError(null);
    setAnalysisWarning(null);
    setAnalysisSummary('');
    const formData = new FormData();
    formData.append('file', selectedFileForAnalysis);

    try {
      console.log('🔵 [Client] APIリクエスト送信中...');
      const response = await fetch('/api/proxy/analyze-file', { method: 'POST', body: formData });
      console.log('🔵 [Client] レスポンス受信:', response.status, response.statusText);
      if (!response.ok) {
        const errData = await response.json();
        console.error('🔴 [Client] エラーレスポンス:', errData);
        let detailedError = errData.error || `File analysis failed: ${response.statusText}`;
        if (errData.details && errData.details.error) {
          detailedError = errData.details.error;
        }
        throw new Error(detailedError);
      }
      const result = await response.json();
      console.log('🔵 [Client] 分析結果:', {
        success: result.success,
        hasSummary: !!result.summary,
        hasAnalysisReport: !!result.analysis_report,
        summaryLength: result.summary?.length,
        error: result.error
      });
      if (result.success) {
        setAnalysisSummary(result.summary);
        
        // 詳細レポートがない場合は警告を表示
        if (!result.analysis_report) {
          const warningMessage = result.error 
            ? `基本分析は完了しましたが、詳細レポート生成に失敗: ${result.error}`
            : '基本分析は完了しましたが、詳細レポートは生成されませんでした。';
          console.warn('⚠️ [Client]', warningMessage);
          setAnalysisWarning(warningMessage);
        } else {
          console.log('✅ [Client] 詳細レポートも正常に生成されました');
        }
      } else {
        throw new Error(result.error || 'Failed to get analysis summary.');
      }
    } catch (e) {
      console.error('🔴 [Client] 分析エラー:', e);
      setAnalysisError(e instanceof Error ? e.message : 'An unknown error occurred during analysis.');
    } finally {
      setAnalysisLoading(false);
    }
  };

  const handleChatSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!chatInput.trim() || chatLoading) return;

    const userMessage: ChatMessage = { sender: 'user', text: chatInput };
    // ユーザーのメッセージと、AIの返信用の空のメッセージを先に追加
    setChatMessages((prev) => [...prev, userMessage, { sender: 'ai', text: '' }]);
    setChatLoading(true);
    setChatInput(''); // 入力欄をクリア

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

      // ストリーミング処理
      const reader = response.body?.getReader();
      if (!reader) {
        throw new Error('Failed to get response reader');
      }
      const decoder = new TextDecoder();
      
      while (true) {
        const { done, value } = await reader.read();
        if (done) {
          break;
        }
        const chunk = decoder.decode(value, { stream: true });
        
        // 最後のメッセージ(AIの返信)にテキストを追記していく
        setChatMessages((prevMessages) => {
          const lastMessage = prevMessages[prevMessages.length - 1];
          if (lastMessage && lastMessage.sender === 'ai') {
            const updatedLastMessage = {
              ...lastMessage,
              text: lastMessage.text + chunk,
            };
            return [...prevMessages.slice(0, -1), updatedLastMessage];
          }
          // このパスは通常通らないはず
          return [...prevMessages, { sender: 'ai', text: chunk }];
        });
      }

    } catch (e) {
      const errorMessage = e instanceof Error ? e.message : 'An unknown error occurred';
      setChatMessages((prev) => {
        const lastMessage = prev[prev.length - 1];
        // エラーメッセージでAIの返信を更新
        if (lastMessage && lastMessage.sender === 'ai' && lastMessage.text === '') {
          return [...prev.slice(0, -1), { sender: 'ai', text: `エラー: ${errorMessage}` }];
        }
        return [...prev, { sender: 'ai', text: `エラー: ${errorMessage}` }];
      });
    } finally {
      setChatLoading(false);
    }
  };

  // --- レンダリング ---
  return (
    <div className="font-sans bg-gray-50 dark:bg-gray-900 text-gray-800 dark:text-gray-200 min-h-screen">
      <header className="bg-white dark:bg-gray-800 shadow-md p-4 flex items-center gap-4">
        <Image className="dark:invert" src="/next.svg" alt="Next.js logo" width={120} height={25} priority />
        <h1 className="text-xl font-semibold">需要予測ダッシュボード</h1>
      </header>

      <main className="p-4 sm:p-8">
        <div className="max-w-4xl mx-auto grid grid-cols-1 gap-8">
          {/* --- Section 1: Demand Forecast Form (existing) --- */}
          {/* ... (omitted for brevity, no changes from before) ... */}

          {/* --- Section 2: File Analysis --- */}
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-lg">
            <h2 className="text-2xl font-bold mb-4 border-b pb-2">① 事前ファイル分析</h2>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 items-center">
              <div className="md:col-span-2">
                <label htmlFor="file-analysis-upload" className="block text-sm font-medium mb-1">分析するファイル (.xlsx, .csv)</label>
                <input
                  id="file-analysis-upload"
                  type="file"
                  ref={analysisFileRef}
                  onChange={(e) => setSelectedFileForAnalysis(e.target.files ? e.target.files[0] : null)}
                  className="w-full text-sm text-gray-500 file:mr-4 file:py-2 file:px-4 file:rounded-full file:border-0 file:text-sm file:font-semibold file:bg-blue-50 file:text-blue-700 hover:file:bg-blue-100"
                  accept=".xlsx, .csv"
                />
              </div>
              <button onClick={handleFileAnalysis} disabled={!selectedFileForAnalysis || analysisLoading} className="rounded-md bg-purple-600 text-white px-4 py-2 hover:bg-purple-700 disabled:bg-gray-500 h-10">
                {analysisLoading ? '分析中...' : '分析開始'}
              </button>
            </div>
            {analysisError && <div className="mt-4 p-3 bg-red-100 text-red-800 border border-red-300 rounded-lg"><p className="font-bold">分析エラー:</p><p>{analysisError}</p></div>}
            {analysisWarning && <div className="mt-4 p-3 bg-yellow-100 text-yellow-800 border border-yellow-300 rounded-lg"><p className="font-bold">⚠️ 注意:</p><p>{analysisWarning}</p></div>}
            {analysisSummary && (
              <div className="mt-4">
                <h3 className="font-bold mb-2">分析サマリー</h3>
                <pre className="bg-gray-100 dark:bg-gray-700 p-4 rounded-md text-xs whitespace-pre-wrap">{analysisSummary}</pre>
              </div>
            )}
          </div>

          {/* --- Section 3: AI Chat --- */}
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-lg">
            <h2 className="text-2xl font-bold mb-4 border-b pb-2">② AIチャット (分析結果を利用)</h2>
            <div className="h-80 overflow-y-auto mb-4 p-4 bg-gray-100 dark:bg-gray-700 rounded-md space-y-4">
              {chatMessages.map((msg, index) => (
                <div key={index} className={`flex flex-col ${msg.sender === 'user' ? 'items-end' : 'items-start'}`}>
                  <div className={`rounded-lg px-4 py-2 max-w-lg ${msg.sender === 'user' ? 'bg-blue-500 text-white' : 'bg-gray-300 dark:bg-gray-600 text-gray-900 dark:text-white'}`}>
                    <p>{msg.text}</p>
                  </div>
                </div>
              ))}
              {chatLoading && <div className="flex flex-col items-start"><div className="rounded-lg px-4 py-2 max-w-lg bg-gray-300 dark:bg-gray-600 text-gray-900 dark:text-white animate-pulse">AIが応答を生成中...</div></div>}
            </div>
            <form onSubmit={handleChatSubmit}>
              <div className="flex items-center gap-2">
                <input type="text" value={chatInput} onChange={(e) => setChatInput(e.target.value)} placeholder={analysisSummary ? '分析結果について質問... (例: このデータの傾向を教えて)' : '先にファイルを分析してください'} disabled={!analysisSummary} className="flex-grow p-2 border rounded bg-gray-50 dark:bg-gray-700 border-gray-300 dark:border-gray-600 disabled:cursor-not-allowed" />
                <button type="submit" disabled={chatLoading || !analysisSummary || !chatInput.trim()} className="rounded-md bg-green-600 text-white px-4 py-2 hover:bg-green-700 disabled:bg-gray-500">
                  送信
                </button>
              </div>
            </form>
          </div>

        </div>
      </main>
      <footer className="text-center p-4 text-sm text-gray-500"><p>HUNT Chat-API Demo</p></footer>
    </div>
  );
}