import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "LLM Spend Audit — Cut Your AI API Bill 30-50% in 72h | TokenGoblin",
  description:
    "Fixed-price LLM and AI agent spend audit. Guaranteed 3x ROI or 100% money back. Ingests OpenAI, Anthropic, and custom logs to deliver actionable savings playbooks.",
  alternates: {
    canonical: "/audit",
  },
  openGraph: {
    title: "Cut Your LLM Spend 30-50% in 72 Hours — TokenGoblin Spend Audit",
    description:
      "Stop leaking budget on over-sized models, runaway loops, and cache misses. Deterministic savings playbooks with 3x ROI guarantee.",
    url: "https://tokengoblin.com/audit",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "TokenGoblin LLM Spend Audit",
    description: "Guaranteed 30-50% LLM cost reduction in 72 hours.",
  },
};

export default function AuditLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <>{children}</>;
}
