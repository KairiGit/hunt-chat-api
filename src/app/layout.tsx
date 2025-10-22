import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import Link from 'next/link';
import "./globals.css";

import { AppProvider } from '@/contexts/AppContext';
import { Toaster } from '@/components/ui/toaster';

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "HUNT Chat-API Dashboard",
  description: "Demand forecasting and AI chat application",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ja" className="h-full">
      <body className={`${geistSans.variable} ${geistMono.variable} h-full antialiased overflow-hidden`}>
        <AppProvider>
          <div className="flex h-full">
            {/* Sidebar */}
            <aside className="w-64 bg-white dark:bg-gray-800 shadow-md flex flex-col">
              <div className="p-6 border-b border-gray-200 dark:border-gray-700">
                <h1 className="text-2xl font-bold text-blue-600 dark:text-blue-400">🏭 HUNT</h1>
                <p className="text-xs text-gray-500 mt-1">需要予測システム</p>
              </div>
              
              <nav className="flex-1 p-4 space-y-6">
                {/* ホーム */}
                <div>
                  <Link href="/" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors">
                    <span className="mr-2">🏠</span>
                    ホーム
                  </Link>
                </div>

                {/* 分析セクション */}
                <div>
                  <div className="px-4 text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
                    分析
                  </div>
                  <Link href="/dashboard" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors">
                    <span className="mr-2">📊</span>
                    ダッシュボード
                  </Link>
                  <Link href="/product-analysis" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors">
                    <span className="mr-2">📦</span>
                    製品別分析
                  </Link>
                  <Link href="/analysis" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors">
                    <span className="mr-2">📁</span>
                    ファイル分析
                  </Link>
                </div>

                {/* AIセクション */}
                <div>
                  <div className="px-4 text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
                    AI機能
                  </div>
                  <Link href="/chat" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors">
                    <span className="mr-2">💬</span>
                    分析チャット
                  </Link>
                  <Link href="/anomaly-response" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors">
                    <span className="mr-2">⚠️</span>
                    異常対応
                  </Link>
                  <Link href="/learning" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors">
                    <span className="mr-2">🧠</span>
                    AI学習
                  </Link>
                </div>

                {/* 設定セクション */}
                <div>
                  <div className="px-4 text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
                    管理
                  </div>
                  <Link href="/settings" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors">
                    <span className="mr-2">⚙️</span>
                    設定
                  </Link>
                </div>
              </nav>

              {/* フッター */}
              <div className="p-4 border-t border-gray-200 dark:border-gray-700">
                <div className="text-xs text-gray-500 text-center">
                  <p>toB製造業向け</p>
                  <p className="mt-1">v1.0.0</p>
                </div>
              </div>
            </aside>

            {/* Main Content */}
            <main className="flex-1 overflow-y-auto p-6">
              {children}
            </main>
          </div>
          <Toaster />
        </AppProvider>
      </body>
    </html>
  );
}
