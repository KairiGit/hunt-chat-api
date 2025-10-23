"use client";

import { useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

// ローカルタイムベースで YYYY-MM-DD を生成
const toLocalISODate = (d: Date) => {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
};

interface Point { date?: string; period?: string; value?: number; sales?: number }
interface LagResult { factor?: string; r: number; p: number; n: number; lag: number }
interface WindowResult { window_start: string; window_end: string; best_lag: number; r: number; p: number; p_adj?: number; n: number }
interface GrangerStat { F?: number; p?: number }
interface GrangerResult { direction?: string; order?: number; granularity?: string; x_to_y?: GrangerStat; y_to_x?: GrangerStat }

export default function EconAnalyticsPage() {
  const [symbol, setSymbol] = useState("NIKKEI");
  const [productId, setProductId] = useState("P001");
  const [granularity, setGranularity] = useState<'daily'|'weekly'|'monthly'>("weekly");
  const [start, setStart] = useState<string>(() => {
    const d = new Date();
    d.setDate(d.getDate() - 90); // 直近90日をデフォルト
    return toLocalISODate(d);
  });
  const [end, setEnd] = useState<string>(() => toLocalISODate(new Date()));
  const [maxLag, setMaxLag] = useState(21);
  const [series, setSeries] = useState<Point[]>([]);
  const [sales, setSales] = useState<Point[]>([]);
  const [lagResults, setLagResults] = useState<LagResult[]|null>(null);
  const [winResults, setWinResults] = useState<WindowResult[]|null>(null);
  const [granger, setGranger] = useState<GrangerResult|null>(null);
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState("");

  const fetchSeries = async () => {
    setBusy(true); setMsg("");
    try {
      const url = `/api/proxy/econ/series?symbol=${encodeURIComponent(symbol)}&start=${start}&end=${end}&granularity=${granularity}`;
      const res = await fetch(url);
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error || res.statusText);
      setSeries(data.series || []);
      setMsg(`✅ 指標 ${symbol} ${granularity} ${data.count}件`);
  } catch (e: unknown) { const msg = e instanceof Error ? e.message : String(e); setMsg(`❌ ${msg}`); } finally { setBusy(false); }
  };

  const fetchSales = async () => {
    setBusy(true); setMsg("");
    try {
      const url = `/api/proxy/econ/sales/series?product_id=${encodeURIComponent(productId)}&start=${start}&end=${end}&granularity=${granularity}`;
      const res = await fetch(url);
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error || res.statusText);
      setSales(data.series || []);
      setMsg(`✅ 売上 ${productId} ${granularity} ${data.count}件`);
  } catch (e: unknown) { const msg = e instanceof Error ? e.message : String(e); setMsg(`❌ ${msg}`); } finally { setBusy(false); }
  };

  const runLag = async () => {
    setBusy(true); setMsg(""); setLagResults(null);
    try {
      // build sales daily array for endpoint expecting explicit sales when using /econ/lagged-correlation
      const salesBody = sales.map(p => ({ date: p.date || p.period!, sales: p.sales ?? p.value ?? 0 }));
      const res = await fetch('/api/proxy/econ/lagged-correlation', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ symbol, start, end, sales: salesBody, max_lag: maxLag, granularity })
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error || res.statusText);
      setLagResults(data.results || []);
      setMsg(`✅ 相関計算 OK (top lag=${data.top?.lag}, r=${data.top?.correlation_coef?.toFixed?.(3)})`);
  } catch (e: unknown) { const msg = e instanceof Error ? e.message : String(e); setMsg(`❌ ${msg}`); } finally { setBusy(false); }
  };

  const runWindowed = async () => {
    setBusy(true); setMsg(""); setWinResults(null);
    try {
      const body = { product_id: productId, symbol, start, end, max_lag: maxLag, window_days: 90, step_days: 30, granularity };
      const res = await fetch('/api/proxy/econ/sales/lagged-correlation/windowed', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body)
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error || res.statusText);
      setWinResults(data.windows || []);
      setMsg(`✅ スライディング窓 OK (${(data.windows||[]).length}件)`);
  } catch (e: unknown) { const msg = e instanceof Error ? e.message : String(e); setMsg(`❌ ${msg}`); } finally { setBusy(false); }
  };

  const runGranger = async () => {
    setBusy(true); setMsg(""); setGranger(null);
    try {
      const body = { product_id: productId, symbol, start, end, order: 3, granularity };
      const res = await fetch('/api/proxy/econ/sales/granger', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error || res.statusText);
      setGranger(data);
      setMsg(`✅ グランジャー OK (${data.direction})`);
  } catch (e: unknown) { const msg = e instanceof Error ? e.message : String(e); setMsg(`❌ ${msg}`); } finally { setBusy(false); }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 p-6">
      <div className="max-w-6xl mx-auto space-y-6">
        <div>
          <h1 className="text-3xl font-bold text-gray-800 mb-2">📈 経済×売上 分析</h1>
          <p className="text-gray-600">Qdrantの系列参照、集約、ラグ相関、スライディング窓、因果性テストを試すミニUI</p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>条件</CardTitle>
            <CardDescription>シンボル・製品・期間・粒度を指定</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-6 gap-3">
              <div>
                <Label>シンボル</Label>
                <Input value={symbol} onChange={(e)=>setSymbol(e.target.value.toUpperCase())} />
              </div>
              <div>
                <Label>製品ID</Label>
                <Input value={productId} onChange={(e)=>setProductId(e.target.value)} />
              </div>
              <div>
                <Label>粒度</Label>
                <select value={granularity} onChange={(e)=>setGranularity(e.target.value as 'daily'|'weekly'|'monthly')} className="w-full p-2 border rounded-md">
                  <option value="daily">日次</option>
                  <option value="weekly">週次</option>
                  <option value="monthly">月次</option>
                </select>
              </div>
              <div>
                <Label>開始</Label>
                <Input type="date" value={start} onChange={(e)=>setStart(e.target.value)} />
              </div>
              <div>
                <Label>終了</Label>
                <Input type="date" value={end} onChange={(e)=>setEnd(e.target.value)} />
              </div>
              <div>
                <Label>最大ラグ(日)</Label>
                <Input type="number" value={maxLag} onChange={(e)=>setMaxLag(parseInt(e.target.value||'0',10))} />
              </div>
            </div>
            <div className="flex gap-2 mt-4">
              <Button onClick={fetchSeries} disabled={busy}>指標取得</Button>
              <Button onClick={fetchSales} disabled={busy}>売上取得</Button>
              <Button onClick={runLag} disabled={busy || sales.length===0}>ラグ相関</Button>
              <Button onClick={runWindowed} disabled={busy}>窓分析</Button>
              <Button onClick={runGranger} disabled={busy}>因果性</Button>
            </div>
            {msg && <div className={`mt-2 text-sm ${msg.startsWith('✅') ? 'text-green-700' : 'text-red-600'}`}>{msg}</div>}
          </CardContent>
        </Card>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Card>
            <CardHeader>
              <CardTitle>経済系列</CardTitle>
              <CardDescription>{symbol} {granularity}</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="max-h-64 overflow-auto text-sm">
                {series.length===0 ? <div className="text-gray-500">なし</div> : (
                  <table className="w-full">
                    <thead><tr><th className="text-left p-1">日付/期間</th><th className="text-right p-1">値</th></tr></thead>
                    <tbody>
            {series.map((p,i)=> (
                        <tr key={i} className="border-b">
                          <td className="p-1">{p.date || p.period}</td>
              <td className="p-1 text-right">{p.value !== undefined ? p.value.toFixed(2) : ''}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>売上系列</CardTitle>
              <CardDescription>{productId} {granularity}</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="max-h-64 overflow-auto text-sm">
                {sales.length===0 ? <div className="text-gray-500">なし</div> : (
                  <table className="w-full">
                    <thead><tr><th className="text-left p-1">日付/期間</th><th className="text-right p-1">売上</th></tr></thead>
                    <tbody>
            {sales.map((p,i)=> (
                        <tr key={i} className="border-b">
                          <td className="p-1">{p.date || p.period}</td>
              <td className="p-1 text-right">{(p.sales ?? p.value) !== undefined ? (p.sales ?? p.value)!.toFixed(2) : ''}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Card>
            <CardHeader>
              <CardTitle>ラグ相関（全体）</CardTitle>
            </CardHeader>
            <CardContent>
              {!lagResults ? <div className="text-sm text-gray-500">未実行</div> : (
                <table className="w-full text-sm">
                  <thead><tr><th className="text-left p-1">ラグ</th><th className="text-right p-1">r</th><th className="text-right p-1">p</th><th className="text-right p-1">n</th></tr></thead>
                  <tbody>
          {lagResults.map((r,i)=> (
                      <tr key={i} className="border-b">
                        <td className="p-1">{r.lag}</td>
            <td className="p-1 text-right">{r.r.toFixed(3)}</td>
            <td className="p-1 text-right">{r.p.toExponential(2)}</td>
                        <td className="p-1 text-right">{r.n}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>スライディング窓 最適ラグ</CardTitle>
            </CardHeader>
            <CardContent>
              {!winResults ? <div className="text-sm text-gray-500">未実行</div> : (
                <table className="w-full text-sm">
                  <thead><tr><th className="text-left p-1">期間</th><th className="text-right p-1">最適ラグ</th><th className="text-right p-1">r</th><th className="text-right p-1">p_adj</th></tr></thead>
                  <tbody>
          {winResults.map((w,i)=> (
                      <tr key={i} className="border-b">
                        <td className="p-1">{w.window_start} — {w.window_end}</td>
                        <td className="p-1 text-right">{w.best_lag}</td>
            <td className="p-1 text-right">{w.r.toFixed(3)}</td>
            <td className="p-1 text-right">{Number(w.p_adj ?? w.p).toExponential(2)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>グランジャー因果</CardTitle>
          </CardHeader>
          <CardContent>
      {!granger ? <div className="text-sm text-gray-500">未実行</div> : (
              <div className="text-sm space-y-1">
                <div>方向: <b>{granger.direction}</b>（order={granger.order}, gran={granger.granularity}）</div>
        <div>x→y: F={granger.x_to_y?.F !== undefined ? granger.x_to_y.F.toFixed(3) : ''} p={granger.x_to_y?.p !== undefined ? Number(granger.x_to_y.p).toExponential(2) : ''}</div>
        <div>y→x: F={granger.y_to_x?.F !== undefined ? granger.y_to_x.F.toFixed(3) : ''} p={granger.y_to_x?.p !== undefined ? Number(granger.y_to_x.p).toExponential(2) : ''}</div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
