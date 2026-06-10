import Link from "next/link";

export function SiteFooter() {
  return (
    <footer className="border-t border-white/10 bg-[#030303]">
      <div className="mx-auto w-full max-w-7xl px-6 py-10">
        <div className="flex flex-col gap-8 md:flex-row md:items-start md:justify-between">
          <div className="space-y-3">
            <Link href="/" className="flex items-center gap-2">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-[#00ff9d] text-xs font-black text-black">
                TG
              </span>
              <span className="text-sm font-semibold text-white">
                TokenGoblin
              </span>
            </Link>
            <p className="max-w-sm text-sm text-zinc-400">
              Deterministic AI spend observability across tasks, models, and
              costs. Built for operators who need evidence, not estimates alone.
            </p>
          </div>
          <div className="grid w-full max-w-2xl grid-cols-2 gap-8 md:grid-cols-3">
            <div className="space-y-2">
              <p className="text-xs font-semibold uppercase tracking-wider text-zinc-400">
                Product
              </p>
              <div className="space-y-1">
                <Link href="/pricing" className="block text-sm text-zinc-300 transition hover:text-white">
                  Plans
                </Link>
                <Link href="/billing" className="block text-sm text-zinc-300 transition hover:text-white">
                  Billing
                </Link>
                <Link href="/signup" className="block text-sm text-zinc-300 transition hover:text-white">
                  Onboard
                </Link>
              </div>
            </div>
            <div className="space-y-2">
              <p className="text-xs font-semibold uppercase tracking-wider text-zinc-400">
                Ecosystem
              </p>
              <div className="space-y-1">
                <a
                  href="https://github.com/Hardonian/Settler"
                  target="_blank"
                  rel="noreferrer"
                  className="block text-sm text-zinc-300 transition hover:text-white"
                >
                  Settler
                </a>
                <a
                  href="https://github.com/Hardonian/AIAS"
                  target="_blank"
                  rel="noreferrer"
                  className="block text-sm text-zinc-300 transition hover:text-white"
                >
                  AIAS
                </a>
                <Link href="/about" className="block text-sm text-zinc-300 transition hover:text-white">
                  About
                </Link>
              </div>
            </div>
            <div className="space-y-2">
              <p className="text-xs font-semibold uppercase tracking-wider text-zinc-400">
                Legal
              </p>
              <div className="space-y-1">
                <Link href="/about" className="block text-sm text-zinc-300 transition hover:text-white">
                  Terms
                </Link>
                <Link href="/about" className="block text-sm text-zinc-300 transition hover:text-white">
                  Privacy
                </Link>
                <a
                  href="mailto:scott@hardonian.com"
                  className="block text-sm text-zinc-300 transition hover:text-white"
                >
                  Contact
                </a>
              </div>
            </div>
          </div>
        </div>
        <div className="mt-10 flex flex-col gap-1 md:flex-row md:items-center md:justify-between">
          <p className="text-xs text-zinc-500">
            © {new Date().getFullYear()} TokenGoblin. Deterministic AI spend observability.
          </p>
          <p className="text-xs text-zinc-600">
            Reconciliation-grade intelligence for Map / Reduce / Replay operations.
          </p>
        </div>
      </div>
    </footer>
  );
}
