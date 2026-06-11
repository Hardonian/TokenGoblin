import { NextResponse } from "next/server";

export async function POST() {
  const response = NextResponse.json({ ok: true, status: "success" });
  response.cookies.delete("tg_api_key");
  response.cookies.delete("tg_tenant_id");
  return response;
}
