'use client';

import { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

interface AnomalyDetection {
  date: string;
  actual_value: number;
  expected_value: number;
  deviation: number;
  z_score: number;
  anomaly_type: string;
  severity: string;
  ai_question?: string;
}

interface SalesPrediction {
  predicted_value: number;
  confidence_interval: {
    lower: number;
    upper: number;
    confidence: number;
  };
  confidence: number;
  prediction_factors: string[];
  regression_equation: string;
}

export default function DashboardPage() {
  const [anomalies, setAnomalies] = useState<AnomalyDetection[]>([]);
  const [prediction, setPrediction] = useState<SalesPrediction | null>(null);
  const [isLoadingAnomalies, setIsLoadingAnomalies] = useState(false);
  const [isLoadingPrediction, setIsLoadingPrediction] = useState(false);
  
  // 予測パラメータ
  const [futureTemp, setFutureTemp] = useState<number>(25);
  const [confidenceLevel, setConfidenceLevel] = useState<number>(0.95);

  // 異常検知用サンプルデータ
  const sampleSales = [100, 105, 110, 115, 95, 120, 300, 125, 130, 135, 140, 145, 50, 150];
  const sampleDates = [
    '2024-01-01', '2024-01-02', '2024-01-03', '2024-01-04', '2024-01-05',
    '2024-01-06', '2024-01-07', '2024-01-08', '2024-01-09', '2024-01-10',
    '2024-01-11', '2024-01-12', '2024-01-13', '2024-01-14',
  ];

  const detectAnomalies = async () => {
    setIsLoadingAnomalies(true);
    try {
      const response = await fetch('/api/v1/ai/detect-anomalies', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          sales: sampleSales,
          dates: sampleDates,
        }),
      });

      const data = await response.json();
      if (data.success) {
        setAnomalies(data.anomalies);
      }
    } catch (error) {
      console.error('異常検知エラー:', error);
    } finally {
      setIsLoadingAnomalies(false);
    }
  };

  const predictSales = async () => {
    setIsLoadingPrediction(true);
    try {
      const response = await fetch('/api/v1/ai/predict-sales', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          product_id: 'P001',
          future_temperature: futureTemp,
          confidence_level: confidenceLevel,
        }),
      });

      const data = await response.json();
      if (data.success) {
        setPrediction(data.prediction);
      }
    } catch (error) {
      console.error('予測エラー:', error);
    } finally {
      setIsLoadingPrediction(false);
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'bg-red-100 border-red-500 text-red-900 dark:bg-red-950 dark:text-red-100';
      case 'high':
        return 'bg-orange-100 border-orange-500 text-orange-900 dark:bg-orange-950 dark:text-orange-100';
      case 'medium':
        return 'bg-yellow-100 border-yellow-500 text-yellow-900 dark:bg-yellow-950 dark:text-yellow-100';
      default:
        return 'bg-gray-100 border-gray-500 text-gray-900 dark:bg-gray-800 dark:text-gray-100';
    }
  };

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold">📊 需要予測ダッシュボード</h1>
        <p className="text-muted-foreground mt-2">
          AI による異常検知と売上予測を実行できます
        </p>
      </div>

      {/* 売上予測セクション */}
      <Card>
        <CardHeader>
          <CardTitle>🔮 売上予測</CardTitle>
          <CardDescription>
            気温をもとに将来の売上を予測します（回帰分析 + 信頼区間）
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <Label htmlFor="temperature">予測気温 (°C)</Label>
              <Input
                id="temperature"
                type="number"
                value={futureTemp}
                onChange={(e) => setFutureTemp(Number(e.target.value))}
                step="0.1"
              />
            </div>
            <div>
              <Label htmlFor="confidence">信頼度</Label>
              <select
                id="confidence"
                className="w-full p-2 border rounded"
                value={confidenceLevel}
                onChange={(e) => setConfidenceLevel(Number(e.target.value))}
              >
                <option value={0.90}>90%</option>
                <option value={0.95}>95%</option>
                <option value={0.99}>99%</option>
              </select>
            </div>
          </div>
          
          <Button onClick={predictSales} disabled={isLoadingPrediction}>
            {isLoadingPrediction ? '予測中...' : '売上を予測'}
          </Button>

          {prediction && (
            <div className="mt-6 space-y-4">
              <div className="bg-blue-50 dark:bg-blue-950 border-2 border-blue-200 dark:border-blue-800 rounded-lg p-6">
                <div className="text-center">
                  <p className="text-sm text-muted-foreground">予測売上</p>
                  <p className="text-4xl font-bold text-blue-600 dark:text-blue-400">
                    {prediction.predicted_value.toFixed(0)} 個
                  </p>
                  <p className="text-sm text-muted-foreground mt-2">
                    {(confidenceLevel * 100).toFixed(0)}% 信頼区間: {' '}
                    {prediction.confidence_interval.lower.toFixed(0)} 〜 {' '}
                    {prediction.confidence_interval.upper.toFixed(0)} 個
                  </p>
                </div>
              </div>

              <div className="space-y-2">
                <p className="font-semibold">回帰式</p>
                <code className="block bg-gray-100 dark:bg-gray-800 p-2 rounded">
                  {prediction.regression_equation}
                </code>
              </div>

              <div className="space-y-2">
                <p className="font-semibold">予測根拠</p>
                <ul className="space-y-1">
                  {prediction.prediction_factors.map((factor, idx) => (
                    <li key={idx} className="flex items-start gap-2 text-sm">
                      <span className="text-green-500">✓</span>
                      <span>{factor}</span>
                    </li>
                  ))}
                </ul>
              </div>

              <div className="flex items-center gap-2">
                <div className="flex-1 bg-gray-200 dark:bg-gray-700 rounded-full h-4">
                  <div
                    className="bg-green-500 h-4 rounded-full"
                    style={{ width: `${prediction.confidence * 100}%` }}
                  />
                </div>
                <span className="text-sm font-medium">
                  信頼度: {(prediction.confidence * 100).toFixed(1)}%
                </span>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 異常検知セクション */}
      <Card>
        <CardHeader>
          <CardTitle>🚨 異常検知</CardTitle>
          <CardDescription>
            売上データから統計的異常値（3σ法）を検出し、AIが質問を生成します
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button onClick={detectAnomalies} disabled={isLoadingAnomalies}>
            {isLoadingAnomalies ? '検出中...' : '異常を検出'}
          </Button>

          {anomalies.length > 0 && (
            <div className="mt-6 space-y-4">
              <p className="text-sm font-semibold">
                {anomalies.length} 件の異常を検出しました
              </p>
              {anomalies.map((anomaly, idx) => (
                <Card
                  key={idx}
                  className={`border-2 ${getSeverityColor(anomaly.severity)}`}
                >
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-lg">
                      {anomaly.anomaly_type === '急増' ? '📈' : '📉'}
                      {anomaly.date} - {anomaly.anomaly_type}
                      <span className="text-xs font-normal px-2 py-1 rounded bg-white dark:bg-gray-800">
                        {anomaly.severity}
                      </span>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    <div className="grid grid-cols-3 gap-4 text-sm">
                      <div>
                        <p className="text-muted-foreground">実績値</p>
                        <p className="font-bold">{anomaly.actual_value.toFixed(0)}</p>
                      </div>
                      <div>
                        <p className="text-muted-foreground">期待値</p>
                        <p className="font-bold">{anomaly.expected_value.toFixed(0)}</p>
                      </div>
                      <div>
                        <p className="text-muted-foreground">偏差 (Z)</p>
                        <p className="font-bold">{anomaly.z_score.toFixed(2)}σ</p>
                      </div>
                    </div>

                    {anomaly.ai_question && (
                      <div className="mt-4 p-4 bg-white dark:bg-gray-900 rounded border">
                        <p className="font-semibold text-sm mb-2">🤖 AIからの質問</p>
                        <p className="text-sm whitespace-pre-wrap">{anomaly.ai_question}</p>
                        <Button className="mt-3" size="sm" variant="outline">
                          回答する
                        </Button>
                      </div>
                    )}
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
