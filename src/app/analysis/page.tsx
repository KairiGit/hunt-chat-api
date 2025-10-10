'use client';

import { useState, useRef } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

export default function AnalysisPage() {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [summary, setSummary] = useState<string>('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
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
    setSummary('');
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

      const result = await response.json();
      if (result.success) {
        setSummary(result.summary);
      } else {
        throw new Error(result.error || 'Failed to get analysis summary.');
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

      {summary && (
        <Card className="max-w-2xl">
          <CardHeader>
            <CardTitle>② 分析サマリー</CardTitle>
            <CardDescription>AIによる分析結果の要約です。この内容はチャットページで引き継がれます。</CardDescription>
          </CardHeader>
          <CardContent>
            <pre className="bg-gray-100 dark:bg-gray-800 p-4 rounded-md text-sm whitespace-pre-wrap font-mono">
              {summary}
            </pre>
          </CardContent>
        </Card>
      )}
    </div>
  );
}