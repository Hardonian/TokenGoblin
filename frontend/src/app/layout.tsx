import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { Header } from "@/components/Header";
import { LeadCaptureWidget } from "@/components/LeadCaptureWidget";
import { SupportChat } from "@/components/SupportChat";
import { Providers } from "@/components/Providers";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "TokenGoblin | AI Spend & Token-Efficiency Observability",
  description:
    "Enterprise operational intelligence and pricing audit controls for autonomous AI agents. Track your agentic LLM spend, discover cost leaks, and find zombie agents.",
  keywords: ["AI Spend", "LLM Costs", "Autonomous Agents", "Token Cost Optimization", "AI Observability", "TokenGoblin"],
  metadataBase: new URL("https://tokengoblin.com"),
  alternates: {
    canonical: "/",
  },
  openGraph: {
    title: "TokenGoblin — Master Your AI Spend",
    description:
      "Enterprise operational intelligence and pricing audit controls for autonomous AI agents. Track your agentic LLM spend, discover cost leaks, and find zombie agents.",
    url: "https://tokengoblin.com",
    siteName: "TokenGoblin",
    type: "website",
    images: [
      {
        url: "/og-image.jpg",
        width: 1200,
        height: 630,
        alt: "TokenGoblin Dashboard Preview",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "TokenGoblin | AI Spend Observability",
    description: "Don't let rogue agents burn your tokens. Uncover cost leaks today.",
    creator: "@TokenGoblin",
  },
};

export const viewport = {
  themeColor: "#000000",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col bg-black text-zinc-100 font-mono">
        <Providers>
          <Header />
          <div className="flex-1">
            {children}
          </div>
          <LeadCaptureWidget />
          <SupportChat />
        </Providers>
      </body>
    </html>
  );
}
