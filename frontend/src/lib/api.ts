/**
 * API base URL helper.
 */
export const apiBase =
  process.env.NEXT_PUBLIC_TG_API_BASE?.replace(/\/$/, "") ||
  "http://localhost:8080";

/**
 * Standard API envelope shape from the Go backend.
 */
export type Envelope<T> = {
  ok: boolean;
  status: string;
  data?: T;
  degraded?: Issue[];
  error?: Issue;
};

export type Issue = {
  code: string;
  message: string;
  field?: string;
};

/**
 * Fetch wrapper that handles JSON envelopes and tenant header.
 */
export async function apiFetch<T>(
  path: string,
  options?: RequestInit & { tenantID?: string }
): Promise<Envelope<T>> {
  const headers: Record<string, string> = {
    "content-type": "application/json",
    ...(options?.headers as Record<string, string>),
  };
  if (options?.tenantID) {
    headers["x-tenant-id"] = options.tenantID;
  }
  const res = await fetch(`${apiBase}${path}`, { ...options, headers });
  return res.json();
}

/**
 * POST helper
 */
export async function apiPost<T>(
  path: string,
  body: unknown,
  tenantID?: string
): Promise<Envelope<T>> {
  return apiFetch<T>(path, {
    method: "POST",
    body: JSON.stringify(body),
    tenantID,
  });
}
