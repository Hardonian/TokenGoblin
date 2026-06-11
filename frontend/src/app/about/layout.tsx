import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'About - Hardonian | TokenGoblin',
  description: 'Learn about the Hardonian product ecosystem: TokenGoblin, Settler, and AIAS.',
};

export default function AboutLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
