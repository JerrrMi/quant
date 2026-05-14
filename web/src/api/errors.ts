export class ApiError extends Error {
  readonly status: number;
  readonly body: unknown;

  constructor(message: string, status: number, body?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

function flattenFieldMessages(issues: unknown): string[] {
  if (typeof issues !== "object" || issues === null) return [];
  const fieldErrors = (issues as { fieldErrors?: Record<string, unknown> }).fieldErrors;
  if (!fieldErrors || typeof fieldErrors !== "object") return [];
  const lines: string[] = [];
  for (const [field, val] of Object.entries(fieldErrors)) {
    if (Array.isArray(val) && val.length > 0 && val.every((v) => typeof v === "string")) {
      lines.push(`${field}: ${val.join("；")}`);
    }
  }
  return lines;
}

export function formatApiErrorMessage(body: unknown, fallback: string): string {
  if (typeof body === "object" && body !== null) {
    const record = body as Record<string, unknown>;
    if (typeof record.message === "string" && record.message.trim()) {
      const issuesLines = flattenFieldMessages(record.issues);
      if (issuesLines.length > 0) {
        return `${record.message}（${issuesLines.join("；")}）`;
      }
      return record.message;
    }
    if (typeof record.error === "string" && record.error.trim()) {
      return record.error;
    }
    const formErrors = record.formErrors;
    if (typeof formErrors === "object" && formErrors !== null) {
      const lines = Object.entries(formErrors as Record<string, unknown>)
        .map(([k, v]) =>
          Array.isArray(v) && v.every((x) => typeof x === "string")
            ? `${k}: ${(v as string[]).join("，")}`
            : null,
        )
        .filter(Boolean) as string[];
      if (lines.length > 0) return lines.join("；");
    }
  }
  return fallback;
}

export async function parseApiError(response: Response): Promise<ApiError> {
  let body: unknown;
  try {
    body = await response.json();
  } catch {
    body = undefined;
  }
  const fallback = response.statusText || "Request failed";
  const message = formatApiErrorMessage(body, fallback);

  return new ApiError(message, response.status, body);
}
