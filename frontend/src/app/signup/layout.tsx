import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Sign Up - TokenGoblin | Initialize Workspace',
  description: 'Initialize your TokenGoblin workspace and start tracking your AI fleet metrics.',
};

export default function SignupLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
