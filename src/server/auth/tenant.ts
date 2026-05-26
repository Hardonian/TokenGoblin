import { TenantAuthError } from "../errors";
import type { TenantContext } from "../types";

const TENANT_ID_PATTERN = /^[A-Za-z0-9][A-Za-z0-9_-]{1,79}$/;

export function getTenantContext(headers: Headers): TenantContext {
  const tenantId = headers.get("x-tenant-id")?.trim();

  if (!tenantId) {
    throw new TenantAuthError(
      401,
      "tenant_missing",
      "Missing required x-tenant-id header."
    );
  }

  if (!TENANT_ID_PATTERN.test(tenantId)) {
    throw new TenantAuthError(
      401,
      "tenant_invalid",
      "Invalid tenant context."
    );
  }

  return { tenantId, source: "x-tenant-id" };
}

export function assertPayloadTenantMatchesContext(
  context: TenantContext,
  payloadTenantId?: string
) {
  if (payloadTenantId && payloadTenantId !== context.tenantId) {
    throw new TenantAuthError(
      403,
      "tenant_mismatch",
      "Payload tenantId does not match request tenant context."
    );
  }
}
