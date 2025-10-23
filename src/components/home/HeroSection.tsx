import { Badge } from "@/components/ui/badge";
import { TrendingUp, Zap, Brain, BarChart3 } from 'lucide-react';

export function HeroSection() {
  return (
    <section className="relative overflow-hidden rounded-lg bg-gray-900 p-8 text-white">
      <div className="relative z-10">
        <div className="flex items-center gap-4 mb-4">
          <div className="p-3 bg-[#11a2f0]/20 rounded-lg">
            <TrendingUp className="h-8 w-8 text-[#11a2f0]" />
          </div>
          <div>
            <h1 className="text-4xl font-bold text-white mb-1">HUNT</h1>
            <p className="text-lg text-gray-300">Highly Unified Needs Tracker</p>
          </div>
        </div>
        <p className="text-lg mb-6 max-w-2xl text-gray-200">
          製造業向けの次世代AI需要予測・異常検知・学習システム
        </p>
        <div className="flex gap-3 flex-wrap">
          <Badge variant="secondary" className="bg-[#11a2f0]/20 text-[#11a2f0] border-transparent">
            <Zap className="h-3 w-3 mr-1.5" />
            RAG搭載
          </Badge>
          <Badge variant="secondary" className="bg-[#11a2f0]/20 text-[#11a2f0] border-transparent">
            <Brain className="h-3 w-3 mr-1.5" />
            継続的学習
          </Badge>
          <Badge variant="secondary" className="bg-[#11a2f0]/20 text-[#11a2f0] border-transparent">
            <BarChart3 className="h-3 w-3 mr-1.5" />
            リアルタイム分析
          </Badge>
        </div>
      </div>
    </section>
  );
}
