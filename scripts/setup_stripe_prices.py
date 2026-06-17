#!/usr/bin/env python3
"""
Stripe Price Setup Script for TokenGoblin

Run this script to create live Stripe products and prices for TokenGoblin.
Outputs the price IDs to add to your environment variables.

Prerequisites:
- Stripe CLI installed and authenticated (stripe login)
- OR Stripe secret key in STRIPE_SECRET_KEY env var

Usage:
  export STRIPE_SECRET_KEY=sk_live_xxx
  python3 scripts/setup_stripe_prices.py
"""

import os
import sys
import json
import subprocess
import time

def run_stripe_cli(args, secret_key=None):
    """Run stripe CLI command."""
    env = os.environ.copy()
    if secret_key:
        env['STRIPE_SECRET_KEY'] = secret_key
    
    result = subprocess.run(
        ['stripe'] + args,
        capture_output=True,
        text=True,
        env=env
    )
    if result.returncode != 0:
        print(f"ERROR: stripe {' '.join(args)}")
        print(result.stderr)
        return None
    return result.stdout

def create_product(name, description, secret_key=None):
    """Create a Stripe product."""
    output = run_stripe_cli([
        'products', 'create',
        '--name', name,
        '--description', description,
        '--metadata[tokengoblin]', 'true'
    ], secret_key)
    
    if output:
        # Parse JSON output
        try:
            return json.loads(output.strip().split('\n')[-1])
        except:
            pass
    return None

def create_price(product_id, amount_cents, currency, interval, nickname, secret_key=None):
    """Create a Stripe price."""
    output = run_stripe_cli([
        'prices', 'create',
        '--product', product_id,
        '--unit-amount', str(amount_cents),
        '--currency', currency,
        '--recurring[interval]', interval,
        '--nickname', nickname,
        '--metadata[tokengoblin]', 'true'
    ], secret_key)
    
    if output:
        try:
            return json.loads(output.strip().split('\n')[-1])
        except:
            pass
    return None

def create_webhook_endpoint(url, secret_key=None):
    """Create a webhook endpoint."""
    output = run_stripe_cli([
        'webhook_endpoints', 'create',
        '--url', url,
        '--enabled-events[]', 'customer.subscription.created',
        '--enabled-events[]', 'customer.subscription.updated',
        '--enabled-events[]', 'customer.subscription.deleted',
        '--enabled-events[]', 'checkout.session.completed',
        '--description', 'TokenGoblin production webhook',
    ], secret_key)
    
    if output:
        try:
            return json.loads(output.strip().split('\n')[-1])
        except:
            pass
    return None

def main():
    secret_key = os.getenv('STRIPE_SECRET_KEY')
    
    if not secret_key:
        print("ERROR: STRIPE_SECRET_KEY environment variable not set")
        print("Run: export STRIPE_SECRET_KEY=sk_live_xxx")
        sys.exit(1)
    
    if not secret_key.startswith('sk_live_'):
        print("WARNING: Using test key (sk_test_). For production, use sk_live_ key.")
    
    print("=" * 60)
    print("TokenGoblin Stripe Price Setup")
    print("=" * 60)
    
    # Create Pro product
    print("\n[1/4] Creating Pro product...")
    pro_product = create_product(
        "TokenGoblin Pro",
        "Forecasting and cost-leak analysis for active AI fleets. 5 tenants, 100K events/mo, spend forecasting, cost leak detection, priority support.",
        secret_key
    )
    if not pro_product:
        print("Failed to create Pro product")
        sys.exit(1)
    pro_product_id = pro_product['id']
    print(f"  ✓ Product created: {pro_product_id}")
    
    # Create Pro price ($29/mo = 2900 cents)
    print("\n[2/4] Creating Pro price ($29/mo)...")
    pro_price = create_price(
        pro_product_id, 2900, 'usd', 'month', 'Pro Monthly',
        secret_key
    )
    if not pro_price:
        print("Failed to create Pro price")
        sys.exit(1)
    pro_price_id = pro_price['id']
    print(f"  ✓ Price created: {pro_price_id}")
    
    # Create Enterprise product
    print("\n[3/4] Creating Enterprise product...")
    ent_product = create_product(
        "TokenGoblin Enterprise",
        "Unlimited observability for platform teams. Unlimited tenants, unlimited events, custom pricing overrides, audit trail, RBAC, SLA, dedicated support.",
        secret_key
    )
    if not ent_product:
        print("Failed to create Enterprise product")
        sys.exit(1)
    ent_product_id = ent_product['id']
    print(f"  ✓ Product created: {ent_product_id}")
    
    # Create Enterprise price ($99/mo = 9900 cents)
    print("\n[4/4] Creating Enterprise price ($99/mo)...")
    ent_price = create_price(
        ent_product_id, 9900, 'usd', 'month', 'Enterprise Monthly',
        secret_key
    )
    if not ent_price:
        print("Failed to create Enterprise price")
        sys.exit(1)
    ent_price_id = ent_price['id']
    print(f"  ✓ Price created: {ent_price_id}")
    
    # Optional: Create webhook endpoint
    print("\n[Optional] Webhook endpoint...")
    webhook_url = input("Enter production webhook URL (e.g., https://api.tokengoblin.com/api/stripe/webhook) or press Enter to skip: ").strip()
    webhook_secret = None
    if webhook_url:
        wh = create_webhook_endpoint(webhook_url, secret_key)
        if wh:
            webhook_secret = wh.get('secret')
            print(f"  ✓ Webhook created: {wh['id']}")
            print(f"  ✓ Webhook secret: {webhook_secret}")
    
    # Summary
    print("\n" + "=" * 60)
    print("SETUP COMPLETE - ADD TO YOUR ENVIRONMENT")
    print("=" * 60)
    
    print(f"""
# Backend (Go server / Docker / Fly.io / Railway)
STRIPE_PRICE_PRO={pro_price_id}
STRIPE_PRICE_ENTERPRISE={ent_price_id}
STRIPE_SECRET_KEY={secret_key}
STRIPE_WEBHOOK_SECRET={webhook_secret or 'whsec_xxx (from Stripe dashboard)'}
TG_INTERNAL_WEBHOOK_SECRET=$(openssl rand -hex 32)

# Frontend (Vercel)
NEXT_PUBLIC_STRIPE_PRICE_PRO={pro_price_id}
NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE={ent_price_id}
NEXT_PUBLIC_TG_API_BASE=https://api.yourdomain.com
""")
    
    # Save to file for reference
    with open('stripe_price_ids.txt', 'w') as f:
        f.write(f"""TokenGoblin Stripe Price IDs
Generated: {time.strftime('%Y-%m-%d %H:%M:%S')}

Pro Product: {pro_product_id}
Pro Price: {pro_price_id} ($29/mo)

Enterprise Product: {ent_product_id}
Enterprise Price: {ent_price_id} ($99/mo)

Webhook Secret: {webhook_secret or 'N/A'}
""")
    
    print("\n✓ Saved to stripe_price_ids.txt")

if __name__ == '__main__':
    main()