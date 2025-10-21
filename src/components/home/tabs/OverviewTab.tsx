import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  FileText,
  CheckCircle2,
  Database,
  MessageSquare,
  Zap,
  Brain,
} from 'lucide-react';

export function OverviewTab() {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <FileText className="h-5 w-5" />
          システム概要
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div>
          <h3 className="font-semibold text-lg mb-2">🎯 解決する課題</h3>
          <ul className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
            <li className="flex items-start gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-500 mt-0.5 flex-shrink-0" />
              <span><strong>需要予測の属人化:</strong> ベテラン社員の勘に依存 → AIによる客観的な予測</span>
            </li>
            <li className="flex items-start gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-500 mt-0.5 flex-shrink-0" />
              <span><strong>異常値の見逃し:</strong> 統計的な異常検知が不十分 → 3σ法による自動検知</span>
            </li>
            <li className="flex items-start gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-500 mt-0.5 flex-shrink-0" />
              <span><strong>学習データの欠如:</strong> 過去の知見が蓄積されない → RAGによる継続的学習</span>
            </li>
            <li className="flex items-start gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-500 mt-0.5 flex-shrink-0" />
              <span><strong>分析の時間コスト:</strong> 手動での相関分析 → ファイルアップロードで即座に分析</span>
            </li>
          </ul>
        </div>

        <div>
          <h3 className="font-semibold text-lg mb-2">💡 システムの特徴</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div className="p-3 bg-purple-50 dark:bg-purple-950/30 rounded-lg">
              <div className="flex items-center gap-2 mb-1">
                <Database className="h-4 w-4 text-purple-600" />
                <strong className="text-sm">RAG搭載</strong>
              </div>
              <p className="text-xs text-gray-600 dark:text-gray-400">
                過去のデータ・会話履歴から最適な回答を生成
              </p>
            </div>
            <div className="p-3 bg-blue-50 dark:bg-blue-950/30 rounded-lg">
              <div className="flex items-center gap-2 mb-1">
                <MessageSquare className="h-4 w-4 text-blue-600" />
                <strong className="text-sm">深掘り質問</strong>
              </div>
              <p className="text-xs text-gray-600 dark:text-gray-400">
                AIが不足情報を自動で質問（最大2回）
              </p>
            </div>
            <div className="p-3 bg-green-50 dark:bg-green-950/30 rounded-lg">
              <div className="flex items-center gap-2 mb-1">
                <Zap className="h-4 w-4 text-green-600" />
                <strong className="text-sm">リアルタイム分析</strong>
              </div>
              <p className="text-xs text-gray-600 dark:text-gray-400">
                CSV/Excelアップロードで即座に分析（3秒以内）
              </p>
            </div>
            <div className="p-3 bg-amber-50 dark:bg-amber-950/30 rounded-lg">
              <div className="flex items-center gap-2 mb-1">
                <Brain className="h-4 w-4 text-amber-600" />
                <strong className="text-sm">異常検知+学習</strong>
              </div>
              <p className="text-xs text-gray-600 dark:text-gray-400">
                3σ法で異常を検出し、原因を学習データ化
              </p>
            </div>
          </div>
        </div>

        <div>
          <h3 className="font-semibold text-lg mb-2">📊 技術スタック</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-2 text-xs">
            <Badge variant="outline">Next.js 15.5</Badge>
            <Badge variant="outline">TypeScript 5.0+</Badge>
            <Badge variant="outline">Go 1.21+</Badge>
            <Badge variant="outline">Gin Framework</Badge>
            <Badge variant="outline">Azure OpenAI</Badge>
            <Badge variant="outline">Qdrant Vector DB</Badge>
            <Badge variant="outline">Tailwind CSS</Badge>
            <Badge variant="outline">shadcn/ui</Badge>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
