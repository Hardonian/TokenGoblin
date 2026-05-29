DROP TABLE tenant_pricing_overrides;

ALTER TABLE tenants
DROP COLUMN stripe_subscription_id,
DROP COLUMN stripe_customer_id,
DROP COLUMN usage_limit_usd,
DROP COLUMN tier;
