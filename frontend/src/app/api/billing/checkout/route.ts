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

    const upstream = await fetch(
      `${getApiBase()}/api/billing/checkout`,
      {
        method: "POST",
        headers: {
          "content-type": "application/json",
        },
        body: JSON.stringify({
          price_id,
          success_url,
          cancel_url,
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
            code: "checkout_failed",
            message: payload?.error?.message || "Checkout failed",
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
