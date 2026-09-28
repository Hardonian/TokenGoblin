type ContactPayload = {
  type?: unknown;
  email?: unknown;
  message?: unknown;
  company?: unknown;
};

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export async function POST(request: Request) {
  let body: ContactPayload;
  try {
    body = (await request.json()) as ContactPayload;
  } catch {
    return Response.json({ ok: false, error: { message: "Invalid JSON body." } }, { status: 400 });
  }

  const type = body.type === "support" ? "support" : "lead";
  const email = typeof body.email === "string" ? body.email.trim().toLowerCase() : "";
  const message = typeof body.message === "string" ? body.message.trim() : "";
  const company = typeof body.company === "string" ? body.company.trim() : "";

  if (company) {
    return Response.json({ ok: true, status: "accepted" });
  }
  if (type === "lead" && !emailPattern.test(email)) {
    return Response.json({ ok: false, error: { message: "Enter a valid email address." } }, { status: 400 });
  }
  if (type === "support" && (message.length < 3 || message.length > 2000)) {
    return Response.json({ ok: false, error: { message: "Message must be between 3 and 2,000 characters." } }, { status: 400 });
  }

  const webhookURL = process.env.TG_CONTACT_WEBHOOK_URL;
  if (!webhookURL) {
    return Response.json(
      { ok: false, error: { message: "Contact delivery is not configured." } },
      { status: 503 },
    );
  }

  const headers = new Headers({ "content-type": "application/json" });
  if (process.env.TG_CONTACT_WEBHOOK_SECRET) {
    headers.set("authorization", `Bearer ${process.env.TG_CONTACT_WEBHOOK_SECRET}`);
  }

  try {
    const upstream = await fetch(webhookURL, {
      method: "POST",
      headers,
      body: JSON.stringify({ type, email: email || undefined, message: message || undefined }),
      cache: "no-store",
      signal: AbortSignal.timeout(8_000),
    });
    if (!upstream.ok) {
      throw new Error("contact webhook rejected request");
    }
  } catch {
    return Response.json(
      { ok: false, error: { message: "Contact delivery is temporarily unavailable." } },
      { status: 502 },
    );
  }

  return Response.json({ ok: true, status: "accepted" }, { status: 202 });
}
