"use client";

import { useState, useEffect, useRef } from 'react';
import Image from "next/image";

// --- 型定義 ---

// 設定情報の型
interface DemandForecastSettings {
  available_regions: { [key: string]: string };
  available_products: string[];
  forecast_range: { min_days: number; max_days: number };
}

// 予測結果のアイテムの型
interface DemandForecastItem {
  date: string;
  predicted_demand: number;
  confidence_level: number;
  weather_impact: number;
  seasonal_impact: number;
}

// APIレスポンス（予測結果）の型
interface DemandForecastResponse {
  success: boolean;
  data?: {
    forecasts: DemandForecastItem[];
    region_name: string;
    product_category: string;
  };
  error?: string;
}

// チャットメッセージの型
interface ChatMessage {
  sender: 'user' | 'ai';
  text: string;
  fileName?: string;
}

// --- コンポーネント ---

export default function Home() {
  // --- State定義 ---
  // 予測フォーム用
  const [settings, setSettings] = useState<DemandForecastSettings | null>(null);
  const [forecastResult, setForecastResult] = useState<DemandForecastResponse['data'] | null>(null);
  const [regionCode, setRegionCode] = useState<string>('240000');
  const [productCategory, setProductCategory] = useState<string>('飲料');
  const [forecastDays, setForecastDays] = useState<number>(7);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // チャット用
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [chatInput, setChatInput] = useState('');
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [chatLoading, setChatLoading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // --- データ取得 ---

  useEffect(() => {
    const fetchSettings = async () => {
      try {
        const response = await fetch('/api/demand-forecast');
        if (!response.ok) {
          throw new Error('Failed to fetch settings.');
        }
        const result = await response.json();
        if (result.success) {
          setSettings(result.data);
          if (result.data.available_regions) {
            setRegionCode(Object.keys(result.data.available_regions)[0] || '240000');
          }
          if (result.data.available_products) {
            setProductCategory(result.data.available_products[0] || '飲料');
          }
        } else {
          throw new Error(result.error || 'Failed to parse settings.');
        }
      } catch (e) {
        setError(e instanceof Error ? e.message : 'An unknown error occurred while fetching settings.');
      }
    };

    fetchSettings();
  }, []);

  const handleForecast = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setForecastResult(null);

    try {
      const response = await fetch('/api/demand-forecast', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          region_code: regionCode,
          product_category: productCategory,
          forecast_days: forecastDays,
        }),
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || `HTTP error! status: ${response.status}`);
      }

      const result: DemandForecastResponse = await response.json();
      if (result.success && result.data) {
        setForecastResult(result.data);
      } else {
        throw new Error(result.error || 'Failed to get forecast data from API');
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'An unknown error occurred');
    } finally {
      setLoading(false);
    }
  };

  // --- チャット処理 ---
  const handleChatSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!chatInput.trim() && !selectedFile) return;

    const userMessage: ChatMessage = {
      sender: 'user',
      text: chatInput,
      fileName: selectedFile?.name,
    };
    setChatMessages((prev) => [...prev, userMessage]);
    setChatLoading(true);
    setError(null);

    const formData = new FormData();
    formData.append('chat_message', chatInput);
    if (selectedFile) {
      formData.append('file', selectedFile);
    }

    try {
      const response = await fetch('/api/chat', {
        method: 'POST',
        body: formData,
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || `HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      if (result.success) {
        const aiMessage: ChatMessage = {
          sender: 'ai',
          text: result.response.text,
        };
        setChatMessages((prev) => [...prev, aiMessage]);
      } else {
        throw new Error(result.error || 'Failed to get chat response from API');
      }
    } catch (e) {
      const errorMessage = e instanceof Error ? e.message : 'An unknown error occurred';
      setError(errorMessage);
      const aiErrorMessage: ChatMessage = {
        sender: 'ai',
        text: `エラーが発生しました: ${errorMessage}`,
      };
      setChatMessages((prev) => [...prev, aiErrorMessage]);
    } finally {
      setChatLoading(false);
      setChatInput('');
      setSelectedFile(null);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  // --- レンダリング ---
  return (
    <div className="font-sans bg-gray-50 dark:bg-gray-900 text-gray-800 dark:text-gray-200 min-h-screen">
      <header className="bg-white dark:bg-gray-800 shadow-md p-4 flex items-center gap-4">
        <Image
          className="dark:invert"
          src="/next.svg"
          alt="Next.js logo"
          width={120}
          height={25}
          priority
        />
        <h1 className="text-xl font-semibold">需要予測ダッシュボード</h1>
      </header>

      <main className="p-4 sm:p-8">
        <div className="max-w-4xl mx-auto grid grid-cols-1 gap-8">
          
          {/* --- 予測条件フォーム --- */}
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-lg mb-8">
            <h2 className="text-2xl font-bold mb-4 border-b pb-2">予測条件</h2>
            {settings ? (
              <form onSubmit={handleForecast} className="grid grid-cols-1 md:grid-cols-3 gap-4 items-end">
                <div>
                  <label htmlFor="region" className="block text-sm font-medium mb-1">地域</label>
                  <select
                    id="region"
                    value={regionCode}
                    onChange={(e) => setRegionCode(e.target.value)}
                    className="w-full p-2 border rounded bg-gray-50 dark:bg-gray-700 border-gray-300 dark:border-gray-600"
                  >
                    {Object.entries(settings.available_regions).map(([code, name]) => (
                      <option key={code} value={code}>{name}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label htmlFor="product" className="block text-sm font-medium mb-1">製品カテゴリ</label>
                  <select
                    id="product"
                    value={productCategory}
                    onChange={(e) => setProductCategory(e.target.value)}
                    className="w-full p-2 border rounded bg-gray-50 dark:bg-gray-700 border-gray-300 dark:border-gray-600"
                  >
                    {settings.available_products.map((product) => (
                      <option key={product} value={product}>{product}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label htmlFor="days" className="block text-sm font-medium mb-1">予測日数</label>
                  <input
                    type="number"
                    id="days"
                    value={forecastDays}
                    onChange={(e) => setForecastDays(Number(e.target.value))}
                    min={settings.forecast_range.min_days}
                    max={settings.forecast_range.max_days}
                    className="w-full p-2 border rounded bg-gray-50 dark:bg-gray-700 border-gray-300 dark:border-gray-600"
                  />
                </div>
                <div className="md:col-span-3 text-center mt-4">
                  <button
                    type="submit"
                    disabled={loading}
                    className="rounded-md border border-solid border-transparent transition-colors flex items-center justify-center bg-blue-600 text-white gap-2 hover:bg-blue-700 font-medium text-base h-12 px-8 w-full sm:w-auto disabled:bg-gray-500 disabled:cursor-not-allowed"
                  >
                    {loading ? '予測中...' : '需要を予測する'}
                  </button>
                </div>
              </form>
            ) : (
              <p>設定を読み込み中...</p>
            )}
          </div>

          {/* --- エラー表示 --- */}
          {error && (
            <div className="my-4 p-4 bg-red-100 text-red-800 border border-red-300 rounded-lg w-full">
              <p className="font-bold">エラーが発生しました:</p>
              <p>{error}</p>
            </div>
          )}

          {/* --- 予測結果 --- */}
          {forecastResult && (
            <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-lg">
              <h2 className="text-2xl font-bold mb-4">
                予測結果: {forecastResult.region_name} - {forecastResult.product_category}
              </h2>
              <div className="overflow-x-auto">
                <table className="min-w-full text-left">
                  <thead className="bg-gray-100 dark:bg-gray-700">
                    <tr>
                      <th className="p-3">日付</th>
                      <th className="p-3 text-right">予測需要量</th>
                      <th className="p-3 text-right">信頼度</th>
                      <th className="p-3 text-right">気象影響</th>
                      <th className="p-3 text-right">季節影響</th>
                    </tr>
                  </thead>
                  <tbody>
                    {forecastResult.forecasts.map((item) => (
                      <tr key={item.date} className="border-b dark:border-gray-700">
                        <td className="p-3">{item.date}</td>
                        <td className="p-3 text-right font-mono">{item.predicted_demand.toFixed(0)}</td>
                        <td className="p-3 text-right font-mono">{(item.confidence_level * 100).toFixed(1)}%</td>
                        <td className="p-3 text-right font-mono">{(item.weather_impact * 100).toFixed(1)}%</td>
                        <td className="p-3 text-right font-mono">{(item.seasonal_impact * 100).toFixed(1)}%</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* --- AIチャット --- */}
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-lg">
            <h2 className="text-2xl font-bold mb-4 border-b pb-2">AIチャット入力</h2>
            
            {/* メッセージ表示エリア */}
            <div className="h-80 overflow-y-auto mb-4 p-4 bg-gray-100 dark:bg-gray-700 rounded-md space-y-4">
              {chatMessages.map((msg, index) => (
                <div key={index} className={`flex flex-col ${msg.sender === 'user' ? 'items-end' : 'items-start'}`}>
                  <div className={`rounded-lg px-4 py-2 max-w-lg ${msg.sender === 'user' ? 'bg-blue-500 text-white' : 'bg-gray-300 dark:bg-gray-600 text-gray-900 dark:text-white'}`}>
                    <p>{msg.text}</p>
                    {msg.fileName && (
                      <div className="mt-2 text-xs opacity-80 border-t border-t-white/50 pt-1">
                        添付ファイル: {msg.fileName}
                      </div>
                    )}
                  </div>
                </div>
              ))}
              {chatLoading && (
                <div className="flex flex-col items-start">
                   <div className="rounded-lg px-4 py-2 max-w-lg bg-gray-300 dark:bg-gray-600 text-gray-900 dark:text-white animate-pulse">
                     AIが応答を生成中...
                   </div>
                </div>
              )}
            </div>

            {/* 入力フォーム */}
            <form onSubmit={handleChatSubmit}>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={chatInput}
                  onChange={(e) => setChatInput(e.target.value)}
                  placeholder="AIにメッセージを送信..."
                  className="flex-grow p-2 border rounded bg-gray-50 dark:bg-gray-700 border-gray-300 dark:border-gray-600"
                />
                <input
                  type="file"
                  ref={fileInputRef}
                  onChange={(e) => setSelectedFile(e.target.files ? e.target.files[0] : null)}
                  className="hidden"
                  id="file-upload"
                  accept=".xlsx, .csv"
                />
                <label htmlFor="file-upload" className="cursor-pointer p-2 border rounded hover:bg-gray-100 dark:hover:bg-gray-600">
                  📎
                </label>
                <button
                  type="submit"
                  disabled={chatLoading}
                  className="rounded-md bg-green-600 text-white px-4 py-2 hover:bg-green-700 disabled:bg-gray-500"
                >
                  送信
                </button>
              </div>
              {selectedFile && (
                <div className="text-sm mt-2 text-gray-500">
                  選択中のファイル: {selectedFile.name}
                </div>
              )}
            </form>
          </div>

        </div>
      </main>
      
      <footer className="text-center p-4 text-sm text-gray-500">
        <p>HUNT Chat-API Demo</p>
      </footer>
    </div>
  );
}
