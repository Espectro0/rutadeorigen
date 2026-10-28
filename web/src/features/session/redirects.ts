import type { SessionState } from "./types";

export type ExpectedStatus = SessionState["status"];

export const APP_HOME = "/dashboard";
export const LOGIN_PAGE = "/";

export function redirectTarget(expected: ExpectedStatus, actual: ExpectedStatus): string {
    if (actual === expected) {
        return "";
    }
    switch (actual) {
        case "unauthenticated":
            return LOGIN_PAGE;
        case "unregistered":
            return "/onboarding";
        case "authenticated":
            return APP_HOME;
    }
}