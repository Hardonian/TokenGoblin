import Link from "next/link";

export function SiteHeader({ transparent = false }: { transparent?: boolean }) {
  return (
    <header
      className={`w-full border-b ${
        transparent
          ? "bg-transparent border-transparent absolute top-0 left-0 z-10"
          : "bg-[#fbfcf8] border-[#d7dccf]"
      }`}
    >
      <div className="mx-auto flex w-full max-w-6xl items-center justify-between px-6 py-4">
        <Link href="/" className="flex items-center gap-2.5 group">
          <div className="w-8 h-8 bg-[#426b51] rounded flex items-center justify-center">
            <span className="text-white text-sm font-bold">TG</span>
          </div>
          <span className="text-lg font-semibold tracking-tight group-hover:text-[#426b51] transition-colors">
            TokenGoblin
          </span>
        </Link>
        <nav className="flex items-center gap-6">
          <Link
            href="/pricing"
            className="text-sm text-[#52604e] hover:text-[#171915] transition-colors"
          >
            Pricing
          </Link>
          <Link
            href="/signup"
            className="text-sm font-semibold text-[#426b51] hover:text-[#2d6a4f] transition-colors"
          >
            Sign Up
          </Link>
        </nav>
      </div>
    </header>
  );
}

export function SiteFooter() {
  return (
    <footer className="border-t border-[#d7dccf] bg-[#fbfcf8]">
      <div className="mx-auto w-full max-w-6xl px-6 py-8">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-2">
            <div className="w-6 h-6 bg-[#426b51] rounded flex items-center justify-center">
              <span className="text-white text-[10px] font-bold">TG</span>
            </div>
            <span className="text-sm font-semibold">TokenGoblin</span>
          </div>
          <div className="flex items-center gap-6 text-sm text-[#61705a]">
            <Link
              href="/pricing"
              className="hover:text-[#171915] transition-colors"
            >
              Pricing
            </Link>
            <Link
              href="/signup"
              className="hover:text-[#171915] transition-colors"
            >
              Sign Up
            </Link>
            <a
              href="https://github.com/Hardonian/TokenGoblin"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-[#171915] transition-colors"
            >
              GitHub
            </a>
            <a
              href="mailto:scott@hardonian.com"
              className="hover:text-[#171915] transition-colors"
            >
              Contact
            </a>
          </div>
        </div>
        <p className="mt-4 text-xs text-[#61705a]">
          © {new Date().getFullYear()} TokenGoblin. Deterministic AI spend
          observability. All rights reserved.
        </p>
      </div>
    </footer>
  );
}
