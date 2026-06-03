"use client";

import { useState } from "react";
import Link from "next/link";

export default function SignupPage() {
  const [tenantID, setTenantID] = useState("");
  const [name, setName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState<{
    tenant_id: string;
    name: string;
    tier: string;
    api_key: string;
    created_at: string;
  } | null>(null);

  const apiBase =
    process.env.NEXT_PUBLIC_TG_API_BASE?.replace(/\/$/, "") ||
    "http://localhost:8080";

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      const res = await fetch(`${apiBase}/api/tenant/register`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ tenant_id: tenantID, name }),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data?.error?.message || "Registration failed");
        return;
      }
      setSuccess(data.data);
    } catch (err) {
      setError("Could not connect to server");
    } finally {
      setLoading(false);
    }
  }

  if (success) {
    return (
      <main className="flex min-h-[80vh] items-center justify-center bg-[#f7f8f3] px-5">
        <div className="w-full max-w-md rounded-2xl border border-[#d7dccf] bg-white p-8 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-[#426b51]/10">
            <svg className="h-6 w-6 text-[#426b51]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <h1 className="text-2xl font-bold">Welcome to TokenGoblin!</h1>
          <p className="mt-2 text-sm text-[#52604e]">
            Your account has been created. Save your API key — it won't be shown again.
          </p>

          <div className="mt-6 rounded-lg border border-amber-200 bg-amber-50 p-4 text-left">
            <p className="text-xs font-semibold uppercase tracking-wide text-amber-700">
              Your API Key
            </p>
            <code className="mt-2 block break-all rounded bg-white px-3 py-2 font-mono text-sm text-[#171915]">
              {success.api_key}
            </code>
          </div>

          <div className="mt-4 rounded-lg border border-[#e0e4d8] bg-[#f7f8f3] p-4 text-left text-sm">
            <p><span className="font-medium">Tenant ID:</span> {success.tenant_id}</p>
            <p className="mt-1"><span className="font-medium">Plan:</span> {success.tier}</p>
          </div>

          <div className="mt-6 flex flex-col gap-3">
            <Link
              href="/"
              className="rounded-lg bg-[#426b51] py-3 text-center text-sm font-semibold text-white hover:bg-[#365a43]"
            >
              Go to Dashboard
            </Link>
            <Link
              href="/pricing"
              className="rounded-lg border border-[#d7dccf] py-3 text-center text-sm font-medium text-[#52604e] hover:bg-[#f7f8f3]"
            >
              View Pricing
            </Link>
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="flex min-h-[80vh] items-center justify-center bg-[#f7f8f3] px-5">
      <div className="w-full max-w-md rounded-2xl border border-[#d7dccf] bg-white p-8">
        <div className="text-center">
          <h1 className="text-2xl font-bold">Create your account</h1>
          <p className="mt-2 text-sm text-[#52604e]">
            Start tracking AI token spend in seconds
          </p>
        </div>

        <form onSubmit={handleSubmit} className="mt-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-[#171915]">
              Tenant ID
            </label>
            <input
              className="mt-1 h-11 w-full rounded-lg border border-[#c5cdbb] bg-white px-3 text-sm outline-none focus:border-[#426b51]"
              placeholder="my-company"
              value={tenantID}
              onChange={(e) =>
                setTenantID(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))
              }
              required
            />
            <p className="mt-1 text-xs text-[#61705a]">
              Lowercase letters, numbers, and hyphens only
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-[#171915]">
              Company / Project Name
            </label>
            <input
              className="mt-1 h-11 w-full rounded-lg border border-[#c5cdbb] bg-white px-3 text-sm outline-none focus:border-[#426b51]"
              placeholder="Acme Corp"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
          </div>

          {error && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="h-11 w-full rounded-lg bg-[#426b51] text-sm font-semibold text-white transition-colors hover:bg-[#365a43] disabled:opacity-50"
          >
            {loading ? "Creating account…" : "Create Account"}
          </button>
        </form>

        <p className="mt-4 text-center text-xs text-[#61705a]">
          By signing up you agree to our Terms of Service and Privacy Policy.
        </p>

        <p className="mt-4 text-center text-sm text-[#52604e]">
          Already have an account?{" "}
          <Link href="/" className="font-medium text-[#426b51] hover:underline">
            Go to Dashboard
          </Link>
        </p>
      </div>
    </main>
  );
}
