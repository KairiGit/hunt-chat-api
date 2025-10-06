"use client";

import { useState } from 'react';
import Image from "next/image";

// 型定義を追加
interface WeatherData {
  time: string;
  weather_code: string;
  temperature: string;
}

interface ApiResponse {
  success: boolean;
  data: WeatherData[];
  count: number;
}

export default function Home() {
  const [weatherData, setWeatherData] = useState<WeatherData[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchWeather = async () => {
    setLoading(true);
    setError(null);
    setWeatherData(null);
    try {
      // GoバックエンドのAPIを呼び出す
      const response = await fetch('http://localhost:8080/api/v1/weather/tokyo');
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const result: ApiResponse = await response.json();
      if (result.success && result.data) {
        setWeatherData(result.data);
      } else {
        throw new Error('Failed to get weather data from API');
      }
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="font-sans grid grid-rows-[20px_1fr_20px] items-center justify-items-center min-h-screen p-8 pb-20 gap-16 sm:p-20">
      <main className="flex flex-col gap-[32px] row-start-2 items-center sm:items-start">
        <Image
          className="dark:invert"
          src="/next.svg"
          alt="Next.js logo"
          width={180}
          height={38}
          priority
        />
        
        <div className="text-center">
          <h1 className="text-2xl font-bold mb-4">Frontend-Backend Integration Demo</h1>
          <p>Click the button to fetch weather data from the Go backend.</p>
        </div>

        <div className="flex flex-col gap-4 items-center w-full">
          <button
            onClick={fetchWeather}
            disabled={loading}
            className="rounded-full border border-solid border-transparent transition-colors flex items-center justify-center bg-blue-500 text-white gap-2 hover:bg-blue-600 font-medium text-sm sm:text-base h-10 sm:h-12 px-4 sm:px-5 w-full sm:w-auto disabled:bg-gray-400"
          >
            {loading ? 'Loading...' : 'Fetch Tokyo Weather'}
          </button>

          {error && (
            <div className="mt-4 p-4 bg-red-100 text-red-700 border border-red-400 rounded w-full">
              <p className="font-bold">Error:</p>
              <p>{error}</p>
            </div>
          )}

          {weatherData && (
            <div className="mt-4 p-4 bg-gray-100 dark:bg-gray-800 rounded w-full max-w-2xl">
              <h2 className="text-xl font-bold mb-2">Tokyo Weather Data</h2>
              <pre className="text-sm whitespace-pre-wrap break-all">
                {JSON.stringify(weatherData, null, 2)}
              </pre>
            </div>
          )}
        </div>
      </main>
      <footer className="row-start-3 flex gap-[24px] flex-wrap items-center justify-center">
        <a
          className="flex items-center gap-2 hover:underline hover:underline-offset-4"
          href="https://nextjs.org?utm_source=create-next-app&utm_medium=appdir-template-tw&utm_campaign=create-next-app"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Image
            aria-hidden
            src="/globe.svg"
            alt="Globe icon"
            width={16}
            height={16}
          />
          Go to nextjs.org →
        </a>
      </footer>
    </div>
  );
}