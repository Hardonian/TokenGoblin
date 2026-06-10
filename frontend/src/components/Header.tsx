import Link from "next/link";

const navLinks = [
  { href: "/", label: "Dashboard" },
  { href: "/pricing", label: "Plans" },
  { href: "/billing", label: "Billing" },
  { href: "/intelligence", label: "Intelligence" },
  { href: "/executive", label: "Executive" },
  { href: "/forecasts", label: "Forecasts" },
  { href: "/models", label: "Models" },
  { href: "https://github.com/Hardonian/Settler", label: "Settler", external: true },
  { href: "https://github.com/Hardonian/AIAS", label: "AIAS", external: true },
  { href: "/about", label: "About" },
  { href: "/signup", label: "Start" },
];

export function Header() {
  return (
    <header className="sticky top-0 z-40 border-b border-white/10 bg-[#030303]/70 backdrop-blur-xl">
      <div className="mx-auto flex h-16 w-full max-w-7xl items-center justify-between px-6">
        <Link href="/" className="flex items-center gap-3 transition hover:opacity-90">
          <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-[#00ff9d] text-sm font-black text-black shadow-[0_0_20px_rgba(0,255,157,0.35)]">
            TG
          </span>
          <span className="text-base font-semibold tracking-tight text-white">
            TokenGoblin
          </span>
        </Link>
        <nav className="hidden items-center gap-5 md:flex">
          {navLinks.map((link) =>
            link.external ? (
              <a
                key={link.href}
                href={link.href}
                target="_blank"
                rel="noreferrer"
                className="text-sm text-zinc-300 transition hover:text-white"
              >
                {link.label}
              </a>
            ) : (
              <Link
                key={link.href}
                href={link.href}
                className="text-sm text-zinc-300 transition hover:text-white"
              >
                {link.label}
              </Link>
            )
          )}
        </nav>
      </div>
    </header>
  );
}
