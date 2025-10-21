import { Badge } from "@/components/ui/badge";
import { TrendingUp, Zap, Brain, BarChart3 } from 'lucide-react';

export function HeroSection() {
  return (
    <section className="relative overflow-hidden rounded-lg bg-gradient-to-br from-blue-600 via-purple-600 to-indigo-700 p-8 text-white">
      <div className="relative z-10">
        <div className="flex items-center gap-3 mb-4">
          <div className="p-3 bg-white/20 rounded-lg backdrop-blur-sm">
            <TrendingUp className="h-8 w-8" />
          </div>
          <div>
            <h1 className="text-4xl font-bold mb-2">HUNT</h1>
            <p className="text-xl text-blue-100">Highly Unified Needs Tracker</p>
          </div>
        </div>
        <p className="text-lg mb-6 max-w-2xl text-blue-50">
          製造業向けの次世代AI需要予測・異常検知・学習システム
        </p>
        <div className="flex gap-3 flex-wrap">
          <Badge variant="secondary" className="bg-white/20 text-white border-white/30">
            <Zap className="h-3 w-3 mr-1" />
            RAG搭載
          </Badge>
          <Badge variant="secondary" className="bg-white/20 text-white border-white/30">
            <Brain className="h-3 w-3 mr-1" />
            継続的学習
          </Badge>
          <Badge variant="secondary" className="bg-white/20 text-white border-white/30">
            <BarChart3 className="h-3 w-3 mr-1" />
            リアルタイム分析
          </Badge>
        </div>
      </div>
      <div className="absolute top-0 right-0 w-64 h-64 bg-white/10 rounded-full blur-3xl"></div>
      <div className="absolute bottom-0 left-0 w-96 h-96 bg-purple-500/20 rounded-full blur-3xl"></div>
    </section>
  );
}
