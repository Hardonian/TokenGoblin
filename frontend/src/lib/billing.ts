import { apiBase } from "@/lib/api";

export type PricePlan = {
  id: string;
  label: string;
  priceId?: string;
  amount: number;
};

export const PRICE_PLANS: PricePlan[] = [
  {
    id: "free",
    label: "Free",
    amount: 0,
  },
  {
    id: "pro",
    label: "Pro",
    priceId: process.env.NEXT_PUBLIC_STRIPE_PRICE_PRO,
    amount: 2900,
  },
  {
    id: "enterprise",
    label: "Enterprise",
    priceId: process.env.NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE,
    amount: 9900,
  },
];

export const PLAN_LIMITS_USD: Record<string, number> = {
  free: 0,
  pro: 29,
  enterprise: 99,
};

export const PLAN_MONTHLY_CREDIT: Record<string, number> = {
  free: 50,
  pro: 500,
  enterprise: 2000,
};

export function buildCheckoutSuccessURL(origin: string, plan: string) {
  const base = origin.replace(/\/$/, "");
  return `${base}/billing?plan=${encodeURIComponent(plan)}`;
}

export function buildCheckoutCancelURL(origin: string, plan: string) {
  const base = origin.replace(/\/$/, "");
  return `${base}/pricing?plan=${encodeURIComponent(plan)}`;
}

export async function createCheckoutSession(params: {
  tenantId: string;
  priceId: string;
  successUrl: string;
  cancelUrl: string;
}) {
  const res = await fetch(`${apiBase}/api/billing/checkout`, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      "x-tenant-id": params.tenantId,
    },
    body: JSON.stringify({
      price_id: params.priceId,
      success_url: params.successUrl,
      cancel_url: params.cancelUrl,
    }),
  });

  const payload = await res.json();
  if (!res.ok || !payload?.ok) {
    throw new Error(payload?.error?.message || "Checkout session failed");
  }

  return payload.data;
}

export async function createBillingPortalSession(params: {
  tenantId: string;
  returnUrl: string;
}) {
  const res = await fetch(`${apiBase}/api/billing/portal`, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      "x-tenant-id": params.tenantId,
    },
    body: JSON.stringify({
      return_url: params.returnUrl,
    }),
  });

  const payload = await res.json();
  if (!res.ok || !payload?.ok) {
    throw new Error(payload?.error?.message || "Billing portal failed");
  }

  return payload.data;
}

export async function getBillingStatus(tenantId: string) {
  const res = await fetch(`${apiBase}/api/billing/status`, {
    headers: {
      "x-tenant-id": tenantId,
    },
  });

  const payload = await res.json();
  if (!res.ok || !payload?.ok) {
    throw new Error(payload?.error?.message || "Billing status failed");
  }

  return payload.data;
}

export async function registerTenant(tenantID: string, name: string) {
  const resolvedApiBase = (apiBase || window.location.origin).replace(/\/$/, "");
  const res = await fetch(`${resolvedApiBase}/api/tenant/register`, {
    method: "POST",
    headers: {
      "content-type": "application/json",
    },
    body: JSON.stringify({ tenant_id: tenantID, name }),
  });

  const payload = await res.json();
  if (!res.ok || !payload?.ok) {
    throw new Error(payload?.error?.message || "Registration failed");
  }

  return payload.data;
}
