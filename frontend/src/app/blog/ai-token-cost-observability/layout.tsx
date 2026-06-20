import { Metadata } from "next";

export const metadata: Metadata = {
  title: "AI Token Cost Observability: Complete Guide to LLM Cost Monitoring",
  description: "Learn how to track, analyze, and optimize AI token spending across your autonomous agent workforce. Complete guide to token cost observability for production LLM deployments.",
  keywords: [
    "AI token cost observability",
    "LLM cost monitoring",
    "token spending tracking",
    "AI agent cost management",
    "prompt compression cost savings",
  ],
  authors: [{ name: "TokenGoblin Team" }],
  openGraph: {
    title: "AI Token Cost Observability: Complete Guide to LLM Cost Monitoring",
    description: "Learn how to track, analyze, and optimize AI token spending across your autonomous agent workforce.",
    type: "article",
  },
};

export default function Layout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
