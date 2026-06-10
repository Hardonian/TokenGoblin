"use client";

import { useState } from "react";
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
  const [open, setOpen] = useState(false);

  return (
    <header className="sticky top-0 z-40 border-b border-white/10 bg-[#030303]/70 backdrop-blur-xl">
      <div className="mx-auto flex h-16 w-full max-w-7xl items-center justify-between px-4 sm:px-6">
        <Link href="/" className="flex items-center gap-3 transition hover:opacity-90">
          <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-[#00ff9d] text-sm font-black text-black shadow-[0_0_20px_rgba(0,255,157,0.35)]">
            TG
          </span>
          <span className="text-base font-semibold tracking-tight text-white">
            TokenGoblin
          </span>
        </Link>

        <button
          type="button"
          aria-expanded={open}
          aria-controls="primary-navigation"
          onClick={() => setOpen((prev) => !prev)}
          className="inline-flex items-center justify-center rounded-md p-2 text-zinc-300 transition hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-[#00ff9d] md:hidden"
        >
          <span className="sr-only">Toggle navigation</span>
          <svg aria-hidden="true" className="h-6 w-6" fill="none" stroke="currentColor" strokeWidth="1.5" viewBox="0 0 24 24">
            {open ? (
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            ) : (
              <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 12h16M4 18h16" />
            )}
          </svg>
        </button>

        <nav id="primary-navigation" className="hidden md:flex md:items-center md:gap-5">
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

      <div className={`${open ? "block" : "hidden"} md:hidden border-t border-white/10 bg-[#050505]`}>
        <nav className="flex flex-col px-4 py-3">
          {navLinks.map((link) =>
            link.external ? (
              <a
                key={link.href}
                href={link.href}
                target="_blank"
                rel="noreferrer"
                className="rounded-md px-3 py-2 text-sm text-zinc-300 transition hover:bg-white/5 hover:text-white"
                onClick={() => setOpen(false)}
              >
                {link.label}
              </a>
            ) : (
              <Link
                key={link.href}
                href={link.href}
                className="rounded-md px-3 py-2 text-sm text-zinc-300 transition hover:bg-white/5 hover:text-white"
                onClick={() => setOpen(false)}
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
