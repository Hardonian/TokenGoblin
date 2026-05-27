import crypto from "crypto";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

type StripeWebhookAck = {
  ok: boolean;
  status: "accepted" | "not_configured" | "error";
  event_id?: string;
  event_type?: string;
  error?: {
    code: string;
    message: string;
  };
};

export async function POST(request: Request): Promise<Response> {
  const secret = process.env.STRIPE_WEBHOOK_SECRET;
  if (!secret) {
    return json(
      {
        ok: false,
        status: "not_configured",
        error: {
          code: "stripe_not_configured",
          message: "Stripe webhook secret is not configured.",
        },
      },
      503,
    );
  }

  const signature = request.headers.get("stripe-signature") ?? "";
  const rawBody = await request.text();
  if (!verifyStripeSignature(rawBody, signature, secret)) {
    return json(
      {
        ok: false,
        status: "error",
        error: {
          code: "stripe_signature_invalid",
          message: "Stripe webhook signature verification failed.",
        },
      },
      400,
    );
  }

  let event: { id?: string; type?: string };
  try {
    event = JSON.parse(rawBody);
  } catch {
    return json(
      {
        ok: false,
        status: "error",
        error: {
          code: "invalid_json",
          message: "Stripe webhook body was not valid JSON.",
        },
      },
      400,
    );
  }

  return json({
    ok: true,
    status: "accepted",
    event_id: event.id,
    event_type: event.type,
  });
}

function verifyStripeSignature(
  rawBody: string,
  header: string,
  secret: string,
): boolean {
  const parts = Object.fromEntries(
    header.split(",").map((part) => {
      const [key, value] = part.split("=", 2);
      return [key, value];
    }),
  );
  const timestamp = parts.t;
  const v1 = parts.v1;
  if (!timestamp || !v1) {
    return false;
  }
  const issuedAt = Number(timestamp);
  if (!Number.isFinite(issuedAt)) {
    return false;
  }
  const ageSeconds = Math.abs(Date.now() / 1000 - issuedAt);
  if (ageSeconds > 300) {
    return false;
  }

  const payload = `${timestamp}.${rawBody}`;
  const expected = crypto
    .createHmac("sha256", secret)
    .update(payload, "utf8")
    .digest("hex");

  try {
    return crypto.timingSafeEqual(
      Buffer.from(expected, "hex"),
      Buffer.from(v1, "hex"),
    );
  } catch {
    return false;
  }
}

function json(body: StripeWebhookAck, status = 200): Response {
  return Response.json(body, { status });
}
