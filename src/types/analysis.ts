// 分析レポートの型定義

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
}

export interface AnalysisResponse {
  analysis_report?: AnalysisReport; // オプショナルに変更
  success: boolean;
  summary: string;
  error?: string; // エラーメッセージを追加
}
