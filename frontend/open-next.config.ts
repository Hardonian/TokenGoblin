import { defineCloudflareConfig } from "@opennextjs/cloudflare";

// OpenNext config: turns this Next.js app (SSR + /api/* rewrites) into a
// Cloudflare Worker + static assets so it runs at the global edge (Pages/Workers),
// replacing the single-region Vercel deploy for lower latency + cost.
//
// The /api/* rewrites in next.config.ts proxy to the TokenGoblin Workers API
// (tokengoblin-api, deploy/workers/wrangler.toml) — set NEXT_PUBLIC_TG_API_BASE
// to that Workers URL in the Cloudflare env.
export default defineCloudflareConfig();
