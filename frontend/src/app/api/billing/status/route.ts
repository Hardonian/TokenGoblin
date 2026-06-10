import { NextResponse } from "next/server";

export const dynamic = "force-dynamic";

function getApiBase() {
  return (
    process.env.TG_API_BASE ||
    process.env.NEXT_PUBLIC_TG_API_BASE ||
    "http://localhost:8080"
  ).replace(/\/$/, "");
}

export async function GET(request: Request) {
  const tenantId = new URL(request.url).searchParams.get("tenant_id");

  if (!tenantId) {
    return NextResponse.json(
      {
        ok: false,
        status: "error",
        error: {
          code: "invalid_request",
          message: "tenant_id query parameter is required.",
        },
      },
      { status: 400 }
    );
  }

  try {
    const upstream = await fetch(
      `${getApiBase()}/api/billing/status?tenant_id=${encodeURIComponent(tenantId)}`,
      {
        headers: {
          "x-tenant-id": tenantId,
        },
      }
    );

    const payload = await upstream.json();

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
