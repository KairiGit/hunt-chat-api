import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { AnalysisReport } from '@/types/analysis';

interface AnalysisReportViewProps {
  report: AnalysisReport;
}

// 深刻度に応じて色を返すヘルパー関数
const getSeverityColor = (severity: string) => {
  switch (severity) {
    case 'critical': return 'bg-red-500';
    case 'high': return 'bg-orange-500';
    case 'medium': return 'bg-yellow-500';
    default: return 'bg-blue-500';
  }
};

export function AnalysisReportView({ report }: AnalysisReportViewProps) {
  return (
    <div className="space-y-6">
      {/* ヘッダー情報 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            📊 分析レポート
            <span className="text-sm font-normal text-muted-foreground">
              {report.report_id}
            </span>
          </CardTitle>
          <CardDescription>
            {new Date(report.analysis_date).toLocaleString('ja-JP')} に生成
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
              <p className="text-muted-foreground">ファイル名</p>
              <p className="font-medium">{report.file_name}</p>
            </div>
            <div>
              <p className="text-muted-foreground">データ期間</p>
              <p className="font-medium">{report.date_range}</p>
            </div>
            <div>
              <p className="text-muted-foreground">データ点数</p>
              <p className="font-medium">{report.data_points.toLocaleString()} 件</p>
            </div>
            <div>
              <p className="text-muted-foreground">天気マッチ</p>
              <p className="font-medium">{report.weather_matches.toLocaleString()} 件</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 異常検知結果 */}
      {report.anomalies && report.anomalies.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>🔍 検出された異常</CardTitle>
            <CardDescription>
              AIが質問を生成しました。AI学習ページで回答することで学習データになります。
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {report.anomalies.map((anomaly, index) => (
                <div
                  key={index}
                  className={`p-4 rounded-lg border-2 border-gray-200`}
                >
                  <div className="flex items-start justify-between mb-2">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <span className={`px-2 py-1 rounded text-xs text-white ${getSeverityColor(anomaly.severity)}`}>
                          {anomaly.severity.toUpperCase()}
                        </span>
                        <span className="text-sm text-gray-600">{anomaly.date}</span>
                      </div>
                      <div className="text-sm">
                        <span className="font-semibold">{anomaly.anomaly_type}</span>
                        <span className="text-gray-600 ml-2">
                          実績: {anomaly.actual_value.toFixed(0)} (期待値: {anomaly.expected_value.toFixed(0)})
                        </span>
                      </div>
                    </div>
                  </div>
                  {anomaly.ai_question && (
                    <div className="mt-2 p-3 bg-blue-50 rounded">
                      <div className="text-sm font-medium text-blue-900 mb-1">💬 AIの質問:</div>
                      <div className="text-sm text-blue-800">{anomaly.ai_question}</div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* 統計サマリー */}
      <Card>
        <CardHeader>
          <CardTitle>📈 統計サマリー</CardTitle>
        </CardHeader>
        <CardContent>
          <pre className="text-sm whitespace-pre-wrap">{report.summary}</pre>
        </CardContent>
      </Card>

      {/* 天気相関分析 */}
      <Card>
        <CardHeader>
          <CardTitle>🌤️ 天気との相関分析</CardTitle>
          <CardDescription>
            売上と気象要素の相関関係を分析しました
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {report.correlations.map((corr, index) => (
              <div key={index} className="border rounded-lg p-4 space-y-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-2xl">
                      {corr.factor === 'temperature' ? '🌡️' : '💧'}
                    </span>
                    <div>
                      <p className="font-semibold">
                        {corr.factor === 'temperature' ? '気温' : '湿度'}
                      </p>
                      <p className="text-sm text-muted-foreground">
                        {corr.interpretation}
                      </p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="text-2xl font-bold">
                      {(corr.correlation_coef * 100).toFixed(1)}%
                    </p>
                    <p className="text-xs text-muted-foreground">
                      相関係数
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-4 text-sm">
                  <div>
                    <span className="text-muted-foreground">P値: </span>
                    <span className="font-medium">{corr.p_value.toFixed(3)}</span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">サンプル数: </span>
                    <span className="font-medium">{corr.sample_size.toLocaleString()}</span>
                  </div>
                </div>
                {/* 相関の強さを視覚化 */}
                <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                  <div
                    className={`h-2 rounded-full ${
                      Math.abs(corr.correlation_coef) > 0.5
                        ? 'bg-green-500'
                        : Math.abs(corr.correlation_coef) > 0.3
                        ? 'bg-yellow-500'
                        : 'bg-gray-400'
                    }`}
                    style={{ width: `${Math.abs(corr.correlation_coef) * 100}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* 回帰分析 */}
      {report.regression && (
        <Card>
          <CardHeader>
            <CardTitle>📉 回帰分析</CardTitle>
            <CardDescription>
              気温と売上の関係を数式で表現しました
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="bg-blue-50 dark:bg-blue-950 border-2 border-blue-200 dark:border-blue-800 rounded-lg p-4">
              <p className="text-center text-lg font-mono font-semibold">
                {report.regression.description}
              </p>
            </div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <div className="text-center p-3 bg-gray-50 dark:bg-gray-800 rounded">
                <p className="text-muted-foreground text-xs">傾き</p>
                <p className="font-bold text-lg">{report.regression.slope.toFixed(2)}</p>
              </div>
              <div className="text-center p-3 bg-gray-50 dark:bg-gray-800 rounded">
                <p className="text-muted-foreground text-xs">切片</p>
                <p className="font-bold text-lg">{report.regression.intercept.toFixed(2)}</p>
              </div>
              <div className="text-center p-3 bg-gray-50 dark:bg-gray-800 rounded">
                <p className="text-muted-foreground text-xs">決定係数 (R²)</p>
                <p className="font-bold text-lg">{(report.regression.r_squared * 100).toFixed(1)}%</p>
              </div>
              <div className="text-center p-3 bg-gray-50 dark:bg-gray-800 rounded">
                <p className="text-muted-foreground text-xs">予測値</p>
                <p className="font-bold text-lg">{report.regression.prediction.toFixed(0)}</p>
              </div>
            </div>
            <div className="text-sm text-muted-foreground space-y-1">
              <p>💡 <strong>解釈:</strong> 気温が1度上がると、売上が約{report.regression.slope.toFixed(2)}単位増加します。</p>
              <p>📊 決定係数 R² = {(report.regression.r_squared * 100).toFixed(1)}% は、気温の変化が売上変動の{(report.regression.r_squared * 100).toFixed(1)}%を説明できることを示しています。</p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* AI洞察 */}
      <Card>
        <CardHeader>
          <CardTitle>🤖 AI による洞察</CardTitle>
          <CardDescription>
            Azure OpenAI が分析結果から導き出した洞察です
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="prose prose-sm dark:prose-invert max-w-none">
            <div className="whitespace-pre-wrap text-sm leading-relaxed">
              {report.ai_insights}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 推奨事項 */}
      <Card>
        <CardHeader>
          <CardTitle>💡 推奨事項</CardTitle>
        </CardHeader>
        <CardContent>
          <ul className="space-y-2">
            {report.recommendations.map((rec, index) => (
              <li key={index} className="flex items-start gap-2">
                <span className="text-green-500 mt-1">✓</span>
                <span className="text-sm">{rec}</span>
              </li>
            ))}
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}