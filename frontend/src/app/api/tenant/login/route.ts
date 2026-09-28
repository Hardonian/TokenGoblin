import { NextResponse } from "next/server";
import { getServerApiBase, readUpstreamJSON } from "@/lib/server-api";
import type { Envelope } from "@/lib/api";

export async function POST(request: Request) {
  try {
    const { api_key } = await request.json();

    if (typeof api_key !== "string" || !api_key.trim()) {
      return NextResponse.json(
        { ok: false, status: "error", error: { message: "api_key is required" } },
        { status: 400 }
      );
    }

    // Call backend to verify the API key
    const normalizedKey = api_key.trim();
    const res = await fetch(`${getServerApiBase()}/api/tenant/login`, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${normalizedKey}`,
      },
      cache: "no-store",
      signal: AbortSignal.timeout(10_000),
    });

    const data = (await readUpstreamJSON(res)) as Envelope<{ tenant_id: string }> | null;

    if (!res.ok || !data?.ok || !data.data?.tenant_id) {
      return NextResponse.json(
        { ok: false, status: "error", error: { message: "Invalid API key" } },
        { status: 401 }
      );
    }

    const response = NextResponse.json({
      ok: true,
      status: "success",
      data: { tenant_id: data.data.tenant_id },
    });

    // Set cookie for Next.js session
    response.cookies.set("tg_api_key", normalizedKey, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      maxAge: 60 * 60 * 24 * 30, // 30 days
      path: "/",
    });

    // Also set tenant_id cookie for frontend components to easily know the tenant
    response.cookies.set("tg_tenant_id", data.data.tenant_id, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      maxAge: 60 * 60 * 24 * 30,
      path: "/",
    });

    return response;
  } catch {
    return NextResponse.json(
      {
        ok: false,
        status: "error",
        error: { message: "Authentication service is temporarily unavailable" },
      },
      { status: 502 }
    );
  }
}
