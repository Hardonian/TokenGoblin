"use client";

import { useEffect } from "react";
import { GoblinMascot } from "@/components/GoblinMascot";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // Log the error to an error reporting service in production
    console.error("Global boundary caught an error:", error);
  }, [error]);

  return (
    <div className="min-h-screen bg-black flex flex-col items-center justify-center font-mono text-zinc-300 p-6 text-center">
      <div className="mb-8">
        <GoblinMascot size={80} />
      </div>
      <h1 className="text-4xl font-black text-red-500 mb-4 tracking-widest uppercase">
        [SYS_FAILURE]
      </h1>
      <p className="text-zinc-400 max-w-lg mb-8 leading-relaxed">
        A rogue agent just crashed the rendering pipeline. The TokenGoblins have been dispatched to investigate the cost leak.
      </p>
      
      <div className="flex gap-4">
        <button
          onClick={() => reset()}
          className="bg-black hover:bg-[#111] border border-var(--color-accent-goblin) text-var(--color-accent-goblin) px-6 py-3 font-bold uppercase tracking-widest transition-colors"
        >
          [ Retry Render ]
        </button>
        <button
          onClick={() => window.location.href = '/'}
          className="bg-[#ffb000] hover:bg-[#ff8c00] text-black px-6 py-3 font-bold uppercase tracking-widest transition-colors"
        >
          [ Reboot System ]
        </button>
      </div>
      
      <div className="mt-12 text-[10px] text-zinc-600 uppercase tracking-widest">
        Error Digest: {error.digest || "UNKNOWN_SIG"}
      </div>
    </div>
  );
}
