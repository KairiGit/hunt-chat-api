"use client";

import { useState } from 'react';
import { Geist, Geist_Mono } from "next/font/google";
import Link from 'next/link';
import Image from 'next/image';
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

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  return (
    <html lang="ja" className="h-full">
      <body className={`${geistSans.variable} ${geistMono.variable} h-full antialiased overflow-hidden`}>
        <AppProvider>
          <div className="flex h-full">
            {/* Overlay */}
            {isSidebarOpen && (
              <div 
                className="fixed inset-0 bg-black bg-opacity-50 z-10 md:hidden"
                onClick={() => setIsSidebarOpen(false)}
              ></div>
            )}

            {/* Sidebar */}
            <aside className={`fixed top-0 left-0 h-full w-64 bg-white dark:bg-gray-800 shadow-md flex flex-col transform ${isSidebarOpen ? 'translate-x-0' : '-translate-x-full'} transition-transform duration-300 ease-in-out md:relative md:translate-x-0 z-20`}>
              <div className="p-4 flex justify-center items-center border-b border-gray-200 dark:border-gray-700">
                <Image src="/img/HUNT-logo.jpeg" alt="HUNT logo" width={180} height={50} priority />
              </div>
              
              <nav className="flex-1 p-4 space-y-6">
                {/* ホーム */}
                <div>
                  <Link href="/" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors" onClick={() => setIsSidebarOpen(false)}>
                    <span className="mr-2">🏠</span>
                    ホーム
                  </Link>
                </div>

                {/* 分析セクション */}
                <div>
                  <div className="px-4 text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
                    分析
                  </div>
                  <Link href="/dashboard" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors" onClick={() => setIsSidebarOpen(false)}>
                    <span className="mr-2">📊</span>
                    ダッシュボード
                  </Link>
                  <Link href="/product-analysis" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors" onClick={() => setIsSidebarOpen(false)}>
                    <span className="mr-2">📦</span>
                    製品別分析
                  </Link>
                  <Link href="/analysis" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors" onClick={() => setIsSidebarOpen(false)}>
                    <span className="mr-2">📁</span>
                    ファイル分析
                  </Link>
                </div>

                {/* AIセクション */}
                <div>
                  <div className="px-4 text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
                    AI機能
                  </div>
                  <Link href="/chat" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors" onClick={() => setIsSidebarOpen(false)}>
                    <span className="mr-2">💬</span>
                    分析チャット
                  </Link>
                  <Link href="/anomaly-response" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors" onClick={() => setIsSidebarOpen(false)}>
                    <span className="mr-2">⚠️</span>
                    異常対応
                  </Link>
                  <Link href="/learning" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors" onClick={() => setIsSidebarOpen(false)}>
                    <span className="mr-2">🧠</span>
                    AI学習
                  </Link>
                </div>

                {/* 設定セクション */}
                <div>
                  <div className="px-4 text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
                    管理
                  </div>
                  <Link href="/settings" className="flex items-center px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-50 dark:hover:bg-gray-700 transition-colors" onClick={() => setIsSidebarOpen(false)}>
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
            <main className="flex-1 overflow-y-auto p-2">
              <div className="md:hidden flex justify-between items-center">
                <button onClick={() => setIsSidebarOpen(true)} className="p-2">
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-6 h-6">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" />
                  </svg>
                </button>
              </div>
              {children}
            </main>
          </div>
          <Toaster />
        </AppProvider>
      </body>
    </html>
  );
}
