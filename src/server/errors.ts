import type { ApiIssue } from "./types";

export class ValidationError extends Error {
  readonly statusCode = 400;
  readonly issues: ApiIssue[];

  constructor(message: string, issues: ApiIssue[]) {
    super(message);
    this.name = "ValidationError";
    this.issues = issues;
  }
}

export class TenantAuthError extends Error {
  readonly statusCode: 401 | 403;
  readonly code: string;

  constructor(statusCode: 401 | 403, code: string, message: string) {
    super(message);
    this.name = "TenantAuthError";
    this.statusCode = statusCode;
    this.code = code;
  }
}

export class DatabaseUnavailableError extends Error {
  readonly code = "database_unavailable";

  constructor(message = "Database is unavailable.") {
    super(message);
    this.name = "DatabaseUnavailableError";
  }
}
