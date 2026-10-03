import type { MetadataRoute } from "next";

export default function sitemap(): MetadataRoute.Sitemap {
  const baseUrl = "https://tokengoblin.com";
  const now = new Date();

  const routes = [
    { path: "", priority: 1.0, changeFrequency: "daily" as const },
    { path: "/audit", priority: 0.95, changeFrequency: "daily" as const },
    { path: "/pricing", priority: 0.9, changeFrequency: "weekly" as const },
    { path: "/about", priority: 0.8, changeFrequency: "monthly" as const },
    { path: "/signup", priority: 0.85, changeFrequency: "monthly" as const },
    { path: "/login", priority: 0.7, changeFrequency: "monthly" as const },
    { path: "/intelligence", priority: 0.75, changeFrequency: "weekly" as const },
    { path: "/forecasts", priority: 0.75, changeFrequency: "weekly" as const },
    { path: "/executive", priority: 0.75, changeFrequency: "weekly" as const },
    { path: "/models", priority: 0.75, changeFrequency: "weekly" as const },
    { path: "/blog", priority: 0.7, changeFrequency: "weekly" as const },
  ];

  return routes.map((route) => ({
    url: `${baseUrl}${route.path}`,
    lastModified: now,
    changeFrequency: route.changeFrequency,
    priority: route.priority,
  }));
}
