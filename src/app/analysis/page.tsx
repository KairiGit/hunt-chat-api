'use client';

import { useState, useRef, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useAppContext } from '@/contexts/AppContext';
import { AnalysisReportView } from '@/components/analysis/AnalysisReportView';
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
  
  const [reportList, setReportList] = useState<AnalysisReportHeader[]>([]);
  const [isLoadingList, setIsLoadingList] = useState(true);
  
  const [selectedReport, setSelectedReport] = useState<AnalysisReport | null>(null);
  const [isLoadingReport, setIsLoadingReport] = useState(false);

  const [isReportDeleteDialogOpen, setReportDeleteDialogOpen] = useState(false);
  const [reportIdToDelete, setReportIdToDelete] = useState<string | null>(null);
  const [isDeleteAllReportsDialogOpen, setDeleteAllReportsDialogOpen] = useState(false);

  const fileInputRef = useRef<HTMLInputElement>(null);

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

  return (
    <div className="space-y-8">
      <h1 className="text-2xl font-bold">ファイル分析</h1>
      
      <Card className="max-w-2xl">
        <CardHeader>
          <CardTitle>① ファイルのアップロード</CardTitle>
          <CardDescription>分析したい販売実績データ（.xlsx, .csv）を選択してください。</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid w-full items-center gap-1.5">
            <Label htmlFor="file-upload">ファイル</Label>
            <Input id="file-upload" type="file" ref={fileInputRef} onChange={handleFileChange} accept=".xlsx, .csv" />
          </div>
        </CardContent>
        <CardFooter className="flex justify-end">
          <Button onClick={handleFileAnalysis} disabled={!selectedFile || isLoading}>
            {isLoading ? '分析中...' : '分析開始'}
          </Button>
        </CardFooter>
      </Card>

      {error && (
        <Card className="max-w-2xl bg-destructive/10 border-destructive">
          <CardHeader>
            <CardTitle className="text-destructive">分析エラー</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-destructive-foreground">{error}</p>
          </CardContent>
        </Card>
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

      {warning && (
        <Card className="max-w-2xl bg-yellow-50 dark:bg-yellow-950 border-yellow-500">
          <CardHeader>
            <CardTitle className="text-yellow-700 dark:text-yellow-400">⚠️ 警告</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-yellow-700 dark:text-yellow-300">{warning}</p>
          </CardContent>
        </Card>
      )}

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

    </div>
  );
}