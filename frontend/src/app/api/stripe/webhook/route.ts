import crypto from "crypto";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

type StripeWebhookAck = {
  ok: boolean;
  status: "accepted" | "applied" | "ignored" | "not_configured" | "error";
  event_id?: string;
  event_type?: string;
  data?: unknown;
  error?: {
    code: string;
    message: string;
  };
};

type VerifiedStripeEvent = {
  event_id: string;
  event_type: string;
  customer_id?: string;
  subscription_id?: string;
  subscription_status?: string;
  tenant_id?: string;
  metadata?: Record<string, string>;
};

export async function POST(request: Request): Promise<Response> {
  const secret = process.env.STRIPE_WEBHOOK_SECRET;
  const internalSecret = process.env.TG_INTERNAL_WEBHOOK_SECRET;
  const apiBase = (
    process.env.TG_API_BASE ??
    process.env.NEXT_PUBLIC_TG_API_BASE ??
    ""
  ).replace(/\/$/, "");
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
  if (!internalSecret || !apiBase) {
    return json(
      {
        ok: false,
        status: "not_configured",
        error: {
          code: "billing_forwarder_not_configured",
          message:
            "Billing lifecycle forwarding requires TG_INTERNAL_WEBHOOK_SECRET and TG_API_BASE.",
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

  const verifiedEvent = normalizeStripeEvent(event);
  const upstream = await fetch(`${apiBase}/internal/billing/stripe-event`, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      authorization: `Bearer ${internalSecret}`,
    },
    body: JSON.stringify(verifiedEvent),
  });

  let upstreamBody: unknown = null;
  try {
    upstreamBody = await upstream.json();
  } catch {
    upstreamBody = null;
  }

  if (!upstream.ok) {
    return json(
      {
        ok: false,
        status: "error",
        event_id: event.id,
        event_type: event.type,
        data: upstreamBody,
        error: {
          code: "billing_lifecycle_failed",
          message: "Verified Stripe event could not be applied to tenant billing.",
        },
      },
      502,
    );
  }

  const status = extractStatus(upstreamBody);
  return json({
    ok: true,
    status: status === "success" ? "applied" : "ignored",
    event_id: event.id,
    event_type: event.type,
    data: upstreamBody,
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

function normalizeStripeEvent(event: { id?: string; type?: string }): VerifiedStripeEvent {
  const eventRecord = asRecord(event);
  const data = asRecord(eventRecord.data);
  const object = asRecord(data.object);
  const metadata = metadataRecord(object.metadata);
  const eventType = asString(eventRecord.type);

  return {
    event_id: asString(eventRecord.id),
    event_type: eventType,
    customer_id: asString(object.customer),
    subscription_id: subscriptionID(eventType, object),
    subscription_status: asString(object.status),
    tenant_id:
      asString(metadata.tenant_id) ||
      asString(metadata.tenantId) ||
      asString(object.client_reference_id),
    metadata,
  };
}

function subscriptionID(eventType: string, object: Record<string, unknown>): string {
  if (eventType.startsWith("customer.subscription.")) {
    return asString(object.id);
  }
  return asString(object.subscription);
}

function metadataRecord(value: unknown): Record<string, string> {
  const record = asRecord(value);
  return Object.fromEntries(
    Object.entries(record)
      .filter(([, item]) => typeof item === "string")
      .map(([key, item]) => [key, item as string]),
  );
}

function extractStatus(value: unknown): string {
  const record = asRecord(value);
  return asString(record.status);
}

function asRecord(value: unknown): Record<string, unknown> {
  if (typeof value === "object" && value !== null && !Array.isArray(value)) {
    return value as Record<string, unknown>;
  }
  return {};
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : "";
}
