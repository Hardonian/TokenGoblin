import { NextResponse } from "next/server";
import {
  getBackendAuthHeaders,
  getServerApiBase,
  readUpstreamJSON,
  upstreamStatus,
} from "@/lib/server-api";

export const dynamic = "force-dynamic";

export async function POST(request: Request) {
  try {
    const body = await request.json();
    const { price_id, success_url, cancel_url } = body;

    if (!price_id || !success_url || !cancel_url) {
      return NextResponse.json(
        {
          ok: false,
          status: "error",
          error: {
            code: "invalid_request",
            message:
              "success_url, cancel_url, and price_id are required.",
          },
        },
        { status: 400 }
      );
    }

    const { headers, tenantID } = await getBackendAuthHeaders();
    if (!tenantID || !headers.has("authorization")) {
      return NextResponse.json(
        { ok: false, status: "error", error: { code: "unauthorized", message: "Sign in to manage billing." } },
        { status: 401 },
      );
    }

    const upstream = await fetch(
      `${getServerApiBase()}/api/billing/checkout`,
      {
        method: "POST",
        headers,
        body: JSON.stringify({
          price_id,
          success_url,
          cancel_url,
        }),
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
            code: "checkout_failed",
            message: payload?.error?.message || "Checkout failed",
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
  } catch {
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
