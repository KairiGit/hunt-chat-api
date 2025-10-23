import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  GitBranch,
  FileText,
  Settings,
  Brain,
  Zap,
  ExternalLink,
} from 'lucide-react';

export function ArchitectureTab() {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <GitBranch className="h-5 w-5" />
          システムアーキテクチャ
        </CardTitle>
        <CardDescription>
          詳細なシーケンス図とフロー図は <a href="https://github.com/KairiGit/hunt-chat-api/blob/main/docs/architecture/UML.md" target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline inline-flex items-center gap-1">
            UML.md <ExternalLink className="h-3 w-3" />
          </a> を参照
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div>
          <h3 className="font-semibold text-lg mb-3">コンポーネント構成</h3>
          <div className="space-y-3">
            <div className="p-3 bg-blue-50 dark:bg-blue-950/30 rounded-lg">
              <strong className="text-sm flex items-center gap-2">
                <FileText className="h-4 w-4 text-blue-600" />
                フロントエンド
              </strong>
              <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                Next.js 15 + TypeScript + Tailwind CSS + shadcn/ui
              </p>
            </div>
            <div className="p-3 bg-green-50 dark:bg-green-950/30 rounded-lg">
              <strong className="text-sm flex items-center gap-2">
                <Settings className="h-4 w-4 text-green-600" />
                バックエンド
              </strong>
              <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                Go 1.21+ + Gin Framework（Vercelサーバーレス対応）
              </p>
            </div>
            <div className="p-3 bg-purple-50 dark:bg-purple-950/30 rounded-lg">
              <strong className="text-sm flex items-center gap-2">
                <Brain className="h-4 w-4 text-purple-600" />
                AI/データ
              </strong>
              <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                Azure OpenAI (GPT-4) + Qdrant Vector Database
              </p>
            </div>
          </div>
        </div>

        <div>
          <h3 className="font-semibold text-lg mb-3">非同期処理によるパフォーマンス最適化</h3>
          <div className="p-4 bg-gradient-to-r from-green-50 to-blue-50 dark:from-green-950/30 dark:to-blue-950/30 rounded-lg">
            <div className="flex items-center gap-2 mb-2">
              <Zap className="h-5 w-5 text-green-600" />
              <strong>70% レスポンスタイム短縮</strong>
            </div>
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-xs text-gray-500 mb-1">改善前</p>
                <p className="text-2xl font-bold text-red-600">~10秒</p>
              </div>
              <div>
                <p className="text-xs text-gray-500 mb-1">改善後</p>
                <p className="text-2xl font-bold text-green-600">~3秒</p>
              </div>
            </div>
            <p className="text-xs text-gray-600 dark:text-gray-400 mt-3">
              AI分析とAI質問生成をGoroutineで並列実行し、ユーザーには基本分析結果を即座に返却
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
