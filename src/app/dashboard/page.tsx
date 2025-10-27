'use client';

import { BarChart, LineChart, PieChart, Bar, Line, Pie, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, Cell } from 'recharts';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useToast } from "@/components/ui/use-toast"
import { useEffect, useState, useCallback } from 'react';

// --- Data Types --- //
type RequestsOverTime = { time: string; requests: number };
type Endpoints = { [key: string]: number };
type StatusCodes = { name: string; value: number };
type AvgResponseTimes = { endpoint: string; responseTime: number };
type RecentError = { id: string; timestamp: string; endpoint: string; error: string; statusCode: number };

type DashboardData = {
  requestsOverTime: RequestsOverTime[];
  endpoints: Endpoints;
  statusCodes: StatusCodes[];
  avgResponseTimes: AvgResponseTimes[];
  recentErrors: RecentError[];
};

// --- Components --- //

const COLORS = { '2xx Success': '#82ca9d', '4xx Client Error': '#ffc658', '5xx Server Error': '#ff8042' };

const DashboardPage = () => {
  const [data, setData] = useState<DashboardData | null>(null);
  const [period, setPeriod] = useState('24h');
  const [loading, setLoading] = useState(true);
  const [isMaintenanceMode, setIsMaintenanceMode] = useState(false);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const { toast } = useToast();

  const fetchDashboardData = useCallback(async () => {
    setLoading(true);
    try {
      const response = await fetch(`/api/proxy/monitoring/logs?period=${period}`);
      if (!response.ok) throw new Error('Failed to fetch dashboard data');
      const result: DashboardData = await response.json();
      setData(result);
    } catch (error) {
      console.error(error);
      toast({ title: "エラー", description: "ダッシュボードデータの取得に失敗しました。", variant: "destructive" });
    } finally {
      setLoading(false);
    }
  }, [period, toast]);

  const fetchHealthStatus = useCallback(async () => {
    try {
      const response = await fetch('/api/proxy/admin/health-status');
      if (!response.ok) throw new Error('Failed to fetch health status');
      const result = await response.json();
      setIsMaintenanceMode(result.isMaintenanceMode);
    } catch (error) {
      console.error(error);
      toast({ title: "エラー", description: "サーバー状態の取得に失敗しました。", variant: "destructive" });
    }
  }, [toast]);

  useEffect(() => {
    fetchDashboardData();
    fetchHealthStatus();
    const dashboardInterval = setInterval(fetchDashboardData, 30000);
    const healthInterval = setInterval(fetchHealthStatus, 10000);

    return () => {
      clearInterval(dashboardInterval);
      clearInterval(healthInterval);
    };
  }, [fetchDashboardData, fetchHealthStatus]);

  const handleMaintenanceChange = async () => {
    const action = isMaintenanceMode ? 'stop' : 'start';
    try {
      const response = await fetch(`/api/proxy/admin/maintenance/${action}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password }),
        }
      );

      const result = await response.json();
      if (!response.ok) {
        throw new Error(result.error || '操作に失敗しました。');
      }

      toast({ title: "成功", description: `メンテナンスモードが${action === 'start' ? '開始' : '停止'}されました。` });
      setIsMaintenanceMode(action === 'start');
      setIsDialogOpen(false);
      setUsername('');
      setPassword('');
      fetchDashboardData(); // 即時更新
    } catch (error) {
      const message = error instanceof Error ? error.message : '不明なエラー';
      toast({ title: "エラー", description: message, variant: "destructive" });
    }
  };

  if (loading && !data) {
    return <div className="p-8">Loading...</div>;
  }

  if (!data) {
    return <div className="p-8">Failed to load data.</div>;
  }

  const endpointData = Object.entries(data.endpoints).map(([name, value]) => ({ name, value }));

  return (
    <div className="p-4 md:p-8 space-y-6 bg-gray-50 dark:bg-gray-900">
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>管理者認証</DialogTitle>
            <DialogDescription>
              サーバーの状態を変更するには、管理者情報を入力してください。
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <Input
              id="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="管理者名"
            />
            <Input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="パスワード"
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsDialogOpen(false)}>キャンセル</Button>
            <Button onClick={handleMaintenanceChange}>適用</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <header className="flex justify-between items-start">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-gray-100">API利用状況ダッシュボード</h1>
          <div className="flex items-center gap-4 mt-2">
            <p className="text-sm text-gray-500 dark:text-gray-400">本番環境APIの利用状況</p>
            <div className="flex items-center gap-2">
              <span className={`relative flex h-3 w-3`}>
                <span className={`animate-ping absolute inline-flex h-full w-full rounded-full ${isMaintenanceMode ? 'bg-red-400' : 'bg-green-400'} opacity-75`}></span>
                <span className={`relative inline-flex rounded-full h-3 w-3 ${isMaintenanceMode ? 'bg-red-500' : 'bg-green-500'}`}></span>
              </span>
              <span className="text-sm font-medium">{isMaintenanceMode ? 'メンテナンス中' : '受付中'}</span>
            </div>
          </div>
        </div>
        <div className="flex flex-col items-end gap-2">
          <Select value={period} onValueChange={setPeriod}>
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="期間を選択" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="1h">直近1時間</SelectItem>
              <SelectItem value="24h">直近24時間</SelectItem>
              <SelectItem value="7d">過去7日間</SelectItem>
            </SelectContent>
          </Select>
          <Button variant={isMaintenanceMode ? "default" : "destructive"} onClick={() => setIsDialogOpen(true)}>
            {isMaintenanceMode ? '受付再開' : '受付停止'}
          </Button>
        </div>
      </header>

      {/* (以下、グラフコンポーネントは変更なし) */}
      <div className={`grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 ${isMaintenanceMode ? 'opacity-50' : ''}`}>
        {/* Total Requests */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>リクエスト数推移</CardTitle>
            <CardDescription>時間あたりのAPIリクエスト総数</CardDescription>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={data.requestsOverTime}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis />
                <Tooltip />
                <Legend />
                <Line type="monotone" dataKey="requests" stroke="#8884d8" activeDot={{ r: 8 }} />
              </LineChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* Status Codes */}
        <Card>
          <CardHeader>
            <CardTitle>HTTPステータスコード</CardTitle>
            <CardDescription>リクエスト結果の割合</CardDescription>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie data={data.statusCodes} dataKey="value" nameKey="name" cx="50%" cy="50%" outerRadius={100} fill="#8884d8" label>
                  {data.statusCodes.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[entry.name as keyof typeof COLORS]} />
                  ))}
                </Pie>
                <Tooltip />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* Endpoints */}
        <Card className="lg:col-span-3">
          <CardHeader>
            <CardTitle>エンドポイント別リクエスト数</CardTitle>
            <CardDescription>最も利用されているAPIエンドポイント</CardDescription>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={400}>
              <BarChart data={endpointData} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis type="number" />
                <YAxis type="category" dataKey="name" width={250} tick={{ fontSize: 12 }} />
                <Tooltip />
                <Legend />
                <Bar dataKey="value" fill="#82ca9d" name="リクエスト数" />
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* Average Response Time */}
        <Card className="lg:col-span-3">
          <CardHeader>
            <CardTitle>平均レスポンスタイム</CardTitle>
            <CardDescription>エンドポイントごとの平均応答速度 (ms)</CardDescription>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={400}>
              <BarChart data={data.avgResponseTimes}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="endpoint" hide />
                <YAxis />
                <Tooltip />
                <Legend />
                <Bar dataKey="responseTime" fill="#ffc658" name="平均レスポンスタイム(ms)" />
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* Recent Errors */}
        <Card className="lg:col-span-3">
          <CardHeader>
            <CardTitle>直近のエラーログ</CardTitle>
            <CardDescription>ステータスコード5xxの詳細</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="text-sm font-mono bg-gray-100 dark:bg-gray-800 rounded-md p-4 max-h-96 overflow-y-auto">
              {data.recentErrors.map(error => (
                <div key={error.id} className="p-2 border-b border-gray-200 dark:border-gray-700 last:border-b-0">
                  <p><span className="font-bold text-red-500">[{error.timestamp}]</span> {error.endpoint}</p>
                  <p className="pl-4">- <span className="font-bold">Status:</span> {error.statusCode}</p>
                  <p className="pl-4">- <span className="font-bold">Error:</span> {error.error}</p>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};

export default DashboardPage;