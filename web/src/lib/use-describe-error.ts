"use client";

import { useTranslations } from "next-intl";
import { isApiError } from "./api-error";

export function useDescribeError(): (error: unknown) => string {
    const errors = useTranslations("Errors");

    return (error) => {
        if (isApiError(error)) {
            if (error.kind === "validation" || error.kind === "conflict") {
                return error.message;
            }
            return errors(`kind.${error.kind}`);
        }
        return errors("kind.unknown");
    };
}
