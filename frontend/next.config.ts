import type { NextConfig } from "next";

const apiBase = process.env.NEXT_PUBLIC_TG_API_BASE || "http://127.0.0.1:8080";

const nextConfig: NextConfig = {
  output: "standalone",
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
};

export default nextConfig;
