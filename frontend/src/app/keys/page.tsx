"use client";

import { useEffect, useState } from "react";

type APIKey = {
  key_id: string;
  name: string;
  role: string;
  created_at: string;
  last_used_at?: string;
};

export default function KeysPage() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [newKeyName, setNewKeyName] = useState("");
  const [generatedKey, setGeneratedKey] = useState("");
  const [generating, setGenerating] = useState(false);

  useEffect(() => {
    fetchKeys();
  }, []);

  const fetchKeys = async () => {
    try {
      const res = await fetch("/api/tenant/keys");
      const data = await res.json();
      if (!data.ok) throw new Error(data.error?.message || "Failed to fetch keys");
      setKeys(data.data || []);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const generateKey = async (e: React.FormEvent) => {
    e.preventDefault();
    setGenerating(true);
    setGeneratedKey("");
    setError("");

    try {
      const res = await fetch("/api/tenant/keys", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: newKeyName || "generated-key" }),
      });
      const data = await res.json();
      if (!data.ok) throw new Error(data.error?.message || "Failed to generate key");
      
      setGeneratedKey(data.data.api_key);
      setNewKeyName("");
      fetchKeys();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setGenerating(false);
    }
  };

  const revokeKey = async (keyId: string) => {
    if (!confirm("Are you sure you want to revoke this key? This action cannot be undone.")) return;
    
    try {
      const res = await fetch(`/api/tenant/keys?key_id=${encodeURIComponent(keyId)}`, {
        method: "DELETE",
      });
      const data = await res.json();
      if (!data.ok) throw new Error(data.error?.message || "Failed to revoke key");
      
      fetchKeys();
    } catch (err: any) {
      setError(err.message);
    }
  };

  return (
    <main className="flex-1 p-6 sm:p-12 max-w-5xl mx-auto w-full">
      <div className="mb-10">
        <h1 className="text-3xl font-mono text-[#e5e5e5] mb-2">
          <span className="text-[#ffb000]">/</span>api_keys
        </h1>
        <p className="text-[#a1a1aa] font-mono text-sm">
          Manage system access tokens and programmatic authentication.
        </p>
      </div>

      {error && (
        <div className="mb-6 text-red-400 text-xs font-mono bg-red-900/20 border border-red-900 p-4 rounded">
          [ERR] {error}
        </div>
      )}

      {/* Generate Key Section */}
      <div className="backdrop-blur-xl bg-[#0a0a0a]/80 border border-[#333333] shadow-lg rounded-md mb-8">
        <div className="bg-[#111111] border-b border-[#333333] p-2 px-4 flex items-center justify-between">
          <span className="text-xs text-[#a1a1aa] font-mono tracking-widest">GENERATE_TOKEN</span>
        </div>
        <div className="p-6">
          <form onSubmit={generateKey} className="flex gap-4 items-end">
            <div className="flex-1">
              <label className="block text-xs text-[#71717a] font-mono mb-1 uppercase tracking-wider">
                Token Name (Optional)
              </label>
              <input
                type="text"
                value={newKeyName}
                onChange={(e) => setNewKeyName(e.target.value)}
                placeholder="e.g., prod-scraper-1"
                className="w-full bg-[#000000] border border-[#333333] focus:border-[#ffb000] rounded px-3 py-2 text-[#e5e5e5] font-mono text-sm outline-none transition-colors"
              />
            </div>
            <button
              type="submit"
              disabled={generating}
              className="bg-[#333] text-[#e5e5e5] hover:bg-[#ffb000] hover:text-black transition-colors font-mono text-sm py-2 px-6 rounded disabled:opacity-50 h-[38px]"
            >
              {generating ? "GENERATING..." : "EXECUTE"}
            </button>
          </form>

          {generatedKey && (
            <div className="mt-6 p-4 bg-green-900/20 border border-green-900 rounded">
              <p className="text-green-400 text-sm font-mono mb-2">
                [SUCCESS] Token generated. Please copy it now. It will not be shown again.
              </p>
              <div className="bg-black p-3 rounded flex items-center justify-between border border-[#333]">
                <code className="text-[#ffb000] font-mono text-sm">{generatedKey}</code>
                <button 
                  onClick={() => navigator.clipboard.writeText(generatedKey)}
                  className="text-xs font-mono text-[#a1a1aa] hover:text-white"
                >
                  [COPY]
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Keys List */}
      <div className="backdrop-blur-xl bg-[#0a0a0a]/80 border border-[#333333] shadow-lg rounded-md">
        <div className="bg-[#111111] border-b border-[#333333] p-2 px-4 flex items-center justify-between">
          <span className="text-xs text-[#a1a1aa] font-mono tracking-widest">ACTIVE_TOKENS</span>
        </div>
        <div className="p-0">
          {loading ? (
            <div className="p-8 text-center text-[#71717a] font-mono text-sm">
              [LOADING_DATA]...
            </div>
          ) : keys.length === 0 ? (
            <div className="p-8 text-center text-[#71717a] font-mono text-sm">
              NO_ACTIVE_TOKENS_FOUND
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left font-mono text-sm">
                <thead>
                  <tr className="border-b border-[#333] bg-[#050505]">
                    <th className="p-4 font-normal text-[#71717a]">NAME</th>
                    <th className="p-4 font-normal text-[#71717a]">ROLE</th>
                    <th className="p-4 font-normal text-[#71717a]">CREATED</th>
                    <th className="p-4 font-normal text-[#71717a]">LAST_USED</th>
                    <th className="p-4 font-normal text-[#71717a] text-right">ACTION</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#222]">
                  {keys.map((key) => (
                    <tr key={key.key_id} className="hover:bg-[#111] transition-colors">
                      <td className="p-4 text-[#e5e5e5]">{key.name}</td>
                      <td className="p-4 text-[#a1a1aa]">{key.role}</td>
                      <td className="p-4 text-[#a1a1aa]">
                        {new Date(key.created_at).toLocaleDateString()}
                      </td>
                      <td className="p-4 text-[#a1a1aa]">
                        {key.last_used_at ? new Date(key.last_used_at).toLocaleDateString() : "NEVER"}
                      </td>
                      <td className="p-4 text-right">
                        <button
                          onClick={() => revokeKey(key.key_id)}
                          className="text-red-400 hover:text-red-300 text-xs tracking-widest"
                        >
                          [REVOKE]
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </main>
  );
}
