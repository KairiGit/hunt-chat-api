import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ArrowRight } from 'lucide-react';
import Link from 'next/link';

export function FeaturesTab() {
  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>📊 ファイル分析</CardTitle>
          <CardDescription>CSV/Excelファイルをアップロードして自動分析</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div>
            <strong className="text-sm">実行される分析:</strong>
            <ul className="mt-2 space-y-1 text-sm text-gray-600 dark:text-gray-400">
              <li>• <strong>基本統計:</strong> 平均、標準偏差、最大/最小値</li>
              <li>• <strong>相関分析:</strong> 気温・湿度・降水量との相関係数</li>
              <li>• <strong>回帰分析:</strong> 線形回帰による予測モデル構築</li>
              <li>• <strong>異常検知:</strong> 3σ法による統計的異常値の検出</li>
              <li>• <strong>AIレポート:</strong> Azure OpenAI による洞察生成</li>
            </ul>
          </div>
          <div>
            <strong className="text-sm">パフォーマンス:</strong>
            <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
              平均レスポンスタイム: <strong className="text-green-600">3秒</strong> （非同期処理により70%短縮）
            </p>
          </div>
          <Link href="/analysis">
            <div className="mt-4 flex items-center gap-2 text-sm text-blue-600 hover:text-blue-700 cursor-pointer">
              <span>ファイル分析を試す</span>
              <ArrowRight className="h-4 w-4" />
            </div>
          </Link>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>🔮 需要予測</CardTitle>
          <CardDescription>製品別・期間別の需要予測</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div>
            <strong className="text-sm">予測機能:</strong>
            <ul className="mt-2 space-y-1 text-sm text-gray-600 dark:text-gray-400">
              <li>• 製品別 × 1週間 → 日別予測 + 信頼区間</li>
              <li>• 製品別 × 1ヶ月 → 週次平均 + 季節性分析</li>
              <li>• 気温ベース予測: 回帰式による需要予測</li>
            </ul>
          </div>
          <Link href="/dashboard">
            <div className="mt-4 flex items-center gap-2 text-sm text-blue-600 hover:text-blue-700 cursor-pointer">
              <span>需要予測を試す</span>
              <ArrowRight className="h-4 w-4" />
            </div>
          </Link>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>🤖 RAGチャット</CardTitle>
          <CardDescription>過去データを活用したAI対話</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div>
            <strong className="text-sm">検索対象:</strong>
            <ul className="mt-2 space-y-1 text-sm text-gray-600 dark:text-gray-400">
              <li>• 過去の会話履歴</li>
              <li>• 分析レポート</li>
              <li>• 異常対応記録</li>
              <li>• システムドキュメント</li>
            </ul>
          </div>
          <Link href="/chat">
            <div className="mt-4 flex items-center gap-2 text-sm text-blue-600 hover:text-blue-700 cursor-pointer">
              <span>AIチャットを試す</span>
              <ArrowRight className="h-4 w-4" />
            </div>
          </Link>
        </CardContent>
      </Card>
    </div>
  );
}
