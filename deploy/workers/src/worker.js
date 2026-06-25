/**
 * Cloudflare Workers API for TokenGoblin
 * "Catch the little monsters eating your AI spend."
 * Free tier: 100K req/day
 */

import { Router } from 'itty-router';

const router = Router();

function json(data, status = 200) {
  return new Response(JSON.stringify(data), {
    status, headers: { 'Content-Type': 'application/json' },
  });
}

function error(message, status = 400) {
  return json({ error: message }, status);
}

router.get('/health', () => json({
  status: 'ok', service: 'TokenGoblin', version: '0.1.0',
  timestamp: new Date().toISOString(),
}));

// ─── API Keys ────────────────────────────────────────────────────

router.get('/api/v1/keys', async (request, env) => {
  const result = await env.DB.prepare(
    'SELECT id, name, platform, created_at FROM api_keys ORDER BY created_at DESC LIMIT 50'
  ).all();
  return json({ keys: result.results });
});

router.post('/api/v1/keys', async (request, env) => {
  const body = await request.json();
  const { name, platform, prefix } = body;
  if (!name) return error('name is required');

  const keyHash = 'tg_' + crypto.randomUUID().slice(0, 32);
  const result = await env.DB.prepare(
    `INSERT INTO api_keys (name, key_hash, platform, prefix, status, created_at)
     VALUES (?, ?, ?, ?, 'active', datetime('now'))`
  ).bind(name, keyHash, prefix || 'tg', prefix || 'tg').run();

  return json({ id: result.meta.last_row_id, name, key: keyHash, message: 'Store this key — it won\'t be shown again' });
});

// ─── Usage Tracking ──────────────────────────────────────────────

router.post('/api/v1/usage', async (request, env) => {
  const body = await request.json();
  const { api_key_id, endpoint, model, tokens_input, tokens_output, cost } = body;

  const result = await env.DB.prepare(
    `INSERT INTO usage_events (api_key_id, endpoint, model, tokens_input, tokens_output, cost, created_at)
     VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`
  ).bind(api_key_id, endpoint || 'unknown', model || 'unknown', tokens_input || 0, tokens_output || 0, cost || 0).run();

  return json({ event_id: result.meta.last_row_id, status: 'recorded' });
});

router.get('/api/v1/usage/:api_key_id', async (request, env) => {
  const keyId = parseInt(request.params.api_key_id);
  const result = await env.DB.prepare(
    `SELECT endpoint, model, SUM(tokens_input) as tokens_in, SUM(tokens_output) as tokens_out,
     SUM(cost) as total_cost, COUNT(*) as requests
     FROM usage_events WHERE api_key_id = ?
     GROUP BY endpoint, model ORDER BY total_cost DESC`
  ).bind(keyId).all();
  return json({ usage: result.results });
});

// ─── Spend Alerts ────────────────────────────────────────────────

router.post('/api/v1/alerts', async (request, env) => {
  const body = await request.json();
  const { api_key_id, threshold, channel } = body;
  if (!api_key_id || !threshold) return error('api_key_id and threshold are required');

  const result = await env.DB.prepare(
    `INSERT INTO spend_alerts (api_key_id, threshold, channel, status, created_at)
     VALUES (?, ?, ?, 'active', datetime('now'))`
  ).bind(api_key_id, threshold, channel || 'slack').run();

  return json({ alert_id: result.meta.last_row_id, threshold, channel: channel || 'slack' });
});

// ─── Stats ───────────────────────────────────────────────────────

router.get('/api/v1/stats', async (request, env) => {
  const keys = await env.DB.prepare('SELECT COUNT(*) as c FROM api_keys').first();
  const events = await env.DB.prepare('SELECT COUNT(*) as c FROM usage_events').first();
  const totalCost = await env.DB.prepare('SELECT COALESCE(SUM(cost), 0) as total FROM usage_events').first();
  const totalTokens = await env.DB.prepare('SELECT COALESCE(SUM(tokens_input + tokens_output), 0) as total FROM usage_events').first();

  return json({
    api_keys: keys.count,
    total_events: events.count,
    total_cost_usd: totalCost.total,
    total_tokens: totalTokens.total,
  });
});

// ─── Webhook (internal) ──────────────────────────────────────────

router.post('/api/v1/webhook/stripe', async (request, env) => {
  // Handle Stripe webhooks for billing
  const body = await request.json();
  if (body.type === 'checkout.session.completed') {
    // Provision API key or extend subscription
    return json({ status: 'processed', type: body.type });
  }
  return json({ status: 'ignored', type: body.type });
});

// ─── GitHub Webhook ──────────────────────────────────────────────

router.post('/api/v1/webhook/github', async (request, env) => {
  const event = request.headers.get('x-github-event');
  const payload = await request.json();
  if (event === 'push') return json({ status: 'received', repo: payload.repository?.full_name });
  return json({ status: 'ignored', event });
});

// ─── Cron: Spend monitoring ──────────────────────────────────────

async function handleCron(event, env) {
  // Check spend alerts
  const alerts = await env.DB.prepare(
    'SELECT * FROM spend_alerts WHERE status = 'active''
  ).all();

  let triggered = 0;
  for (const alert of alerts.results) {
    const spend = await env.DB.prepare(
      `SELECT COALESCE(SUM(cost), 0) as total FROM usage_events
       WHERE api_key_id = ? AND created_at >= datetime('now', '-1 day')`
    ).bind(alert.api_key_id).first();

    if (spend.total >= alert.threshold) {
      triggered++;
      await env.DB.prepare(
        "UPDATE spend_alerts SET status = 'triggered' WHERE id = ?"
      ).bind(alert.id).run();
    }
  }

  return json({ alerts_checked: alerts.results.length, triggered });
}

router.all('*', () => error('Not found', 404));

export default {
  async fetch(request, env, ctx) { return router.fetch(request, env, ctx); },
  async scheduled(event, env, ctx) { ctx.waitUntil(handleCron(event, env)); },
};
