#!/usr/bin/env bash
# TokenGoblin Production Deployment Script
# Usage: ./scripts/deploy.sh [fly|railway|render]

set -euo pipefail

PLATFORM="${1:-fly}"
PROJECT_NAME="tokengoblin"

echo "🚀 TokenGoblin Deployment to $PLATFORM"
echo "======================================"

check_env() {
    echo "📋 Checking required environment variables..."
    required=(
        "STRIPE_SECRET_KEY"
        "STRIPE_WEBHOOK_SECRET"
        "STRIPE_PRICE_PRO"
        "STRIPE_PRICE_ENTERPRISE"
        "TG_INTERNAL_WEBHOOK_SECRET"
    )
    
    missing=()
    for var in "${required[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            missing+=("$var")
        fi
    done
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        echo "❌ Missing required environment variables:"
        printf '   %s\n' "${missing[@]}"
        echo ""
        echo "Run scripts/setup_stripe_prices.py first to get Stripe price IDs"
        echo "Generate TG_INTERNAL_WEBHOOK_SECRET: openssl rand -hex 32"
        exit 1
    fi
    
    echo "✓ All required env vars present"
}

deploy_fly() {
    echo "📦 Deploying to Fly.io..."
    
    # Check flyctl
    if ! command -v flyctl &> /dev/null; then
        echo "Installing flyctl..."
        curl -L https://fly.io/install.sh | sh
        export PATH="$HOME/.fly/bin:$PATH"
    fi
    
    # Launch or deploy
    if [[ ! -f fly.toml ]]; then
        echo "Initializing Fly.io app..."
        flyctl launch --name "$PROJECT_NAME" --dockerfile Dockerfile.backend --no-deploy
    fi
    
    # Set secrets
    echo "Setting Fly.io secrets..."
    flyctl secrets set \
        STRIPE_SECRET_KEY="$STRIPE_SECRET_KEY" \
        STRIPE_WEBHOOK_SECRET="$STRIPE_WEBHOOK_SECRET" \
        STRIPE_PRICE_PRO="$STRIPE_PRICE_PRO" \
        STRIPE_PRICE_ENTERPRISE="$STRIPE_PRICE_ENTERPRISE" \
        TG_INTERNAL_WEBHOOK_SECRET="$TG_INTERNAL_WEBHOOK_SECRET"
    
    # Deploy
    echo "Deploying..."
    flyctl deploy --dockerfile Dockerfile.backend
    
    # Get URL
    APP_URL=$(flyctl status --json | jq -r '.Hostname')
    echo ""
    echo "✅ Deployed to https://$APP_URL"
    echo ""
    echo "📋 NEXT STEPS:"
    echo "1. Add webhook in Stripe Dashboard: https://$APP_URL/api/v1/webhooks/stripe"
    echo "2. For Vercel frontend, set NEXT_PUBLIC_TG_API_BASE=https://$APP_URL"
    echo "3. Test full flow: signup -> ingest -> billing -> upgrade"
}

deploy_railway() {
    echo "📦 Deploying to Railway..."
    
    if ! command -v railway &> /dev/null; then
        echo "Installing Railway CLI..."
        npm i -g @railway/cli
    fi
    
    # Login if needed
    railway login
    
    # Link or create project
    if [[ ! -f .railway ]]; then
        railway init --name "$PROJECT_NAME"
    fi
    
    # Set variables
    echo "Setting Railway variables..."
    railway variables set \
        STRIPE_SECRET_KEY="$STRIPE_SECRET_KEY" \
        STRIPE_WEBHOOK_SECRET="$STRIPE_WEBHOOK_SECRET" \
        STRIPE_PRICE_PRO="$STRIPE_PRICE_PRO" \
        STRIPE_PRICE_ENTERPRISE="$STRIPE_PRICE_ENTERPRISE" \
        TG_INTERNAL_WEBHOOK_SECRET="$TG_INTERNAL_WEBHOOK_SECRET"
    
    # Deploy
    echo "Deploying..."
    railway up --dockerfile Dockerfile.backend
    
    # Get URL
    APP_URL=$(railway status --json | jq -r '.deployments[0].url')
    echo ""
    echo "✅ Deployed to $APP_URL"
}

# Main
check_env

case "$PLATFORM" in
    fly)
        deploy_fly
        ;;
    railway)
        deploy_railway
        ;;
    *)
        echo "Usage: $0 [fly|railway]"
        echo "  fly      - Deploy to Fly.io (recommended for Go)"
        echo "  railway  - Deploy to Railway"
        exit 1
        ;;
esac

echo ""
echo "🎉 Deployment complete!"
echo ""
echo "📝 Post-deployment checklist:"
echo "   □ Configure Stripe webhook: https://your-app-url/api/v1/webhooks/stripe"
echo "   □ Set Vercel env: NEXT_PUBLIC_TG_API_BASE=https://your-app-url"
echo "   □ Set Vercel env: NEXT_PUBLIC_STRIPE_PRICE_PRO=$STRIPE_PRICE_PRO"
echo "   □ Set Vercel env: NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE=$STRIPE_PRICE_ENTERPRISE"
echo "   □ Test: signup → ingest → dashboard → billing → upgrade"
echo "   □ Monitor: UptimeRobot or similar"