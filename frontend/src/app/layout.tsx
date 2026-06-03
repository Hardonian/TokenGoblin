import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { Header } from "@/components/Header";

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
    "Enterprise operational intelligence and pricing audit controls for autonomous AI agents.",
  openGraph: {
    title: "TokenGoblin",
    description:
      "Enterprise operational intelligence and pricing audit controls for autonomous AI agents.",
    url: "https://tokengoblin.com",
    siteName: "TokenGoblin",
    type: "website",
  },
};

export const viewport = {
  themeColor: "#0e100d",
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
      <body className="min-h-full flex flex-col">
        <Header />
        {children}
      </body>
    </html>
  );
}
