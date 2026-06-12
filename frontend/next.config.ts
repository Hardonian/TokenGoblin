import type { NextConfig } from "next";

const apiBase = process.env.NEXT_PUBLIC_TG_API_BASE || "http://127.0.0.1:8080";

const nextConfig: NextConfig = {
  output: "standalone",
  compress: true,
  poweredByHeader: false,
  compiler: {
    removeConsole: process.env.NODE_ENV === "production",
  },
  turbopack: {
    root: process.cwd(),
  },
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${apiBase}/api/:path*`,
      },
      {
        source: "/v1/:path*",
        destination: `${apiBase}/v1/:path*`,
      },
      {
        source: "/v2/:path*",
        destination: `${apiBase}/v2/:path*`,
      },
    ];
  },
  async headers() {
    return [
      {
        source: "/(.*)",
        headers: [
          { key: "X-Frame-Options", value: "DENY" },
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "Strict-Transport-Security", value: "max-age=63072000; includeSubDomains; preload" },
          { key: "Content-Security-Policy", value: "default-src 'self'; script-src 'self' 'unsafe-eval' 'unsafe-inline' https://js.stripe.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; frame-src https://js.stripe.com; connect-src 'self' https://api.stripe.com;" },
        ],
      },
    ];
  },
};

export default nextConfig;
