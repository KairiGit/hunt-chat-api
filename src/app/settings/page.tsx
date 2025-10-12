'use client';

import { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

interface Settings {
  company_name: string;
  analysis_period: 'daily' | 'weekly' | 'monthly';
  forecast_horizon_weeks: number;
  confidence_level: number;
  notification_email: string;
  business_days: number[];
  fiscal_year_start: string;
}

interface Product {
  id: string;
  name: string;
  category: string;
  unit: string;
  lead_time_days: number;
}

export default function SettingsPage() {
  const [settings, setSettings] = useState<Settings>({
    company_name: '株式会社サンプル製造',
    analysis_period: 'weekly',
    forecast_horizon_weeks: 4,
    confidence_level: 0.95,
    notification_email: '',
    business_days: [1, 2, 3, 4, 5], // 月〜金
    fiscal_year_start: '04-01',
  });

  const [products, setProducts] = useState<Product[]>([
    { id: 'P001', name: '製品A', category: '電子部品', unit: '個', lead_time_days: 7 },
    { id: 'P002', name: '製品B', category: '機械部品', unit: 'セット', lead_time_days: 14 },
    { id: 'P003', name: '製品C', category: '組立品', unit: '台', lead_time_days: 21 },
    { id: 'P004', name: '製品D', category: '消耗品', unit: 'パック', lead_time_days: 3 },
    { id: 'P005', name: '製品E', category: '特注品', unit: 'ロット', lead_time_days: 30 },
  ]);

  const [isSaving, setIsSaving] = useState(false);
  const [saveMessage, setSaveMessage] = useState('');

  const handleSaveSettings = async () => {
    setIsSaving(true);
    setSaveMessage('');
    
    try {
      // TODO: APIエンドポイントに保存
      await new Promise(resolve => setTimeout(resolve, 1000)); // シミュレーション
      
      setSaveMessage('✅ 設定を保存しました');
      setTimeout(() => setSaveMessage(''), 3000);
    } catch (error) {
      setSaveMessage('❌ 保存に失敗しました');
    } finally {
      setIsSaving(false);
    }
  };

  const weekdays = [
    { value: 0, label: '日' },
    { value: 1, label: '月' },
    { value: 2, label: '火' },
    { value: 3, label: '水' },
    { value: 4, label: '木' },
    { value: 5, label: '金' },
    { value: 6, label: '土' },
  ];

  const toggleBusinessDay = (day: number) => {
    if (settings.business_days.includes(day)) {
      setSettings({
        ...settings,
        business_days: settings.business_days.filter(d => d !== day)
      });
    } else {
      setSettings({
        ...settings,
        business_days: [...settings.business_days, day].sort()
      });
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-50 p-8">
      <div className="max-w-6xl mx-auto space-y-6">
        <div>
          <h1 className="text-4xl font-bold text-gray-800 mb-2">⚙️ システム設定</h1>
          <p className="text-gray-600">需要予測システムの設定を管理します</p>
        </div>

        {/* 会社情報 */}
        <Card>
          <CardHeader>
            <CardTitle>🏢 会社情報</CardTitle>
            <CardDescription>基本情報の設定</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <Label htmlFor="company-name">会社名</Label>
              <Input
                id="company-name"
                value={settings.company_name}
                onChange={(e) => setSettings({ ...settings, company_name: e.target.value })}
                placeholder="株式会社〇〇"
              />
            </div>
            
            <div>
              <Label htmlFor="notification-email">通知先メールアドレス</Label>
              <Input
                id="notification-email"
                type="email"
                value={settings.notification_email}
                onChange={(e) => setSettings({ ...settings, notification_email: e.target.value })}
                placeholder="alerts@example.com"
              />
            </div>

            <div>
              <Label htmlFor="fiscal-year">会計年度開始月日</Label>
              <Input
                id="fiscal-year"
                value={settings.fiscal_year_start}
                onChange={(e) => setSettings({ ...settings, fiscal_year_start: e.target.value })}
                placeholder="04-01"
              />
              <p className="text-xs text-gray-500 mt-1">形式: MM-DD（例：04-01 は 4月1日）</p>
            </div>
          </CardContent>
        </Card>

        {/* 分析設定 */}
        <Card>
          <CardHeader>
            <CardTitle>📊 分析設定</CardTitle>
            <CardDescription>分析期間とパラメータの設定</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <Label>分析期間の単位</Label>
              <div className="grid grid-cols-3 gap-2 mt-2">
                {[
                  { value: 'daily', label: '日次', icon: '📅' },
                  { value: 'weekly', label: '週次', icon: '📆' },
                  { value: 'monthly', label: '月次', icon: '📊' },
                ].map((period) => (
                  <button
                    key={period.value}
                    onClick={() => setSettings({ ...settings, analysis_period: period.value as 'daily' | 'weekly' | 'monthly' })}
                    className={`p-4 rounded-lg border-2 transition-all ${
                      settings.analysis_period === period.value
                        ? 'border-blue-500 bg-blue-50 shadow-md'
                        : 'border-gray-200 hover:border-gray-300'
                    }`}
                  >
                    <div className="text-2xl mb-1">{period.icon}</div>
                    <div className="font-semibold">{period.label}</div>
                  </button>
                ))}
              </div>
              <p className="text-xs text-gray-500 mt-2">
                💡 <strong>推奨：</strong> toB製造業では週次分析が最適です
              </p>
            </div>

            <div>
              <Label htmlFor="forecast-weeks">予測期間（週数）</Label>
              <Input
                id="forecast-weeks"
                type="number"
                min="1"
                max="52"
                value={settings.forecast_horizon_weeks}
                onChange={(e) => setSettings({ ...settings, forecast_horizon_weeks: parseInt(e.target.value) })}
              />
              <p className="text-xs text-gray-500 mt-1">
                現在: {settings.forecast_horizon_weeks}週間（約{Math.round(settings.forecast_horizon_weeks / 4.33)}ヶ月）先まで予測
              </p>
            </div>

            <div>
              <Label htmlFor="confidence">信頼区間レベル</Label>
              <Input
                id="confidence"
                type="number"
                min="0.8"
                max="0.99"
                step="0.01"
                value={settings.confidence_level}
                onChange={(e) => setSettings({ ...settings, confidence_level: parseFloat(e.target.value) })}
              />
              <p className="text-xs text-gray-500 mt-1">
                現在: {(settings.confidence_level * 100).toFixed(0)}% 信頼区間
              </p>
            </div>

            <div>
              <Label>営業日設定</Label>
              <div className="flex gap-2 mt-2">
                {weekdays.map((day) => (
                  <button
                    key={day.value}
                    onClick={() => toggleBusinessDay(day.value)}
                    className={`px-4 py-2 rounded-lg border-2 transition-all ${
                      settings.business_days.includes(day.value)
                        ? 'border-blue-500 bg-blue-500 text-white shadow-md'
                        : 'border-gray-200 hover:border-gray-300'
                    }`}
                  >
                    {day.label}
                  </button>
                ))}
              </div>
              <p className="text-xs text-gray-500 mt-2">
                選択中: {settings.business_days.length}日/週
              </p>
            </div>
          </CardContent>
        </Card>

        {/* 製品マスタ */}
        <Card>
          <CardHeader>
            <CardTitle>📦 製品マスタ</CardTitle>
            <CardDescription>登録製品の管理</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b">
                    <th className="text-left p-2">製品ID</th>
                    <th className="text-left p-2">製品名</th>
                    <th className="text-left p-2">カテゴリ</th>
                    <th className="text-left p-2">単位</th>
                    <th className="text-left p-2">リードタイム</th>
                    <th className="text-left p-2">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {products.map((product) => (
                    <tr key={product.id} className="border-b hover:bg-gray-50">
                      <td className="p-2 font-mono text-sm">{product.id}</td>
                      <td className="p-2">{product.name}</td>
                      <td className="p-2">
                        <span className="px-2 py-1 bg-blue-100 text-blue-800 rounded text-xs">
                          {product.category}
                        </span>
                      </td>
                      <td className="p-2">{product.unit}</td>
                      <td className="p-2">{product.lead_time_days}日</td>
                      <td className="p-2">
                        <Button variant="outline" size="sm">編集</Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div className="mt-4">
              <Button variant="outline" className="w-full">
                + 新しい製品を追加
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* 保存ボタン */}
        <div className="flex justify-end gap-4 items-center">
          {saveMessage && (
            <span className={`text-sm ${saveMessage.includes('✅') ? 'text-green-600' : 'text-red-600'}`}>
              {saveMessage}
            </span>
          )}
          <Button
            onClick={handleSaveSettings}
            disabled={isSaving}
            className="px-8"
          >
            {isSaving ? '保存中...' : '💾 設定を保存'}
          </Button>
        </div>
      </div>
    </div>
  );
}
