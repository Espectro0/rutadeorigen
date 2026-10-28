export const ERROR_KINDS = [
    "validation",
    "unauthorized",
    "forbidden",
    "not_found",
    "conflict",
    "internal",
    "unavailable",
] as const;

export type ValidErrorKind = (typeof ERROR_KINDS)[number];
export type ErrorKind = ValidErrorKind | "unknown";

export function parseErrorKind(value: string): ErrorKind {
    return (ERROR_KINDS as readonly string[]).includes(value) ? (value as ValidErrorKind) : "unknown";
}

export class ApiError extends Error {
    readonly code: string;
    readonly kind: ErrorKind;
    readonly status: number;
    readonly correlationId: string;

    constructor(params: {
        code: string;
        kind: ErrorKind;
        message: string;
        status: number;
        correlationId?: string;
    }) {
        super(params.message);
        this.name = "ApiError";
        this.code = params.code;
        this.kind = params.kind;
        this.status = params.status;
        this.correlationId = params.correlationId ?? "";
    }
}

export function isApiError(error: unknown): error is ApiError {
    return error instanceof ApiError;
}