import { NextResponse } from 'next/server';

export const dynamic = 'force-dynamic';

export async function POST(request: Request) {
  let targetUrl = '';
  try {
    const formData = await request.formData();

    // Vercel環境ではVERCEL_URLを、ローカル環境ではGO_BACKEND_URLを使用
    const baseUrl = process.env.GO_BACKEND_URL || `https://${process.env.VERCEL_URL}`;
    targetUrl = `${baseUrl}/api/v1/ai/analyze-file`;

    const headers = new Headers();
    headers.append('X-API-KEY', process.env.API_KEY || '');

    const response = await fetch(targetUrl, {
      method: 'POST',
      headers: headers,
      body: formData,
    });

    const data = await response.json();

    // 🔍 詳細デバッグログ（バックエンドの重要情報を含む）
    console.log('[Proxy /analyze-file] Response status:', response.status);
    console.log('[Proxy /analyze-file] Has analysis_report:', 'analysis_report' in data);
    console.log('[Proxy /analyze-file] Data keys:', Object.keys(data));
    
    // バックエンドのデバッグ情報を確認
    if (data.debug) {
      console.log('[Proxy /analyze-file] Backend debug info:');
      console.log('  - Total rows:', data.debug.total_rows);
      console.log('  - Successful parses:', data.debug.successful_parses);
      console.log('  - Failed parses:', data.debug.failed_parses);
      console.log('  - Date column index:', data.debug.date_col_index);
      console.log('  - Product column index:', data.debug.product_col_index);
      console.log('  - Sales column index:', data.debug.sales_col_index);
      console.log('  - Header:', data.debug.header);
      if (data.debug.parse_errors && data.debug.parse_errors.length > 0) {
        console.log('  - Parse errors (first 5):', data.debug.parse_errors.slice(0, 5));
      }
    }
    
    // 販売データのカウント
    if (data.sales_data_count !== undefined) {
      console.log('[Proxy /analyze-file] Sales data count:', data.sales_data_count);
    }
    
    // 分析レポートの詳細
    if (data.analysis_report) {
      console.log('[Proxy /analyze-file] Analysis report details:');
      console.log('  - Report ID:', data.analysis_report.report_id);
      console.log('  - Date range:', data.analysis_report.date_range);
      console.log('  - Data points:', data.analysis_report.data_points);
      console.log('  - Weather matches:', data.analysis_report.weather_matches);
      console.log('  - Correlations count:', data.analysis_report.correlations?.length || 0);
    } else {
      console.warn('[Proxy /analyze-file] ⚠️ analysis_report is missing from response');
      if (data.error) {
        console.warn('[Proxy /analyze-file] Error message:', data.error);
      }
    }
    
    return NextResponse.json(data, {
      status: response.status,
    });

  } catch (error) {
    console.error('Proxy error in /api/proxy/analyze-file:', error);
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    return NextResponse.json(
      {
        error: 'An internal server error occurred in the proxy.',
        details: `Failed to fetch target: ${targetUrl}`,
        proxyError: errorMessage,
      },
      { status: 500 }
    );
  }
}