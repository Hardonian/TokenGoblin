import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  experimental: {
    serverExternalPackages: ["node:sqlite"]
  }
};

export default nextConfig;
