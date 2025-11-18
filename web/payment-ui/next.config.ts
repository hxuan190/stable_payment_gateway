import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  // Enable WebSocket support
  experimental: {
    serverActions: {
      bodySizeLimit: "2mb",
    },
  },
};

export default nextConfig;
