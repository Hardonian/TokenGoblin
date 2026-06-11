import { NextResponse } from "next/server";
import { apiBase, Envelope } from "@/lib/api";

export async function POST(request: Request) {
  try {
    const { api_key } = await request.json();

    if (!api_key) {
      return NextResponse.json(
        { ok: false, status: "error", error: { message: "api_key is required" } },
        { status: 400 }
      );
    }

    // Call backend to verify the API key
    const res = await fetch(`${apiBase}/api/tenant/login`, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${api_key}`,
      },
    });

    const data: Envelope<{ tenant_id: string }> = await res.json();

    if (!data.ok || !data.data?.tenant_id) {
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
    response.cookies.set("tg_api_key", api_key, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      maxAge: 60 * 60 * 24 * 30, // 30 days
      path: "/",
    });

    // Also set tenant_id cookie for frontend components to easily know the tenant
    response.cookies.set("tg_tenant_id", data.data.tenant_id, {
      httpOnly: false, // Accessible to JS if needed
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      maxAge: 60 * 60 * 24 * 30,
      path: "/",
    });

    return response;
  } catch (err) {
    return NextResponse.json(
      { ok: false, status: "error", error: { message: (err as Error).message } },
      { status: 500 }
    );
  }
}
