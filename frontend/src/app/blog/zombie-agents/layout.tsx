import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Zombie Agent Detection: How We Saved $12K/Month with Automated Quarantine",
  description: "Deep dive into acceptance-rate-based anomaly detection for AI agents. How TokenGoblin identifies and automatically quarantines zombie agents consuming budget with near-zero value.",
  keywords: [
    "zombie agent detection",
    "AI agent anomaly detection",
    "automated agent quarantine",
    "LLM cost optimization",
    "acceptance rate monitoring",
  ],
  authors: [{ name: "TokenGoblin Team" }],
  openGraph: {
    title: "Zombie Agent Detection: How We Saved $12K/Month",
    description: "Deep dive into acceptance-rate-based anomaly detection for autonomous AI agents.",
    type: "article",
  },
};

export default function Layout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
