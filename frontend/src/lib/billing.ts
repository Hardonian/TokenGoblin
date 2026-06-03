import { apiBase } from "@/lib/api";

// ── Billing types ──────────────────────────────────────────────────────────

export type BillingStatus = {
  tenant_id: string;
  tier: string;
  stripe_customer_id?: string;
  current_month_cost_usd: number;
  usage_limit_usd: number;
  usage_percent: number;
  subscription_id?: string;
  needs_upgrade: boolean;
  near_limit: boolean;
  at_limit: boolean;
};

export type Tenant = {
  tenant_id: string;
  name: string;
  tier: string;
  usage_limit_usd: number;
  stripe_customer_id?: string;
  stripe_subscription_id?: string;
  created_at: string;
  updated_at: string;
};

export type RegisterTenantResponse = {
  tenant_id: string;
  name: string;
  tier: string;
  api_key: string;
  created_at: string;
};

export type CheckoutResponse = {
  checkout_url: string;
  session_id: string;
};

export type PortalResponse = {
  portal_url: string;
};

// ── API calls ──────────────────────────────────────────────────────────────

export async function registerTenant(
  tenantID: string,
  name: string
): Promise<RegisterTenantResponse> {
  const res = await fetch(`${apiBase}/api/tenant/register`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ tenant_id: tenantID, name }),
  });
  const data = await res.json();
  if (!data.ok) {
    throw new Error(data.error?.message || "Registration failed");
  }
  return data.data;
}

export async function getBillingStatus(
  tenantID: string
): Promise<BillingStatus> {
  const res = await fetch(`${apiBase}/api/billing/status`, {
    headers: { "x-tenant-id": tenantID },
  });
  const data = await res.json();
  if (!data.ok) {
    throw new Error(data.error?.message || "Failed to fetch billing status");
  }
  return data.data;
}

export async function createCheckoutSession(
  tenantID: string,
  priceID: string,
  successURL: string,
  cancelURL: string
): Promise<CheckoutResponse> {
  const res = await fetch(`${apiBase}/api/billing/checkout`, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      "x-tenant-id": tenantID,
    },
    body: JSON.stringify({
      price_id: priceID,
      success_url: successURL,
      cancel_url: cancelURL,
    }),
  });
  const data = await res.json();
  if (!data.ok) {
    throw new Error(data.error?.message || "Failed to create checkout session");
  }
  return data.data;
}

export async function createPortalSession(
  tenantID: string,
  returnURL: string
): Promise<PortalResponse> {
  const res = await fetch(`${apiBase}/api/billing/portal`, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      "x-tenant-id": tenantID,
    },
    body: JSON.stringify({ return_url: returnURL }),
  });
  const data = await res.json();
  if (!data.ok) {
    throw new Error(data.error?.message || "Failed to create portal session");
  }
  return data.data;
}
