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
      if (result.success && result.analysis_report) {
        setAnalysisSummary(result.summary);
        setAnalysisReport(result.analysis_report);
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

      {/* 分析レポート表示 */}
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
