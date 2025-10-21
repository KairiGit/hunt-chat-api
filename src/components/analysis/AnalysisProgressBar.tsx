'use client';

import { useEffect, useState } from 'react';
import { Progress } from '@/components/ui/progress';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { CheckCircle2, Loader2, Clock } from 'lucide-react';

interface AnalysisProgressBarProps {
  isAnalyzing: boolean;
  onComplete?: () => void;
  onError?: (error: string) => void;
}

const STEPS = [
  { key: 'init', name: 'ファイル読み込み', icon: '📁', duration: 500 },
  { key: 'parse', name: 'CSV解析', icon: '🔍', duration: 800 },
  { key: 'stats', name: '統計分析', icon: '📊', duration: 1500 },
  { key: 'ai', name: 'AI分析', icon: '🤖', duration: 3000 },
  { key: 'anomaly', name: '異常検知', icon: '⚠️', duration: 800 },
  { key: 'save', name: 'データベース保存', icon: '💾', duration: 500 },
  { key: 'complete', name: '完了', icon: '✅', duration: 0 },
];

export function AnalysisProgressBar({ isAnalyzing, onComplete, onError }: AnalysisProgressBarProps) {
  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [completedSteps, setCompletedSteps] = useState<Set<number>>(new Set());
  const [startTime, setStartTime] = useState<number>(0);
  const [elapsedTime, setElapsedTime] = useState<number>(0);

  useEffect(() => {
    if (!isAnalyzing) {
      // リセット
      setCurrentStepIndex(0);
      setCompletedSteps(new Set());
      setStartTime(0);
      setElapsedTime(0);
      return;
    }

    // 分析開始時
    if (startTime === 0) {
      setStartTime(Date.now());
    }

    // タイマーで経過時間を更新
    const timer = setInterval(() => {
      setElapsedTime(Date.now() - startTime);
    }, 100);

    // 各ステップをシミュレート
    let stepTimer: NodeJS.Timeout;
    const runNextStep = (stepIndex: number) => {
      if (stepIndex >= STEPS.length || !isAnalyzing) {
        return;
      }

      setCurrentStepIndex(stepIndex);

      const step = STEPS[stepIndex];
      stepTimer = setTimeout(() => {
        setCompletedSteps((prev) => new Set([...prev, stepIndex]));
        
        if (stepIndex < STEPS.length - 1) {
          runNextStep(stepIndex + 1);
        } else if (onComplete) {
          onComplete();
        }
      }, step.duration);
    };

    runNextStep(0);

    return () => {
      clearInterval(timer);
      clearTimeout(stepTimer);
    };
  }, [isAnalyzing, startTime, onComplete]);

  const formatElapsedTime = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    const seconds = Math.floor(ms / 1000);
    if (seconds < 60) return `${seconds}秒`;
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    return `${minutes}分${remainingSeconds}秒`;
  };

  const progress = Math.round(((completedSteps.size + (currentStepIndex > 0 ? 0.5 : 0)) / STEPS.length) * 100);

  if (!isAnalyzing) return null;

  return (
    <Card className="h-full border-blue-200 dark:border-blue-800 bg-blue-50/50 dark:bg-blue-950/20">
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2 text-blue-900 dark:text-blue-100 text-lg">
          <Loader2 className="h-5 w-5 animate-spin text-blue-600 dark:text-blue-400" />
          分析中...
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* 全体の進捗バー */}
        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span className="font-medium text-blue-900 dark:text-blue-100">
              {STEPS[currentStepIndex]?.name || '処理中...'}
            </span>
            <span className="font-semibold text-blue-700 dark:text-blue-300">{progress}%</span>
          </div>
          <Progress value={progress} className="h-2" />
          <div className="flex items-center gap-2 text-xs text-blue-700 dark:text-blue-300">
            <Clock className="h-3 w-3" />
            <span>経過時間: {formatElapsedTime(elapsedTime)}</span>
          </div>
        </div>

        {/* ステップ一覧 - コンパクト版 */}
        <div className="space-y-1.5">
          {STEPS.map((step, index) => {
            const isCompleted = completedSteps.has(index);
            const isCurrent = currentStepIndex === index;

            return (
              <div
                key={step.key}
                className={`flex items-center gap-2 p-2 rounded-md transition-all duration-300 ${
                  isCurrent
                    ? 'bg-blue-100 dark:bg-blue-900/30 border border-blue-300 dark:border-blue-700'
                    : isCompleted
                    ? 'bg-green-50 dark:bg-green-950/20'
                    : 'bg-gray-50 dark:bg-gray-900/50'
                }`}
              >
                <div className="flex-shrink-0">
                  {isCompleted ? (
                    <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
                  ) : isCurrent ? (
                    <Loader2 className="h-4 w-4 animate-spin text-blue-600 dark:text-blue-400" />
                  ) : (
                    <div className="h-4 w-4 rounded-full border-2 border-gray-300 dark:border-gray-700" />
                  )}
                </div>
                <span className="text-lg">{step.icon}</span>
                <span
                  className={`text-sm font-medium flex-1 ${
                    isCurrent
                      ? 'text-blue-900 dark:text-blue-100'
                      : isCompleted
                      ? 'text-green-900 dark:text-green-100'
                      : 'text-gray-500 dark:text-gray-400'
                  }`}
                >
                  {step.name}
                </span>
                {isCompleted && (
                  <span className="text-xs font-semibold text-green-600 dark:text-green-400">✓</span>
                )}
                {isCurrent && (
                  <span className="text-xs font-semibold text-blue-600 dark:text-blue-400">
                    ...
                  </span>
                )}
              </div>
            );
          })}
        </div>

        {/* 非同期処理の注記 - コンパクト版 */}
        {completedSteps.has(3) && ( // AI分析ステップ完了後
          <div className="mt-3 p-2.5 bg-purple-50 dark:bg-purple-950/20 border border-purple-200 dark:border-purple-800 rounded-md">
            <div className="flex items-start gap-2">
              <span className="text-base">💡</span>
              <div className="text-xs text-purple-900 dark:text-purple-100">
                <p className="font-semibold">非同期処理について</p>
                <p className="text-[11px] mt-0.5 leading-relaxed">
                  AI分析とAI質問生成はバックグラウンドで継続実行されます。
                  結果は後から自動的に更新されます（2-15秒程度）。
                </p>
              </div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
