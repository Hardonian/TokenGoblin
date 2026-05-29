ALTER TABLE tenants
ADD COLUMN tier TEXT NOT NULL DEFAULT 'free',
ADD COLUMN usage_limit_usd REAL NOT NULL DEFAULT 10.00,
ADD COLUMN stripe_customer_id TEXT,
ADD COLUMN stripe_subscription_id TEXT;

CREATE TABLE tenant_pricing_overrides (
    override_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(tenant_id),
    provider TEXT NOT NULL,
    model_id TEXT NOT NULL,
    prompt_price_per_million REAL NOT NULL,
    completion_price_per_million REAL NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, provider, model_id)
);
