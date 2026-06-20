"use client";

import { useState } from "react";
import { authFetcher, useAuth } from "@/lib/auth";

export function SettingsModal({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
  const { apiKey, tenantId } = useAuth();
  const [webhookUrl, setWebhookUrl] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);

  if (!isOpen) return null;

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!apiKey || !tenantId) return;
    
    setLoading(true);
    setError("");
    setSuccess(false);

    try {
      const res = await fetch("/api/tenant/webhook", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${apiKey}`,
          "x-tenant-id": tenantId
        },
        body: JSON.stringify({ webhook_url: webhookUrl })
      });
      const data = await res.json();
      if (!res.ok || !data.ok) throw new Error(data.error?.message || "Failed to update webhook");
      setSuccess(true);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm">
      <div className="w-full max-w-md bg-[#0a0a0a] border border-[#333] shadow-2xl overflow-hidden rounded-md">
        <div className="bg-[#111] border-b border-[#333] p-2 flex items-center justify-between">
          <span className="text-xs text-[#a1a1aa] font-mono tracking-widest">TENANT_CONFIG</span>
          <button onClick={onClose} className="text-[#a1a1aa] hover:text-[#ffb000] font-mono text-xs">
            [X]
          </button>
        </div>

        <div className="p-6">
          <h2 className="text-lg mb-4 font-mono text-[#e5e5e5]">
            <span className="text-[#ffb000]">&gt;</span> ALERTS_WEBHOOK
          </h2>
          <p className="text-xs text-[#a1a1aa] mb-6 font-mono leading-relaxed">
            Configure a webhook URL to receive real-time anomaly alerts from the Goblin Scream detector.
          </p>

          <form onSubmit={handleSave} className="space-y-4">
            <div>
              <label className="block text-xs text-[#71717a] font-mono mb-1 tracking-wider uppercase">
                Webhook URL
              </label>
              <input
                type="url"
                value={webhookUrl}
                onChange={(e) => setWebhookUrl(e.target.value)}
                placeholder="https://hooks.slack.com/services/..."
                className="w-full bg-[#000] border border-[#333] focus:border-[#ffb000] rounded px-3 py-2 text-[#e5e5e5] font-mono text-sm outline-none transition-colors"
                required
              />
            </div>

            {error && (
              <div className="text-red-400 text-xs font-mono bg-red-900/20 border border-red-900 p-2 rounded">
                [ERR] {error}
              </div>
            )}

            {success && (
              <div className="text-green-400 text-xs font-mono bg-green-900/20 border border-green-900 p-2 rounded">
                [OK] Webhook endpoint updated successfully.
              </div>
            )}

            <div className="flex gap-4 pt-4 border-t border-[#333]">
              <button
                type="button"
                onClick={onClose}
                className="flex-1 bg-[#111] text-[#a1a1aa] hover:bg-[#222] hover:text-white transition-colors font-mono text-sm py-2 rounded"
              >
                CANCEL
              </button>
              <button
                type="submit"
                disabled={loading}
                className="flex-1 bg-[#ffb000] text-black hover:bg-[#ff8c00] transition-colors font-mono font-bold text-sm py-2 rounded disabled:opacity-50"
              >
                {loading ? "SAVING..." : "SAVE_CONFIG()"}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
