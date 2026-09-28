import "server-only";

import { cookies } from "next/headers";

export function getServerApiBase() {
  const apiBase = process.env.TG_API_BASE ?? process.env.NEXT_PUBLIC_TG_API_BASE;
  if (!apiBase && process.env.NODE_ENV === "production") {
    throw new Error("TG_API_BASE is required in production");
  }
  return (apiBase ?? "http://localhost:8080").replace(/\/$/, "");
}

export async function getBackendAuthHeaders() {
  const cookieStore = await cookies();
  const apiKey = cookieStore.get("tg_api_key")?.value;
  const tenantID = cookieStore.get("tg_tenant_id")?.value;
  const headers = new Headers({ "content-type": "application/json" });

  if (apiKey) {
    headers.set("authorization", `Bearer ${apiKey}`);
  }
  if (tenantID) {
    headers.set("x-tenant-id", tenantID);
  }
  return { headers, tenantID };
}

export async function readUpstreamJSON(response: Response): Promise<unknown> {
  const contentType = response.headers.get("content-type") ?? "";
  if (!contentType.includes("application/json")) {
    return null;
  }
  try {
    return await response.json();
  } catch {
    return null;
  }
}

export function upstreamStatus(status: number) {
  if ([400, 401, 403, 404, 409, 422, 429].includes(status)) {
    return status;
  }
  return 502;
}
