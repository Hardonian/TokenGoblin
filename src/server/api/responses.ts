import { NextResponse } from "next/server";
import { DatabaseUnavailableError, TenantAuthError, ValidationError } from "../errors";
import type { ApiIssue, ApiStatus } from "../types";

interface ApiEnvelope<T> {
  ok: boolean;
  status: ApiStatus;
  data?: T;
  degraded?: ApiIssue[];
  warnings?: ApiIssue[];
  error?: ApiIssue;
}

export function apiResponse<T>(
  body: ApiEnvelope<T>,
  init?: ResponseInit
): NextResponse<ApiEnvelope<T>> {
  return NextResponse.json(body, init);
}

export function routeErrorResponse(error: unknown): NextResponse {
  if (error instanceof ValidationError) {
    return apiResponse(
      {
        ok: false,
        status: "error",
        error: { code: "validation_error", message: error.message },
        degraded: error.issues
      },
      { status: error.statusCode }
    );
  }

  if (error instanceof TenantAuthError) {
    return apiResponse(
      {
        ok: false,
        status: "error",
        error: { code: error.code, message: error.message }
      },
      { status: error.statusCode }
    );
  }

  if (error instanceof DatabaseUnavailableError) {
    return apiResponse(
      {
        ok: false,
        status: "degraded",
        degraded: [
          {
            code: error.code,
            message: "Storage is unavailable; request could not be persisted."
          }
        ],
        error: {
          code: error.code,
          message: "Storage is unavailable; request could not be persisted."
        }
      },
      { status: 503 }
    );
  }

  console.error("Unhandled route error", {
    name: error instanceof Error ? error.name : "unknown"
  });

  return apiResponse(
    {
      ok: false,
      status: "degraded",
      error: {
        code: "service_unavailable",
        message: "Request could not be completed."
      }
    },
    { status: 503 }
  );
}

export function dashboardUnavailableResponse<T>(data: T, issue: ApiIssue) {
  return apiResponse(
    {
      ok: true,
      status: "degraded",
      data,
      degraded: [issue]
    },
    { status: 200 }
  );
}
