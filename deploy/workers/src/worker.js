/**
 * Cloudflare Worker for TokenGoblin API.
 * Dependency-free router for LLM token/cost telemetry and spend alerts.
 */

const SERVICE = 'tokengoblin-api';
const VERSION = '0.1.0';
const CORS = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Methods': 'GET,POST,OPTIONS',
  'Access-Control-Allow-Headers': 'Content-Type, Authorization',
};

function json(data, status = 200) {
  return new Response(JSON.stringify(data), { status, headers: { 'Content-Type': 'application/json', ...CORS } });
}
function error(message, status = 400) { return json({ error: message }, status); }
async function parseJSON(request) { try { return await request.json(); } catch { return {}; } }

async function ensureSchema(env) {
  await env.DB.prepare(`CREATE TABLE IF NOT EXISTS usage_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    api_key_id TEXT,
    model TEXT,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cost REAL NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
  )`).run();
  await env.DB.prepare(`CREATE TABLE IF NOT EXISTS spend_alerts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    api_key_id TEXT,
    threshold REAL NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
  )`).run();
}

async function createUsage(request, env) {
  await ensureSchema(env);
  const body = await parseJSON(request);
  const input = Number(body.input_tokens || 0);
  const output = Number(body.output_tokens || 0);
  const cost = Number(body.cost || 0);
  await env.DB.prepare(
    `INSERT INTO usage_events (api_key_id, model, input_tokens, output_tokens, cost, created_at)
     VALUES (?, ?, ?, ?, ?, datetime('now'))`
  ).bind(body.api_key_id || null, body.model || 'unknown', input, output, cost).run();
  const latest = await env.DB.prepare('SELECT * FROM usage_events ORDER BY id DESC LIMIT 1').first();
  return json({ service: SERVICE, usage: latest }, 201);
}

async function usageSummary(env) {
  await ensureSchema(env);
  const row = await env.DB.prepare(
    `SELECT COUNT(*) AS events, COALESCE(SUM(input_tokens),0) AS input_tokens,
            COALESCE(SUM(output_tokens),0) AS output_tokens, COALESCE(SUM(cost),0) AS cost
     FROM usage_events`
  ).first();
  return json({ service: SERVICE, summary: row });
}

async function listAlerts(env) {
  await ensureSchema(env);
  const result = await env.DB.prepare('SELECT * FROM spend_alerts ORDER BY created_at DESC LIMIT 50').all();
  return json({ service: SERVICE, alerts: result.results || [] });
}

async function handleCron(event, env) {
  await ensureSchema(env);
  const alerts = await env.DB.prepare("SELECT * FROM spend_alerts WHERE status = 'active'").all();
  let triggered = 0;
  for (const alert of alerts.results || []) {
    const spend = await env.DB.prepare(
      `SELECT COALESCE(SUM(cost), 0) as total FROM usage_events
       WHERE api_key_id = ? AND created_at >= datetime('now', '-1 day')`
    ).bind(alert.api_key_id).first();
    if (Number(spend.total) >= Number(alert.threshold)) {
      triggered++;
      await env.DB.prepare("UPDATE spend_alerts SET status = 'triggered' WHERE id = ?").bind(alert.id).run();
    }
  }
  return { alerts_checked: (alerts.results || []).length, triggered };
}

async function route(request, env) {
  const url = new URL(request.url);
  if (request.method === 'OPTIONS') return new Response(null, { headers: CORS });
  if (url.pathname === '/' || url.pathname === '/health' || url.pathname === '/api/v1/health') {
    return json({ status: 'ok', service: SERVICE, version: VERSION, timestamp: new Date().toISOString() });
  }
  if (url.pathname === '/api/v1/usage' && request.method === 'POST') return createUsage(request, env);
  if (url.pathname === '/api/v1/usage' && request.method === 'GET') return usageSummary(env);
  if (url.pathname === '/api/v1/alerts' && request.method === 'GET') return listAlerts(env);
  return error('Not found', 404);
}

export default {
  async fetch(request, env, ctx) { return route(request, env); },
  async scheduled(event, env, ctx) { ctx.waitUntil(handleCron(event, env)); },
};
