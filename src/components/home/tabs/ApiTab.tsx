import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { FileText, ExternalLink } from 'lucide-react';

const apiEndpoints = [
  {
    method: 'POST',
    path: '/api/v1/ai/analyze-file',
    description: 'ファイルアップロードと分析実行',
  },
  {
    method: 'POST',
    path: '/api/v1/ai/chat-input',
    description: 'RAGチャット入力',
  },
  {
    method: 'POST',
    path: '/api/v1/ai/analyze-weekly',
    description: '製品別分析（日次/週次/月次）',
  },
  {
    method: 'GET',
    path: '/api/v1/ai/unanswered-anomalies',
    description: '未回答異常の一覧取得',
  },
];

const documentLinks = [
  {
    title: 'UML・アーキテクチャ図',
    href: 'https://github.com/KairiGit/hunt-chat-api/blob/main/docs/architecture/UML.md',
  },
  {
    title: 'API マニュアル',
    href: 'https://github.com/KairiGit/hunt-chat-api/blob/main/docs/api/API_MANUAL.md',
  },
  {
    title: 'RAGシステムガイド',
    href: 'https://github.com/KairiGit/hunt-chat-api/blob/main/docs/guides/RAG_SYSTEM_GUIDE.md',
  },
  {
    title: '製品別分析ガイド',
    href: 'https://github.com/KairiGit/hunt-chat-api/blob/main/docs/guides/WEEKLY_ANALYSIS_GUIDE.md',
  },
];

export function ApiTab() {
  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="h-5 w-5" />
            主要APIエンドポイント
          </CardTitle>
          <CardDescription>
            詳細は <a href="https://github.com/KairiGit/hunt-chat-api/blob/main/docs/api/API_MANUAL.md" target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline inline-flex items-center gap-1">
              API_MANUAL.md <ExternalLink className="h-3 w-3" />
            </a> を参照
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {apiEndpoints.map((endpoint) => (
            <div key={endpoint.path} className="p-3 border rounded-lg">
              <div className="flex items-center gap-2 mb-1">
                <Badge variant="secondary">{endpoint.method}</Badge>
                <code className="text-xs font-mono">{endpoint.path}</code>
              </div>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                {endpoint.description}
              </p>
            </div>
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>ドキュメントリンク</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {documentLinks.map((link) => (
              <a
                key={link.href}
                href={link.href}
                target="_blank"
                rel="noopener noreferrer"
                className="p-3 border rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors flex items-center justify-between"
              >
                <span className="text-sm font-medium">{link.title}</span>
                <ExternalLink className="h-4 w-4 text-gray-400" />
              </a>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
