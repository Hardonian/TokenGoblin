import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

// Define protected paths that require tenant context
const protectedPaths = ["/billing", "/executive", "/forecasts", "/intelligence", "/models"];

export function middleware(request: NextRequest) {
  // Check if the current path is protected
  const isProtected = protectedPaths.some((path) =>
    request.nextUrl.pathname.startsWith(path)
  ) || request.nextUrl.pathname === "/"; // Dashboard is protected

  const apiKey = request.cookies.get("tg_api_key")?.value;

  if (isProtected && !apiKey) {
    const loginUrl = new URL("/login", request.url);
    return NextResponse.redirect(loginUrl);
  }

  // Add security headers to all responses
  const headers = new Headers();
  headers.set("X-Content-Type-Options", "nosniff");
  headers.set("X-Frame-Options", "DENY");
  headers.set("X-XSS-Protection", "1; mode=block");
  headers.set("Referrer-Policy", "strict-origin-when-cross-origin");

  // Let the request proceed, injecting headers
  const reqHeaders = new Headers(request.headers);
  if (apiKey) {
    reqHeaders.set("Authorization", `Bearer ${apiKey}`);
  }

  const response = NextResponse.next({
    request: {
      headers: reqHeaders,
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
