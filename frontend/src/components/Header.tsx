import Link from "next/link";

const navLinks = [
  { href: "/", label: "Dashboard" },
  { href: "/pricing", label: "Pricing" },
  { href: "/billing", label: "Billing" },
  { href: "/signup", label: "Sign Up" },
];

export function Header() {
  return (
    <header className="border-b border-[#d7dccf] bg-[#fbfcf8]">
      <div className="mx-auto flex h-14 w-full max-w-7xl items-center justify-between px-5">
        <Link href="/" className="text-lg font-bold tracking-tight text-[#171915]">
          TokenGoblin
        </Link>
        <nav className="flex items-center gap-1">
          {navLinks.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className="rounded-md px-3 py-2 text-sm font-medium text-[#52604e] transition-colors hover:bg-[#e0e4d8] hover:text-[#171915]"
            >
              {link.label}
            </Link>
          ))}
        </nav>
      </div>
    </header>
  );
}
