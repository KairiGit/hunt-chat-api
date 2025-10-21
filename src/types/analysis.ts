// 分析レポートの型定義

export interface AnomalyDetection {
  date: string;
  product_id: string;
  actual_value: number;
  expected_value: number;
  deviation: number;
  z_score: number;
  anomaly_type: string;
  severity: string;
  ai_question?: string;
  question_choices?: string[];
}

export interface CorrelationResult {
  factor: string;
  correlation_coef: number;
  p_value: number;
  sample_size: number;
  interpretation: string;
}

export interface RegressionResult {
  slope: number;
  intercept: number;
  r_squared: number;
  prediction: number;
  confidence: number;
  description: string;
}

export interface AnalysisReport {
  report_id: string;
  file_name: string;
  analysis_date: string;
  data_points: number;
  date_range: string;
  weather_matches: number;
  summary: string;
  correlations: CorrelationResult[];
  regression: RegressionResult | null; // nullを許容
  ai_insights: string;
  recommendations: string[];
  anomalies: AnomalyDetection[];
}

export interface AnalysisResponse {
  analysis_report?: AnalysisReport; // オプショナルに変更
  success: boolean;
  summary: string;
  error?: string; // エラーメッセージを追加
  ai_insights_pending?: boolean; // 🆕 AI分析実行中フラグ
  ai_questions_pending?: boolean; // 🆕 AI質問生成中フラグ
  backend_version?: string; // 🆕 バックエンドバージョン
  sales_data_count?: number; // デバッグ用
  debug?: { // 🔍 デバッグ情報を追加
    header: string[];
    date_col_index: number;
    product_col_index: number;
    sales_col_index: number;
    total_rows: number;
    successful_parses: number;
    failed_parses: number;
    first_3_rows: string[][];
    parse_errors: string[];
  };
}

export interface AnalysisReportHeader {
  report_id: string;
  file_name: string;
  analysis_date: string;
  date_range: string;
}