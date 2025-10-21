'use client';

import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

interface WeeklySummary {
  week_number: number;
  week_start: string;
  week_end: string;
  total_sales: number;
  average_sales: number;
  min_sales: number;
  max_sales: number;
  business_days: number;
  week_over_week: number;
  std_dev: number;
  avg_temperature: number;
}

interface WeeklyOverallStats {
  average_weekly_sales: number;
  median_weekly_sales: number;
  std_dev_weekly_sales: number;
  best_week: number;
  worst_week: number;
  growth_rate: number;
  volatility: number;
}

interface WeeklyTrends {
  direction: string;
  strength: number;
  seasonality: string;
  peak_week: number;
  low_week: number;
  average_growth: number;
}

interface WeeklyAnalysis {
  product_id: string;
  product_name: string;
  analysis_period: string;
  total_weeks: number;
  weekly_summary: WeeklySummary[];
  overall_stats: WeeklyOverallStats;
  trends: WeeklyTrends;
  recommendations: string[];
}

export default function WeeklyAnalysisPage() {
  const [analysis, setAnalysis] = useState<WeeklyAnalysis | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  
  const [productId, setProductId] = useState('P001');
  const [startDate, setStartDate] = useState('2024-01-01');
  const [endDate, setEndDate] = useState('2024-03-31');
  const [granularity, setGranularity] = useState<'daily' | 'weekly' | 'monthly'>('weekly');

  const products = [
    { id: 'P001', name: '製品A' },
    { id: 'P002', name: '製品B' },
    { id: 'P003', name: '製品C' },
    { id: 'P004', name: '製品D' },
    { id: 'P005', name: '製品E' },
  ];

  const analyzeWeeklySales = async () => {
    setIsLoading(true);
    try {
      const response = await fetch('/api/v1/ai/analyze-weekly', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          product_id: productId,
          start_date: startDate,
          end_date: endDate,
          granularity: granularity,
        }),
      });

      if (!response.ok) {
        throw new Error('週次分析に失敗しました');
      }

      const result = await response.json();
      setAnalysis(result.data);
    } catch (error) {
      console.error('Error:', error);
      alert('分析中にエラーが発生しました');
    } finally {
      setIsLoading(false);
    }
  };

  const formatNumber = (num: number, decimals: number = 0) => {
    return num.toFixed(decimals).replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  };

  const getTrendIcon = (direction: string) => {
    switch (direction) {
      case '上昇': return '📈';
      case '下降': return '📉';
      case '横ばい': return '📊';
      default: return '📊';
    }
  };

  const getWeekOverWeekColor = (value: number) => {
    if (value > 5) return 'text-green-600';
    if (value < -5) return 'text-red-600';
    return 'text-gray-600';
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-50 p-8">
      <div className="max-w-7xl mx-auto space-y-6">
        {/* ヘッダー */}
        <div>
          <h1 className="text-4xl font-bold text-gray-800 mb-2">
            📊 {granularity === 'daily' ? '日次' : granularity === 'monthly' ? '月次' : '週次'}売上分析
          </h1>
          <p className="text-gray-600">
            製造業に最適化された{granularity === 'daily' ? '日別' : granularity === 'monthly' ? '月別' : '週別'}での販売実績分析
          </p>
        </div>

        {/* 分析条件入力 */}
        <Card>
          <CardHeader>
            <CardTitle>分析条件</CardTitle>
            <CardDescription>製品と期間を選択して週次分析を実行します</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-5 gap-4">
              <div>
                <Label htmlFor="product">製品</Label>
                <select
                  id="product"
                  value={productId}
                  onChange={(e) => setProductId(e.target.value)}
                  className="w-full p-2 border rounded-lg"
                >
                  {products.map((product) => (
                    <option key={product.id} value={product.id}>
                      {product.id} - {product.name}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <Label htmlFor="granularity">集約粒度</Label>
                <select
                  id="granularity"
                  value={granularity}
                  onChange={(e) => setGranularity(e.target.value as 'daily' | 'weekly' | 'monthly')}
                  className="w-full p-2 border rounded-lg"
                >
                  <option value="daily">📅 日次</option>
                  <option value="weekly">📆 週次</option>
                  <option value="monthly">📊 月次</option>
                </select>
              </div>

              <div>
                <Label htmlFor="start-date">開始日</Label>
                <Input
                  id="start-date"
                  type="date"
                  value={startDate}
                  onChange={(e) => setStartDate(e.target.value)}
                />
              </div>

              <div>
                <Label htmlFor="end-date">終了日</Label>
                <Input
                  id="end-date"
                  type="date"
                  value={endDate}
                  onChange={(e) => setEndDate(e.target.value)}
                />
              </div>

              <div className="flex items-end">
                <Button
                  onClick={analyzeWeeklySales}
                  disabled={isLoading}
                  className="w-full"
                >
                  {isLoading ? '分析中...' : '📊 分析実行'}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>

        {analysis && (
          <>
            {/* サマリーカード */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
              <Card className="bg-gradient-to-br from-blue-500 to-blue-600 text-white">
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium opacity-90">分析期間</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">
                    {analysis.total_weeks}{granularity === 'daily' ? '日間' : granularity === 'monthly' ? 'ヶ月' : '週間'}
                  </div>
                  <p className="text-xs opacity-75 mt-1">{analysis.analysis_period}</p>
                </CardContent>
              </Card>

              <Card className="bg-gradient-to-br from-green-500 to-green-600 text-white">
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium opacity-90">
                    {granularity === 'daily' ? '日' : granularity === 'monthly' ? '月' : '週'}平均売上
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">
                    {formatNumber(analysis.overall_stats.average_weekly_sales)}
                  </div>
                  <p className="text-xs opacity-75 mt-1">
                    個/{granularity === 'daily' ? '日' : granularity === 'monthly' ? '月' : '週'}
                  </p>
                </CardContent>
              </Card>

              <Card className="bg-gradient-to-br from-purple-500 to-purple-600 text-white">
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium opacity-90">成長率</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">
                    {analysis.overall_stats.growth_rate > 0 ? '+' : ''}
                    {formatNumber(analysis.overall_stats.growth_rate, 1)}%
                  </div>
                  <p className="text-xs opacity-75 mt-1">期間全体</p>
                </CardContent>
              </Card>

              <Card className="bg-gradient-to-br from-orange-500 to-orange-600 text-white">
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium opacity-90">変動係数</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">
                    {formatNumber(analysis.overall_stats.volatility, 2)}
                  </div>
                  <p className="text-xs opacity-75 mt-1">
                    {analysis.overall_stats.volatility > 0.3 ? '変動大' : '安定'}
                  </p>
                </CardContent>
              </Card>
            </div>

            {/* トレンド分析 */}
            <Card>
              <CardHeader>
                <CardTitle>📈 トレンド分析</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                  <div className="text-center p-4 bg-gray-50 rounded-lg">
                    <div className="text-4xl mb-2">{getTrendIcon(analysis.trends.direction)}</div>
                    <div className="text-xl font-bold mb-1">{analysis.trends.direction}トレンド</div>
                    <div className="text-sm text-gray-600">
                      平均成長率: {analysis.trends.average_growth > 0 ? '+' : ''}
                      {formatNumber(analysis.trends.average_growth, 1)}%/週
                    </div>
                  </div>

                  <div className="text-center p-4 bg-gray-50 rounded-lg">
                    <div className="text-4xl mb-2">🏆</div>
                    <div className="text-xl font-bold mb-1">ピーク: 第{analysis.trends.peak_week}週</div>
                    <div className="text-sm text-gray-600">
                      最低: 第{analysis.trends.low_week}週
                    </div>
                  </div>

                  <div className="text-center p-4 bg-gray-50 rounded-lg">
                    <div className="text-4xl mb-2">🌤️</div>
                    <div className="text-lg font-bold mb-1">季節性</div>
                    <div className="text-sm text-gray-600">{analysis.trends.seasonality}</div>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* 週次サマリーテーブル */}
            <Card>
              <CardHeader>
                <CardTitle>
                  📅 {granularity === 'daily' ? '日次' : granularity === 'monthly' ? '月次' : '週次'}内訳
                </CardTitle>
                <CardDescription>
                  各{granularity === 'daily' ? '日' : granularity === 'monthly' ? '月' : '週'}の詳細な売上データ
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead>
                      <tr className="border-b bg-gray-50">
                        <th className="text-left p-3">
                          {granularity === 'daily' ? '日' : granularity === 'monthly' ? '月' : '週'}
                        </th>
                        <th className="text-left p-3">期間</th>
                        <th className="text-right p-3">合計売上</th>
                        <th className="text-right p-3">日平均</th>
                        <th className="text-right p-3">
                          前{granularity === 'daily' ? '日' : granularity === 'monthly' ? '月' : '週'}比
                        </th>
                        <th className="text-right p-3">営業日</th>
                        <th className="text-right p-3">平均気温</th>
                      </tr>
                    </thead>
                    <tbody>
                      {analysis.weekly_summary.map((week) => (
                        <tr
                          key={week.week_number}
                          className="border-b hover:bg-gray-50"
                        >
                          <td className="p-3 font-semibold">
                            第{week.week_number}{granularity === 'daily' ? '日' : granularity === 'monthly' ? '月' : '週'}
                          </td>
                          <td className="p-3 text-sm text-gray-600">
                            {week.week_start} 〜 {week.week_end}
                          </td>
                          <td className="p-3 text-right font-semibold">
                            {formatNumber(week.total_sales)}
                          </td>
                          <td className="p-3 text-right">
                            {formatNumber(week.average_sales, 1)}
                          </td>
                          <td className={`p-3 text-right font-semibold ${getWeekOverWeekColor(week.week_over_week)}`}>
                            {week.week_over_week > 0 ? '+' : ''}
                            {formatNumber(week.week_over_week, 1)}%
                          </td>
                          <td className="p-3 text-right">{week.business_days}日</td>
                          <td className="p-3 text-right">{formatNumber(week.avg_temperature, 1)}°C</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </CardContent>
            </Card>

            {/* 推奨事項 */}
            <Card>
              <CardHeader>
                <CardTitle>💡 推奨事項</CardTitle>
                <CardDescription>分析結果に基づく具体的なアクション</CardDescription>
              </CardHeader>
              <CardContent>
                <ul className="space-y-2">
                  {analysis.recommendations.map((recommendation, index) => (
                    <li
                      key={index}
                      className="flex items-start gap-2 p-3 bg-blue-50 rounded-lg"
                    >
                      <span className="text-blue-600 mt-0.5">▶</span>
                      <span className="flex-1">{recommendation}</span>
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>

            {/* 統計サマリー */}
            <Card>
              <CardHeader>
                <CardTitle>📊 統計サマリー</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div className="p-4 bg-gray-50 rounded-lg">
                    <div className="text-sm text-gray-600 mb-1">週平均売上</div>
                    <div className="text-xl font-bold">
                      {formatNumber(analysis.overall_stats.average_weekly_sales)}
                    </div>
                  </div>
                  <div className="p-4 bg-gray-50 rounded-lg">
                    <div className="text-sm text-gray-600 mb-1">中央値</div>
                    <div className="text-xl font-bold">
                      {formatNumber(analysis.overall_stats.median_weekly_sales)}
                    </div>
                  </div>
                  <div className="p-4 bg-gray-50 rounded-lg">
                    <div className="text-sm text-gray-600 mb-1">標準偏差</div>
                    <div className="text-xl font-bold">
                      {formatNumber(analysis.overall_stats.std_dev_weekly_sales, 1)}
                    </div>
                  </div>
                  <div className="p-4 bg-gray-50 rounded-lg">
                    <div className="text-sm text-gray-600 mb-1">最高週 / 最低週</div>
                    <div className="text-xl font-bold">
                      {analysis.overall_stats.best_week} / {analysis.overall_stats.worst_week}
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </>
        )}
      </div>
    </div>
  );
}
