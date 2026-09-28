import { NextResponse } from "next/server";
import {
  getBackendAuthHeaders,
  getServerApiBase,
  readUpstreamJSON,
  upstreamStatus,
} from "@/lib/server-api";

export const dynamic = "force-dynamic";

export async function GET() {
  const { headers, tenantID } = await getBackendAuthHeaders();
  if (!tenantID || !headers.has("authorization")) {
    return NextResponse.json(
      {
        ok: false,
        status: "error",
        error: {
          code: "unauthorized",
          message: "Sign in to view billing status.",
        },
      },
      { status: 401 }
    );
  }

  try {
    const upstream = await fetch(
      `${getServerApiBase()}/api/billing/status`,
      {
        headers,
        cache: "no-store",
        signal: AbortSignal.timeout(10_000),
      }
    );

    const payload = (await readUpstreamJSON(upstream)) as { ok?: boolean; data?: unknown; error?: { message?: string } } | null;

    if (!upstream.ok || !payload?.ok) {
      return NextResponse.json(
        {
          ok: false,
          status: "error",
          error: {
            code: "status_failed",
            message: payload?.error?.message || "Billing status failed",
          },
        },
        { status: upstreamStatus(upstream.status) }
      );
    }

    return NextResponse.json({
      ok: true,
      status: "success",
      data: payload.data,
    });
  } catch (error) {
    return NextResponse.json(
      {
        ok: false,
        status: "error",
        error: {
          code: "unexpected_error",
          message: "Billing service is temporarily unavailable.",
        },
      },
      { status: 502 }
    );
  }
}
