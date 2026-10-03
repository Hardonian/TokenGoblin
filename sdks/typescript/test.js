const test = require("node:test");
const assert = require("node:assert/strict");
const { TokenGoblinClient } = require("./dist/index.js");

test("TokenGoblinClient throws if apiKey is missing", () => {
  const orig = process.env.TOKEN_GOBLIN_API_KEY;
  delete process.env.TOKEN_GOBLIN_API_KEY;
  try {
    assert.throws(
      () => new TokenGoblinClient({}),
      /API Key must be provided/
    );
  } finally {
    if (orig !== undefined) process.env.TOKEN_GOBLIN_API_KEY = orig;
  }
});

test("TokenGoblinClient throws on non-positive timeout", () => {
  assert.throws(
    () => new TokenGoblinClient({ apiKey: "test-key", timeoutMs: -1 }),
    /timeoutMs must be a positive number/
  );
  assert.throws(
    () => new TokenGoblinClient({ apiKey: "test-key", timeoutMs: 0 }),
    /timeoutMs must be a positive number/
  );
});

test("TokenGoblinClient initializes and strips trailing slashes", () => {
  const client = new TokenGoblinClient({
    apiKey: "test-key",
    baseUrl: "https://api.tokengoblin.com///",
    timeoutMs: 5000,
  });
  assert.equal(client["baseUrl"], "https://api.tokengoblin.com");
  assert.equal(client["timeoutMs"], 5000);
});

test("TokenGoblinClient falls back to env variable", () => {
  process.env.TOKEN_GOBLIN_API_KEY = "env-secret-999";
  try {
    const client = new TokenGoblinClient();
    assert.equal(client["apiKey"], "env-secret-999");
  } finally {
    delete process.env.TOKEN_GOBLIN_API_KEY;
  }
});
