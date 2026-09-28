import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const protectedPaths = [
  "/",
  "/billing",
  "/executive",
  "/forecasts",
  "/intelligence",
  "/keys",
  "/models",
  "/pricing/overrides",
  "/refiner",
  "/scholar",
];

function isProtectedPath(pathname: string) {
  return protectedPaths.some(
    (path) => pathname === path || (path !== "/" && pathname.startsWith(`${path}/`)),
  );
}

export function proxy(request: NextRequest) {
  const apiKey = request.cookies.get("tg_api_key")?.value;

  if (isProtectedPath(request.nextUrl.pathname) && !apiKey) {
    const loginURL = new URL("/login", request.url);
    loginURL.searchParams.set("next", request.nextUrl.pathname);
    return NextResponse.redirect(loginURL);
  }

  const requestHeaders = new Headers(request.headers);
  if (apiKey) {
    requestHeaders.set("authorization", `Bearer ${apiKey}`);
  }

  return NextResponse.next({ request: { headers: requestHeaders } });
}

export const config = {
  matcher: [
    "/((?!_next/static|_next/image|favicon.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp)$).*)",
  ],
};
