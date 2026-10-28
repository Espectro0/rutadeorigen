import { ApiError, parseErrorKind } from "./api-error";

const BASE_URL = "/api";

function text(value: unknown): string {
    return typeof value === "string" ? value : "";
}

async function toApiError(response: Response): Promise<ApiError> {
    const body: unknown = await response.json().catch(() => ({}));
    const record = typeof body === "object" && body !== null ? (body as Record<string, unknown>) : {};

    return new ApiError({
        code: text(record.code) || `HTTP_${response.status}`,
        kind: parseErrorKind(text(record.kind)),
        message: text(record.error) || response.statusText,
        status: response.status,
        correlationId: text(record.correlation_id),
    });
}

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const hasBody = init.body !== undefined;
    const response = await fetch(`${BASE_URL}${path}`, {
        ...init,
        credentials: "include",
        headers: {
            Accept: "application/json",
            ...(hasBody ? { "Content-Type": "application/json" } : {}),
            ...init.headers,
        },
    });

    if (!response.ok) {
        throw await toApiError(response);
    }
    if (response.status === 204) {
        return undefined as T;
    }
    return (await response.json()) as T;
}