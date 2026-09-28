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
    const { return_url } = body;

    if (!return_url) {
      return NextResponse.json(
        {
          ok: false,
          status: "error",
          error: {
            code: "invalid_request",
            message: "return_url is required.",
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
      `${getServerApiBase()}/api/billing/portal`,
      {
        method: "POST",
        headers,
        body: JSON.stringify({
          return_url,
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
            code: "portal_failed",
            message: payload?.error?.message || "Portal failed",
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
