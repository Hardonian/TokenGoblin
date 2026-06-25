CREATE TABLE IF NOT EXISTS api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    platform TEXT DEFAULT 'tg',
    prefix TEXT DEFAULT 'tg',
    status TEXT DEFAULT 'active',
    user_email TEXT,
    plan TEXT DEFAULT 'free',
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS usage_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    api_key_id INTEGER NOT NULL,
    endpoint TEXT NOT NULL,
    model TEXT NOT NULL,
    tokens_input INTEGER DEFAULT 0,
    tokens_output INTEGER DEFAULT 0,
    cost REAL DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS spend_alerts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    api_key_id INTEGER NOT NULL,
    threshold REAL NOT NULL,
    channel TEXT DEFAULT 'slack',
    status TEXT DEFAULT 'active',
    triggered_at TEXT,
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_usage_key ON usage_events(api_key_id);
CREATE INDEX IF NOT EXISTS idx_usage_created ON usage_events(created_at);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON spend_alerts(status);
