import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts");

const nextConfig: NextConfig = {
  output: "standalone",
  async rewrites() {
    if (process.env.NODE_ENV !== "development") {
      return [];
    }
    return [
      { source: "/api/public/:path*", destination: "http://localhost:8081/public/:path*" },
      { source: "/api/:path*", destination: "http://localhost:8080/:path*" },
      { source: "/auth/:path*", destination: "http://localhost:8080/auth/:path*" },
    ];
  },
  // Desactivadas temporalmente mientras se valida i18n; reactivar una por una.
  // cacheComponents: true,
  // partialPrefetching: true,
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default withNextIntl(nextConfig);
