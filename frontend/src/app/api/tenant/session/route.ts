import { cookies } from "next/headers";

export const dynamic = "force-dynamic";

export async function GET() {
  const cookieStore = await cookies();
  const authenticated = Boolean(cookieStore.get("tg_api_key")?.value);
  const tenantID = cookieStore.get("tg_tenant_id")?.value ?? null;

  return Response.json(
    {
      ok: authenticated && Boolean(tenantID),
      status: authenticated && tenantID ? "success" : "anonymous",
      data: { authenticated, tenant_id: tenantID },
    },
    { headers: { "cache-control": "no-store" } },
  );
}
