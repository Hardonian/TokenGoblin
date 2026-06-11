import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

// Define protected paths that require tenant context
const protectedPaths = ["/billing", "/executive", "/forecasts", "/intelligence", "/models"];

export function middleware(request: NextRequest) {
  // Check if the current path is protected
  const isProtected = protectedPaths.some((path) =>
    request.nextUrl.pathname.startsWith(path)
  ) || request.nextUrl.pathname === "/"; // Dashboard is protected

  // In a real application, we would check for a JWT cookie here.
  // Since TokenGoblin demo relies on a URL parameter or headers for demo tenant injection,
  // we will enforce that the tenant_id is present if it's a protected path, OR just let the
  // components handle the empty state, but we add security headers.

  // Add security headers to all responses
  const headers = new Headers();
  headers.set("X-Content-Type-Options", "nosniff");
  headers.set("X-Frame-Options", "DENY");
  headers.set("X-XSS-Protection", "1; mode=block");
  headers.set("Referrer-Policy", "strict-origin-when-cross-origin");

  // Let the request proceed, injecting headers
  const response = NextResponse.next({
    request: {
      headers: request.headers,
    },
  });

  headers.forEach((value, key) => {
    response.headers.set(key, value);
  });

  return response;
}

export const config = {
  matcher: [
    "/((?!api|_next/static|_next/image|favicon.ico|signup|pricing|about).*)",
  ],
};
