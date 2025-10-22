"use client";

import { useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export default function SettingsPage() {
  const [symbol, setSymbol] = useState("NIKKEI");
  const [file, setFile] = useState<File | null>(null);
  const [csvText, setCsvText] = useState("");
  const [result, setResult] = useState<string>("");
  const [busy, setBusy] = useState(false);

  const submitImport = async () => {
    setBusy(true);
    setResult("");
    try {
  let res: Response;
      if (file) {
        const form = new FormData();
        form.append("symbol", symbol);
        form.append("file", file);
        res = await fetch("/api/proxy/econ/import", { method: "POST", body: form });
      } else if (csvText.trim()) {
        res = await fetch("/api/proxy/econ/import", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ symbol, csv_text: csvText }),
        });
      } else {
        setResult("❌ CSVファイルまたはCSVテキストを入力してください");
        return;
      }
      const ct = res.headers.get("content-type") || "";
      if (!res.ok) {
        if (ct.includes("application/json")) {
          const data = await res.json().catch(() => ({} as any));
          setResult(`❌ 失敗(${res.status}): ${data?.error || JSON.stringify(data) || res.statusText}`);
        } else {
          const text = await res.text().catch(() => "");
          setResult(`❌ 失敗(${res.status}): ${text || res.statusText}`);
        }
      } else {
        const data = ct.includes("application/json") ? await res.json().catch(() => ({} as any)) : {};
        setResult(`✅ 成功: ${data.symbol ?? symbol} を ${data.stored ?? 0} 件取り込み`);
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setResult(`❌ エラー: ${msg}`);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-50 p-8">
      <div className="max-w-3xl mx-auto space-y-6">
        <div>
          <h1 className="text-3xl font-bold text-gray-800 mb-2">⚙️ 設定</h1>
          <p className="text-gray-600">日経平均のCSVを取り込みます（重複は自動スキップ）</p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>📈 日経平均 取り込み</CardTitle>
            <CardDescription>CSVアップロード or テキスト貼り付けのどちらでもOK</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <Label htmlFor="symbol">シンボル</Label>
              <Input id="symbol" value={symbol} onChange={(e) => setSymbol(e.target.value.toUpperCase())} />
              <p className="text-xs text-gray-500 mt-1">例: NIKKEI（他の指数・銘柄名でも可）</p>
            </div>

            <div className="grid gap-2">
              <Label>CSVファイル</Label>
              <Input type="file" accept=".csv,text/csv" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
              <p className="text-xs text-gray-500">UTF-8推奨。Shift-JISは事前に変換してください。</p>
            </div>

            <div className="grid gap-2">
              <Label>または CSVテキスト</Label>
              <textarea
                className="w-full min-h-[140px] rounded-md border border-gray-300 p-2 text-sm"
                placeholder="Date,Close\n2024-01-04,35000\n..."
                value={csvText}
                onChange={(e) => setCsvText(e.target.value)}
              />
            </div>

            <div className="flex items-center gap-3">
              <Button onClick={submitImport} disabled={busy} className="px-6">
                {busy ? "取り込み中..." : "📥 取り込む"}
              </Button>
              {result && <span className={`text-sm ${result.startsWith("✅") ? "text-green-700" : "text-red-600"}`}>{result}</span>}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
