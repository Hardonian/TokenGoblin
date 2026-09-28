import { NextResponse } from "next/server";

export const dynamic = "force-dynamic";

function getApiBase() {
  return (
    process.env.TG_API_BASE ||
    process.env.NEXT_PUBLIC_TG_API_BASE ||
    "http://localhost:8080"
  ).replace(/\/$/, "");
}

export async function POST(request: Request) {
  try {
    const body = await request.json();
    const { tenant_id, name } = body;

    if (typeof name !== "string" || !name.trim()) {
      return NextResponse.json(
        {
          ok: false,
          status: "error",
          error: {
            code: "invalid_request",
            message: "name is required.",
          },
        },
        { status: 400 }
      );
    }

    const upstream = await fetch(
      `${getApiBase()}/api/tenant/register`,
      {
        method: "POST",
        headers: {
          "content-type": "application/json",
        },
        body: JSON.stringify({
          tenant_id: typeof tenant_id === "string" ? tenant_id.trim() : "",
          name: name.trim(),
        }),
        cache: "no-store",
        signal: AbortSignal.timeout(10_000),
      }
    );

    const payload = await upstream.json();

    if (!upstream.ok || !payload?.ok) {
      return NextResponse.json(
        {
          ok: false,
          status: "error",
          error: {
            code: "registration_failed",
            message: payload?.error?.message || "Registration failed",
          },
        },
        { status: 502 }
      );
    }

    const response = NextResponse.json({
      ok: true,
      status: "success",
      data: payload.data,
    });
    const cookieOptions = {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax" as const,
      maxAge: 60 * 60 * 24 * 30,
      path: "/",
    };
    response.cookies.set("tg_api_key", payload.data.api_key, cookieOptions);
    response.cookies.set("tg_tenant_id", payload.data.tenant_id, cookieOptions);
    return response;
  } catch (error) {
    return NextResponse.json(
      {
        ok: false,
        status: "error",
        error: {
          code: "unexpected_error",
          message: "Registration service is temporarily unavailable.",
        },
      },
      { status: 500 }
    );
  }
}
