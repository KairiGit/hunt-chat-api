import { Card, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  FileSpreadsheet,
  MessageSquare,
  AlertCircle,
  Package,
  TrendingUp,
  Settings,
} from 'lucide-react';
import Link from 'next/link';

const navigationItems = [
  {
    href: '/analysis',
    icon: FileSpreadsheet,
    title: 'ファイル分析',
    description: 'CSV/Excelファイルをアップロードして即座に分析',
    color: 'purple',
    hoverBorderColor: 'hover:border-purple-300',
    bgColor: 'bg-purple-100 dark:bg-purple-900/30',
    iconColor: 'text-purple-600',
  },
  {
    href: '/chat',
    icon: MessageSquare,
    title: 'AI分析チャット',
    description: 'RAG搭載のAIと対話しながら分析',
    color: 'blue',
    hoverBorderColor: 'hover:border-blue-300',
    bgColor: 'bg-blue-100 dark:bg-blue-900/30',
    iconColor: 'text-blue-600',
  },
  {
    href: '/anomaly-response',
    icon: AlertCircle,
    title: '異常対応',
    description: '検出された異常の原因を対話的に特定',
    color: 'amber',
    hoverBorderColor: 'hover:border-amber-300',
    bgColor: 'bg-amber-100 dark:bg-amber-900/30',
    iconColor: 'text-amber-600',
  },
  {
    href: '/product-analysis',
    icon: Package,
    title: '製品別分析',
    description: '製品ごとの販売実績を詳細に分析',
    color: 'green',
    hoverBorderColor: 'hover:border-green-300',
    bgColor: 'bg-green-100 dark:bg-green-900/30',
    iconColor: 'text-green-600',
  },
  {
    href: '/dashboard',
    icon: TrendingUp,
    title: '需要予測',
    description: '製品別・期間別の需要予測',
    color: 'indigo',
    hoverBorderColor: 'hover:border-indigo-300',
    bgColor: 'bg-indigo-100 dark:bg-indigo-900/30',
    iconColor: 'text-indigo-600',
  },
  {
    href: '/settings',
    icon: Settings,
    title: '設定',
    description: 'システム設定・製品マスタ管理',
    color: 'gray',
    hoverBorderColor: 'hover:border-gray-300',
    bgColor: 'bg-gray-100 dark:bg-gray-700',
    iconColor: 'text-gray-600 dark:text-gray-400',
  },
];

export function QuickNavigation() {
  return (
    <section className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {navigationItems.map((item) => {
        const Icon = item.icon;
        return (
          <Link key={item.href} href={item.href}>
            <Card className={`hover:shadow-lg transition-all cursor-pointer border-2 ${item.hoverBorderColor}`}>
              <CardHeader>
                <div className="flex items-center gap-3">
                  <div className={`p-2 ${item.bgColor} rounded-lg`}>
                    <Icon className={`h-6 w-6 ${item.iconColor}`} />
                  </div>
                  <CardTitle className="text-lg">{item.title}</CardTitle>
                </div>
                <CardDescription>
                  {item.description}
                </CardDescription>
              </CardHeader>
            </Card>
          </Link>
        );
      })}
    </section>
  );
}
