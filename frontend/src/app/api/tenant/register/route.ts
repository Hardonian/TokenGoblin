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

    if (!tenant_id || !name) {
      return NextResponse.json(
        {
          ok: false,
          status: "error",
          error: {
            code: "invalid_request",
            message: "tenant_id and name are required.",
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
          tenant_id,
          name,
        }),
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
          message:
            error instanceof Error
              ? error.message
              : "Unexpected error",
        },
      },
      { status: 500 }
    );
  }
}
