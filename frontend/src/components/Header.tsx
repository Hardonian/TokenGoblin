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
    <header className="sticky top-0 z-40 border-b border-[#333] bg-black">
      <div className="mx-auto flex h-14 w-full items-center justify-between px-4 sm:px-6">
        <Link href="/" className="flex items-center gap-2 group transition-colors">
          <span className="text-[#ffb000] font-black text-lg group-hover:text-[#ff8c00]">
            root@tg:~#
          </span>
          <span className="text-sm tracking-widest text-zinc-300 group-hover:text-white uppercase">
            TokenGoblin
          </span>
        </Link>

        <button
          type="button"
          aria-expanded={open}
          aria-controls="primary-navigation"
          onClick={() => setOpen((prev) => !prev)}
          className="inline-flex items-center justify-center rounded-none p-2 text-[#ffb000] border border-transparent hover:border-[#ffb000] transition-colors focus:outline-none md:hidden"
        >
          <span className="sr-only">Toggle navigation</span>
          <span className="font-bold">{open ? "[X]" : "[=]"}</span>
        </button>

        <nav id="primary-navigation" className="hidden md:flex md:items-center md:gap-4">
          {navLinks.map((link) =>
            link.external ? (
              <a
                key={link.href}
                href={link.href}
                target="_blank"
                rel="noreferrer"
                className="text-xs text-zinc-500 hover:text-[#ffb000] transition-colors tracking-wider"
              >
                [{link.label.toUpperCase()}]
              </a>
            ) : (
              <Link
                key={link.href}
                href={link.href}
                className="text-xs text-zinc-500 hover:text-[#ffb000] transition-colors tracking-wider"
              >
                [{link.label.toUpperCase()}]
              </Link>
            )
          )}
        </nav>
      </div>

      <div className={`${open ? "block" : "hidden"} md:hidden border-t border-[#333] bg-black`}>
        <nav className="flex flex-col px-4 py-3">
          {navLinks.map((link) =>
            link.external ? (
              <a
                key={link.href}
                href={link.href}
                target="_blank"
                rel="noreferrer"
                className="px-2 py-2 text-sm text-zinc-400 hover:text-[#ffb000] transition-colors uppercase tracking-widest"
                onClick={() => setOpen(false)}
              >
                &gt; {link.label}
              </a>
            ) : (
              <Link
                key={link.href}
                href={link.href}
                className="px-2 py-2 text-sm text-zinc-400 hover:text-[#ffb000] transition-colors uppercase tracking-widest"
                onClick={() => setOpen(false)}
              >
                &gt; {link.label}
              </Link>
            )
          )}
        </nav>
      </div>
    </header>
  );
}
