'use client';

import { useState, useRef } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useAppContext } from '@/contexts/AppContext';
import { AnalysisReportView } from '@/components/analysis/AnalysisReportView';
import type { AnalysisReport, AnalysisResponse } from '@/types/analysis';

export default function AnalysisPage() {
  const { analysisSummary, setAnalysisSummary } = useAppContext();

  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [warning, setWarning] = useState<string | null>(null);
  const [analysisReport, setAnalysisReport] = useState<AnalysisReport | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      setSelectedFile(e.target.files[0]);
    }
  };

  const handleFileAnalysis = async () => {
    if (!selectedFile) return;

    setIsLoading(true);
    setError(null);
    setWarning(null);
    setAnalysisSummary('');
    setAnalysisReport(null);
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
      
      // 🔍 デバッグ情報をコンソールに出力
      console.log('🔵 [Client] レスポンス全体:', result);
      console.log('🔵 [Client] デバッグ情報:', result.debug);
      if (result.debug) {
        console.log('📋 ヘッダー:', result.debug.header);
        console.log('📊 列インデックス:', {
          date: result.debug.date_col_index,
          product: result.debug.product_col_index,
          sales: result.debug.sales_col_index,
        });
        console.log('📈 解析結果:', {
          total: result.debug.total_rows,
          successful: result.debug.successful_parses,
          failed: result.debug.failed_parses,
        });
        if (result.debug.parse_errors && result.debug.parse_errors.length > 0) {
          console.log('⚠️ 解析エラー:', result.debug.parse_errors);
        }
        if (result.debug.first_3_rows) {
          console.log('📋 最初の3行:', result.debug.first_3_rows);
        }
      }
      
      // エラーメッセージがある場合
      if (result.error) {
        throw new Error(result.error);
      }
      
      // 成功時の処理
      if (result.success) {
        setAnalysisSummary(result.summary || '');
        
        // analysis_reportがある場合のみ設定
        if (result.analysis_report) {
          setAnalysisReport(result.analysis_report);
        } else {
          // レポートがない場合は警告を表示（より詳細な情報を含める）
          const warningMessage = result.error 
            ? `詳細レポートの生成に失敗しました。${result.error}`
            : '基本的な分析は完了しましたが、詳細レポートの生成に失敗しました。データ量や気象データの取得に問題がある可能性があります。';
          setWarning(warningMessage);
          console.warn('分析は成功しましたが、詳細レポートが生成されませんでした', result);
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

      {/* サマリー表示（レポートがない場合も表示） */}
      {analysisSummary && !analysisReport && (
        <Card className="max-w-4xl">
          <CardHeader>
            <CardTitle>📊 基本分析結果</CardTitle>
            <CardDescription>
              ファイルの基本的な分析が完了しました
            </CardDescription>
          </CardHeader>
          <CardContent>
            <pre className="whitespace-pre-wrap text-sm bg-gray-50 dark:bg-gray-900 p-4 rounded-md overflow-auto max-h-96">
              {analysisSummary}
            </pre>
          </CardContent>
        </Card>
      )}

      {/* 詳細分析レポート表示 */}
      {analysisReport && (
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <h2 className="text-xl font-bold">② 分析レポート</h2>
            <span className="text-sm text-muted-foreground">
              この内容はチャットページで引き継がれます
            </span>
          </div>
          <AnalysisReportView report={analysisReport} />
        </div>
      )}
    </div>
  );
}
