'use client';

import { useState, useRef, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useAppContext } from '@/contexts/AppContext';
import { AnalysisReportView } from '@/components/analysis/AnalysisReportView';
import { AnalysisProgressBar } from '@/components/analysis/AnalysisProgressBar';
import type { AnalysisReport, AnalysisResponse, AnalysisReportHeader } from '@/types/analysis';
import { useToast } from '@/components/ui/use-toast';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";

export default function AnalysisPage() {
  const { analysisSummary, setAnalysisSummary } = useAppContext();
  const { toast } = useToast();

  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [warning, setWarning] = useState<string | null>(null);
  const [granularity, setGranularity] = useState<'daily' | 'weekly' | 'monthly'>('weekly');
  const [pendingGranularity, setPendingGranularity] = useState<'daily' | 'weekly' | 'monthly' | null>(null);
  const [isGranularityChangeDialogOpen, setGranularityChangeDialogOpen] = useState(false);
  
  const [reportList, setReportList] = useState<AnalysisReportHeader[]>([]);
  const [isLoadingList, setIsLoadingList] = useState(true);
  
  const [selectedReport, setSelectedReport] = useState<AnalysisReport | null>(null);
  const [isLoadingReport, setIsLoadingReport] = useState(false);

  const [isReportDeleteDialogOpen, setReportDeleteDialogOpen] = useState(false);
  const [reportIdToDelete, setReportIdToDelete] = useState<string | null>(null);
  const [isDeleteAllReportsDialogOpen, setDeleteAllReportsDialogOpen] = useState(false);

  const fileInputRef = useRef<HTMLInputElement>(null);

  // =========================
  // ④ 経済×売上 相関/因果: 状態
  // =========================
  type EconPoint = { date?: string; period?: string; value?: number };
  type SalesPoint = { date?: string; period?: string; sales?: number; value?: number };
  type WindowResult = { window_start: string; window_end: string; best_lag: number; r: number; p: number; p_adj?: number; n: number };
  type LagResult = { lag?: number; best_lag?: number; r?: number; correlation_coef?: number; p?: number; p_value?: number; n?: number };
  type GrangerStat = { F?: number; p?: number };
  type GrangerResult = { direction?: string; order?: number; granularity?: string; x_to_y?: GrangerStat; y_to_x?: GrangerStat };

  const [econSymbol, setEconSymbol] = useState('NIKKEI');
  const [productId, setProductId] = useState('P001');
  const [econStart, setEconStart] = useState<string>('');
  const [econEnd, setEconEnd] = useState<string>('');
  const [econSeries, setEconSeries] = useState<EconPoint[]>([]);
  const [salesSeries, setSalesSeries] = useState<SalesPoint[]>([]);
  const [lagResults, setLagResults] = useState<LagResult[] | null>(null);
  const [winResults, setWinResults] = useState<WindowResult[] | null>(null);
  const [granger, setGranger] = useState<GrangerResult | null>(null);
  const [econMsg, setEconMsg] = useState<string>('');

  // レポート選択時に期間を自動反映（YYYY-MM-DD を2つ拾う）
  useEffect(() => {
    const range = selectedReport?.date_range || '';
    const matches = range.match(/\d{4}-\d{2}-\d{2}/g) || [];
    if (matches.length >= 2) {
  // Guard against potential undefined for TypeScript
  setEconStart(matches[0] ?? '');
  setEconEnd(matches[1] ?? '');
    }
  }, [selectedReport]);

  const handleGranularityChange = (newGranularity: 'daily' | 'weekly' | 'monthly') => {
    // 既に分析済みの場合はアラートを表示
    if (selectedReport || analysisSummary) {
      setPendingGranularity(newGranularity);
      setGranularityChangeDialogOpen(true);
    } else {
      setGranularity(newGranularity);
    }
  };

  const confirmGranularityChange = () => {
    if (pendingGranularity) {
      setGranularity(pendingGranularity);
      setSelectedReport(null);
      setAnalysisSummary('');
      setPendingGranularity(null);
    }
    setGranularityChangeDialogOpen(false);
  };

  const cancelGranularityChange = () => {
    setPendingGranularity(null);
    setGranularityChangeDialogOpen(false);
  };

  const fetchReportList = async () => {
    setIsLoadingList(true);
    try {
      const response = await fetch('/api/proxy/analysis-reports');
      const data = await response.json();
      if (data.success) {
        setReportList(data.reports || []);
      } else {
        throw new Error(data.error || 'Failed to fetch report list.');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error fetching reports.');
    } finally {
      setIsLoadingList(false);
    }
  };

  useEffect(() => {
    fetchReportList();
  }, []);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      setSelectedFile(e.target.files[0]);
    }
  };

  const handleReportSelect = async (reportId: string) => {
    if (selectedReport?.report_id === reportId) {
      setSelectedReport(null);
      return;
    }

    setIsLoadingReport(true);
    setSelectedReport(null);
    setError(null);
    try {
      const response = await fetch(`/api/proxy/analysis-report?id=${reportId}`);
      const data = await response.json();
      if (data.success && data.report) {
        setSelectedReport(data.report);
      } else {
        throw new Error(data.error || 'Failed to fetch report details.');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error fetching report details.');
    } finally {
      setIsLoadingReport(false);
    }
  };

  const openDeleteDialog = (reportId: string) => {
    setReportIdToDelete(reportId);
    setReportDeleteDialogOpen(true);
  };

  const handleDeleteReport = async () => {
    if (!reportIdToDelete) return;

    try {
      const response = await fetch(`/api/proxy/analysis-report?id=${reportIdToDelete}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || 'Failed to delete report.');
      }

      toast({
        variant: "success",
        title: "✅ 削除完了",
        description: "レポートを削除しました。",
      });

      setReportList(reportList.filter(r => r.report_id !== reportIdToDelete));
      if (selectedReport?.report_id === reportIdToDelete) {
        setSelectedReport(null);
      }

    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error deleting report.');
    } finally {
      setReportIdToDelete(null);
      setReportDeleteDialogOpen(false);
    }
  };

  const handleDeleteAllReports = async () => {
    try {
      const response = await fetch('/api/proxy/analysis-reports', {
        method: 'DELETE',
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || 'Failed to delete all reports.');
      }

      toast({
        variant: "success",
        title: "✅ 全件削除完了",
        description: "すべての分析レポートを削除しました。",
      });

      setReportList([]);
      setSelectedReport(null);

    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error deleting all reports.');
    } finally {
      setDeleteAllReportsDialogOpen(false);
    }
  };

  const handleFileAnalysis = async () => {
    if (!selectedFile) return;

    setIsLoading(true);
    setError(null);
    setWarning(null);
    setAnalysisSummary('');
    setSelectedReport(null);
    const formData = new FormData();
    formData.append('file', selectedFile);
    formData.append('granularity', granularity); // 🆕 粒度を追加

    try {
      const response = await fetch('/api/proxy/analyze-file', { 
        method: 'POST', 
        body: formData 
      });

      if (!response.ok) {
        const errData = await response.json();
        let detailedError = errData.error || `File analysis failed: ${response.statusText}`;
        if (errData.details) {
          detailedError += `: ${errData.details}`;
        }
        throw new Error(detailedError);
      }

      const result: AnalysisResponse = await response.json();
      
      if (result.error) {
        throw new Error(result.error);
      }
      
      if (result.success) {
        setAnalysisSummary(result.summary || '');
        if (result.analysis_report) {
          setSelectedReport(result.analysis_report);
          fetchReportList();
          
          // 非同期処理のステータスを表示
          if (result.ai_insights_pending) {
            toast({
              title: "🤖 AI分析実行中",
              description: "AI洞察の生成をバックグラウンドで実行中です。完了まで2-5秒かかります。",
            });
          }
          if (result.ai_questions_pending) {
            toast({
              title: "💬 AI質問生成中",
              description: "異常検知のAI質問をバックグラウンドで生成中です。完了まで5-10秒かかります。",
            });
          }
        } else {
          const warningMessage = result.error 
            ? `詳細レポートの生成に失敗しました。${result.error}`
            : '基本的な分析は完了しましたが、詳細レポートの生成に失敗しました。データ量や気象データの取得に問題がある可能性があります。';
          setWarning(warningMessage);
        }
      } else {
        throw new Error(result.summary || 'Failed to get analysis summary.');
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'An unknown error occurred during analysis.');
    } finally {
      setIsLoading(false);
    }
  };

  // =========================
  // ④ 経済×売上 相関/因果: ハンドラ
  // =========================
  const fetchEcon = async () => {
    setEconMsg('');
    try {
      const s = econStart || '2024-01-01';
      const e = econEnd || new Date().toISOString().slice(0, 10);
      const url = `/api/proxy/econ/series?symbol=${encodeURIComponent(econSymbol)}&start=${s}&end=${e}&granularity=${granularity}`;
      const res = await fetch(url);
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error || res.statusText);
      setEconSeries(data.series || []);
      setEconMsg(`✅ 指標 ${econSymbol}：${data.count ?? (data.series?.length || 0)} 件`);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      setEconMsg(`❌ ${msg}`);
    }
  };

  const fetchSales = async () => {
    setEconMsg('');
    try {
      const s = econStart || '2024-01-01';
      const e = econEnd || new Date().toISOString().slice(0, 10);
      const url = `/api/proxy/econ/sales/series?product_id=${encodeURIComponent(productId)}&start=${s}&end=${e}&granularity=${granularity}`;
      const res = await fetch(url);
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error || res.statusText);
      setSalesSeries(data.series || []);
      setEconMsg(`✅ 売上 ${productId}：${data.count ?? (data.series?.length || 0)} 件`);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      setEconMsg(`❌ ${msg}`);
    }
  };

  const runLag = async () => {
    setEconMsg('');
    setLagResults(null);
    try {
      const s = econStart || '2024-01-01';
      const e = econEnd || new Date().toISOString().slice(0, 10);
      // sales: 日付キーは date or period を受け入れ、数値は sales or value
      const salesBody = (salesSeries || []).map(p => ({ date: p.date || p.period!, sales: p.sales ?? p.value ?? 0 }));
      const res = await fetch('/api/proxy/econ/lagged-correlation', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ symbol: econSymbol, start: s, end: e, sales: salesBody, max_lag: 21, granularity: granularity })
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error || res.statusText);
      setLagResults(data.results || []);
      setEconMsg(`✅ ラグ相関 OK（top lag=${data.top?.lag ?? data.top?.best_lag ?? '-'}）`);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      setEconMsg(`❌ ${msg}`);
    }
  };

  const runWindowed = async () => {
    setEconMsg('');
    setWinResults(null);
    try {
      const s = econStart || '2024-01-01';
      const e = econEnd || new Date().toISOString().slice(0, 10);
      const body = { product_id: productId, symbol: econSymbol, start: s, end: e, max_lag: 21, window_days: 90, step_days: 30, granularity: granularity };
      const res = await fetch('/api/proxy/econ/sales/lagged-correlation/windowed', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error || res.statusText);
      setWinResults(data.windows || []);
      setEconMsg(`✅ スライディング窓 OK（${(data.windows || []).length}件）`);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      setEconMsg(`❌ ${msg}`);
    }
  };

  const runGranger = async () => {
    setEconMsg('');
    setGranger(null);
    try {
      const s = econStart || '2024-01-01';
      const e = econEnd || new Date().toISOString().slice(0, 10);
      const body = { product_id: productId, symbol: econSymbol, start: s, end: e, order: 3, granularity: granularity };
      const res = await fetch('/api/proxy/econ/sales/granger', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error || res.statusText);
      setGranger(data);
      setEconMsg(`✅ グランジャー OK（${data.direction}）`);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      setEconMsg(`❌ ${msg}`);
    }
  };

  return (
    <div className="space-y-8">
      <h1 className="text-2xl font-bold">ファイル分析</h1>
      
      {/* ファイルアップロードと進捗バーを横並び */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 左側: ファイルアップロード */}
        <Card>
          <CardHeader>
            <CardTitle>① ファイルのアップロード</CardTitle>
            <CardDescription>分析したい販売実績データ（.xlsx, .csv）を選択してください。</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="grid w-full items-center gap-1.5">
                <Label htmlFor="granularity">データ集約粒度</Label>
                <select
                  id="granularity"
                  value={granularity}
                  onChange={(e) => handleGranularityChange(e.target.value as 'daily' | 'weekly' | 'monthly')}
                  className="w-full p-2 border rounded-lg"
                  disabled={isLoading}
                >
                  <option value="daily">📅 日次（詳細分析・短期トレンド）</option>
                  <option value="weekly">📆 週次（推奨・中期トレンド）</option>
                  <option value="monthly">📊 月次（長期トレンド・高速処理）</option>
                </select>
                <p className="text-xs text-gray-500 mt-1">
                  {granularity === 'daily' && '⚡ 処理時間: やや遅い | 📊 詳細度: 高 | 💡 用途: 短期分析（1週間〜1ヶ月）'}
                  {granularity === 'weekly' && '⚡ 処理時間: 普通 | 📊 詳細度: 中 | 💡 用途: 中期分析（1ヶ月〜6ヶ月）⭐'}
                  {granularity === 'monthly' && '⚡ 処理時間: 高速 | 📊 詳細度: 低 | 💡 用途: 長期分析（6ヶ月以上）'}
                </p>
              </div>
              
              <div className="grid w-full items-center gap-1.5">
                <Label htmlFor="file-upload">ファイル</Label>
                <Input id="file-upload" type="file" ref={fileInputRef} onChange={handleFileChange} accept=".xlsx, .csv" />
              </div>
            </div>
          </CardContent>
          <CardFooter className="flex justify-end">
            <Button onClick={handleFileAnalysis} disabled={!selectedFile || isLoading}>
              {isLoading ? '分析中...' : '分析開始'}
            </Button>
          </CardFooter>
        </Card>

        {/* 右側: 進捗バーまたはプレースホルダー */}
        <div className="min-h-[400px]">
          {isLoading ? (
            <AnalysisProgressBar isAnalyzing={isLoading} />
          ) : (
            <Card className="h-full border-dashed border-2 border-gray-300 dark:border-gray-700">
              <CardContent className="flex flex-col items-center justify-center h-full p-8 text-center">
                <div className="space-y-4">
                  <div className="text-6xl">📊</div>
                  <div className="space-y-2">
                    <p className="text-lg font-semibold text-gray-700 dark:text-gray-300">
                      分析の進捗をここに表示
                    </p>
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      ファイルを選択して「分析開始」をクリックすると、<br />
                      リアルタイムで処理の進行状況が表示されます
                    </p>
                  </div>
                  <div className="pt-4 space-y-2 text-xs text-left text-gray-600 dark:text-gray-400">
                    <div className="flex items-center gap-2">
                      <span>📁</span>
                      <span>ファイル読み込み</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span>🔍</span>
                      <span>CSV解析</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span>📊</span>
                      <span>統計分析</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span>🤖</span>
                      <span>AI分析</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span>⚠️</span>
                      <span>異常検知</span>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      {/* エラーと警告メッセージ */}
      {(error || warning) && (
        <div className="space-y-4">
          {error && (
            <Card className="bg-destructive/10 border-destructive">
              <CardHeader>
                <CardTitle className="text-destructive">分析エラー</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-destructive-foreground">{error}</p>
              </CardContent>
            </Card>
          )}
          
          {warning && (
            <Card className="bg-yellow-50 dark:bg-yellow-950 border-yellow-500">
              <CardHeader>
                <CardTitle className="text-yellow-700 dark:text-yellow-400">⚠️ 警告</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-yellow-700 dark:text-yellow-300">{warning}</p>
              </CardContent>
            </Card>
          )}
        </div>
      )}

      {/* レポート一覧 */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-bold">② 過去の分析レポート</h2>
          {reportList.length > 0 && (
            <Button variant="destructive" size="sm" onClick={() => setDeleteAllReportsDialogOpen(true)}>
              全レポートを削除
            </Button>
          )}
        </div>
        {isLoadingList ? (
          <p>レポート一覧を読み込み中...</p>
        ) : reportList.length === 0 ? (
          <p className="text-sm text-muted-foreground">保存されているレポートはありません。</p>
        ) : (
          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ファイル名</TableHead>
                    <TableHead>データ区間</TableHead>
                    <TableHead>分析日時</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {reportList.map((report) => (
                    <TableRow 
                      key={report.report_id} 
                      className={`cursor-pointer ${selectedReport?.report_id === report.report_id ? 'bg-muted/50' : 'hover:bg-muted/50'}`}>
                      <TableCell 
                        className="font-medium"
                        onClick={() => handleReportSelect(report.report_id)}>
                          {report.file_name}
                      </TableCell>
                      <TableCell onClick={() => handleReportSelect(report.report_id)}>
                        {report.date_range}
                      </TableCell>
                      <TableCell onClick={() => handleReportSelect(report.report_id)}>
                        {new Date(report.analysis_date).toLocaleString('ja-JP')}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button 
                          variant="ghost" 
                          size="sm"
                          onClick={(e) => {
                            e.stopPropagation(); // Prevent row click
                            openDeleteDialog(report.report_id);
                          }}>
                          🗑️
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        )}
      </div>

      {/* 詳細分析レポート表示 */}
      {isLoadingReport && <p>レポート詳細を読み込み中...</p>}
      
      {selectedReport && (
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <h2 className="text-xl font-bold">③ 分析レポート詳細</h2>
            <span className="text-sm text-muted-foreground">
              この内容はチャットページで引き継がれます
            </span>
          </div>
          <AnalysisReportView report={selectedReport} />
        </div>
      )}

      {/* 個別レポート削除ダイアログ */}
      <AlertDialog open={isReportDeleteDialogOpen} onOpenChange={setReportDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>本当にこのレポートを削除しますか？</AlertDialogTitle>
            <AlertDialogDescription>
              この操作は取り消せません。レポートは完全に削除されます。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>キャンセル</AlertDialogCancel>
            <AlertDialogAction onClick={handleDeleteReport} className="bg-red-500 hover:bg-red-600">
              削除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 全レポート削除ダイアログ */}
      <AlertDialog open={isDeleteAllReportsDialogOpen} onOpenChange={setDeleteAllReportsDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>本当によろしいですか？</AlertDialogTitle>
            <AlertDialogDescription>
              すべての分析レポートがデータベースから完全に削除されます。この操作は取り消せません。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>キャンセル</AlertDialogCancel>
            <AlertDialogAction onClick={handleDeleteAllReports} className="bg-red-500 hover:bg-red-600">
              すべて削除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 粒度変更確認ダイアログ */}
      <AlertDialog open={isGranularityChangeDialogOpen} onOpenChange={setGranularityChangeDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>⚠️ データ粒度を変更しますか？</AlertDialogTitle>
            <AlertDialogDescription>
              粒度を変更すると、現在の分析結果がクリアされます。
            </AlertDialogDescription>
          </AlertDialogHeader>
          {pendingGranularity && (
            <div className="mt-2 p-3 bg-blue-50 dark:bg-blue-950 rounded-md">
              <p className="font-semibold text-blue-900 dark:text-blue-100">
                {granularity === 'daily' && '日次'}
                {granularity === 'weekly' && '週次'}
                {granularity === 'monthly' && '月次'}
                {' → '}
                {pendingGranularity === 'daily' && '日次'}
                {pendingGranularity === 'weekly' && '週次'}
                {pendingGranularity === 'monthly' && '月次'}
              </p>
              <p className="text-sm text-blue-800 dark:text-blue-200 mt-1">
                {pendingGranularity === 'daily' && '📅 詳細な日次分析に切り替えます'}
                {pendingGranularity === 'weekly' && '📆 週次分析に切り替えます（推奨）'}
                {pendingGranularity === 'monthly' && '📊 月次の高速分析に切り替えます'}
              </p>
            </div>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel onClick={cancelGranularityChange}>キャンセル</AlertDialogCancel>
            <AlertDialogAction onClick={confirmGranularityChange} className="bg-blue-500 hover:bg-blue-600">
              変更する
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* ④ 経済×売上 相関/因果 分析 */}
      <div className="space-y-4">
        <h2 className="text-xl font-bold">④ 経済×売上 相関/因果 分析</h2>
        <Card>
          <CardHeader>
            <CardTitle>条件（粒度は上部の選択を流用）</CardTitle>
            <CardDescription>シンボル・製品・期間を指定して系列取得 → 相関/窓/因果を実行</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-6 gap-3">
              <div>
                <Label>シンボル</Label>
                <Input value={econSymbol} onChange={(e) => setEconSymbol(e.target.value.toUpperCase())} />
              </div>
              <div>
                <Label>製品ID</Label>
                <Input value={productId} onChange={(e) => setProductId(e.target.value)} />
              </div>
              <div>
                <Label>開始</Label>
                <Input type="date" value={econStart} onChange={(e) => setEconStart(e.target.value)} />
              </div>
              <div>
                <Label>終了</Label>
                <Input type="date" value={econEnd} onChange={(e) => setEconEnd(e.target.value)} />
              </div>
              <div className="md:col-span-2 flex items-end gap-2">
                <Button onClick={fetchEcon}>指標取得</Button>
                <Button onClick={fetchSales}>売上取得</Button>
                <Button onClick={runLag} disabled={salesSeries.length === 0}>ラグ相関</Button>
                <Button onClick={runWindowed}>窓分析</Button>
                <Button onClick={runGranger}>因果性</Button>
              </div>
            </div>
            {econMsg && (
              <div className={`mt-2 text-sm ${econMsg.startsWith('✅') ? 'text-green-700' : 'text-red-600'}`}>{econMsg}</div>
            )}
          </CardContent>
        </Card>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Card>
            <CardHeader>
              <CardTitle>経済系列</CardTitle>
              <CardDescription>{econSymbol}（{granularity}）</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="max-h-64 overflow-auto text-sm">
                {econSeries.length === 0 ? (
                  <div className="text-gray-500">なし</div>
                ) : (
                  <table className="w-full">
                    <thead>
                      <tr><th className="text-left p-1">日付/期間</th><th className="text-right p-1">値</th></tr>
                    </thead>
                    <tbody>
                      {econSeries.map((p, i) => (
                        <tr key={i} className="border-b">
                          <td className="p-1">{p.date || p.period}</td>
                          <td className="p-1 text-right">{p.value !== undefined ? p.value.toFixed(2) : ''}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>売上系列</CardTitle>
              <CardDescription>{productId}（{granularity}）</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="max-h-64 overflow-auto text-sm">
                {salesSeries.length === 0 ? (
                  <div className="text-gray-500">なし</div>
                ) : (
                  <table className="w-full">
                    <thead>
                      <tr><th className="text-left p-1">日付/期間</th><th className="text-right p-1">売上</th></tr>
                    </thead>
                    <tbody>
                      {salesSeries.map((p, i) => (
                        <tr key={i} className="border-b">
                          <td className="p-1">{p.date || p.period}</td>
                          <td className="p-1 text-right">{(p.sales ?? p.value) !== undefined ? (p.sales ?? p.value)!.toFixed(2) : ''}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Card>
            <CardHeader>
              <CardTitle>ラグ相関（全体）</CardTitle>
            </CardHeader>
            <CardContent>
              {!lagResults ? (
                <div className="text-sm text-gray-500">未実行</div>
              ) : (
                <table className="w-full text-sm">
                  <thead>
                    <tr><th className="text-left p-1">ラグ</th><th className="text-right p-1">r</th><th className="text-right p-1">p</th><th className="text-right p-1">n</th></tr>
                  </thead>
                  <tbody>
                    {lagResults.map((r: LagResult, i: number) => (
                      <tr key={i} className="border-b">
                        <td className="p-1">{r.lag ?? r.best_lag}</td>
                        <td className="p-1 text-right">{(r.r ?? r.correlation_coef)?.toFixed?.(3)}</td>
                        <td className="p-1 text-right">{Number(r.p ?? r.p_value).toExponential?.(2)}</td>
                        <td className="p-1 text-right">{r.n}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>スライディング窓 最適ラグ</CardTitle>
            </CardHeader>
            <CardContent>
              {!winResults ? (
                <div className="text-sm text-gray-500">未実行</div>
              ) : (
                <table className="w-full text-sm">
                  <thead>
                    <tr><th className="text-left p-1">期間</th><th className="text-right p-1">最適ラグ</th><th className="text-right p-1">r</th><th className="text-right p-1">p_adj</th></tr>
                  </thead>
                  <tbody>
                    {winResults.map((w, i) => (
                      <tr key={i} className="border-b">
                        <td className="p-1">{w.window_start} — {w.window_end}</td>
                        <td className="p-1 text-right">{w.best_lag}</td>
                        <td className="p-1 text-right">{w.r?.toFixed?.(3)}</td>
                        <td className="p-1 text-right">{Number(w.p_adj ?? w.p).toExponential?.(2)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>グランジャー因果</CardTitle>
          </CardHeader>
          <CardContent>
            {!granger ? (
              <div className="text-sm text-gray-500">未実行</div>
            ) : (
              <div className="text-sm space-y-1">
                <div>方向: <b>{granger.direction}</b>（order={granger.order}, gran={granger.granularity}）</div>
                <div>x→y: F={granger.x_to_y?.F?.toFixed?.(3)} p={Number(granger.x_to_y?.p).toExponential?.(2)}</div>
                <div>y→x: F={granger.y_to_x?.F?.toFixed?.(3)} p={Number(granger.y_to_x?.p).toExponential?.(2)}</div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

    </div>
  );
}
