"use client";

import { useEffect } from "react";

export default function ErrorBoundary({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // Log the error to an error reporting service
    console.error(error);
  }, [error]);

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-[#0e100d] text-white p-4">
      <h2 className="text-2xl font-bold mb-4">Something went wrong!</h2>
      <p className="text-gray-400 mb-6 max-w-md text-center">
        We encountered an unexpected error while rendering this page. The issue has been logged.
      </p>
      <button
        onClick={() => reset()}
        className="px-4 py-2 bg-[#709540] text-[#171915] font-semibold rounded hover:bg-[#85ae50] transition-colors"
      >
        Try again
      </button>
    </div>
  );
}
