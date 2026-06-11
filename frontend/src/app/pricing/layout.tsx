import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Pricing - TokenGoblin | Autonomous Spend Control',
  description: 'Transparent pricing for TokenGoblin. Start free and scale your AI operational observability as your fleet grows.',
};

export default function PricingLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
