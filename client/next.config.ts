import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // `next build` runs ESLint and fails on the existing backlog (unused vars,
  // empty interfaces), none of which affects runtime. Lint stays a separate
  // `npm run lint` step; the build still type-checks.
  eslint: {
    ignoreDuringBuilds: true,
  },
};

export default nextConfig;
