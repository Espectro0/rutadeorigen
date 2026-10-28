"use client";

import { useEffect, type ReactNode } from "react";
import { useTranslations } from "next-intl";
import { useRouter } from "@/i18n/navigation";
import { redirectTarget, type ExpectedStatus } from "./redirects";
import { useSession } from "./session-provider";

export function RequireSession({
                                   expected,
                                   children,
                               }: {
    expected: ExpectedStatus;
    children: ReactNode;
}) {
    const { session, reload } = useSession();
    const router = useRouter();
    const t = useTranslations("Session");
    const errors = useTranslations("Errors");

    const target =
        session.status === "loading" || session.status === "error"
            ? ""
            : redirectTarget(expected, session.status);

    useEffect(() => {
        if (target !== "") {
            router.replace(target);
        }
    }, [target, router]);

    if (session.status === "error") {
        return (
            <div role="alert" className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
                <p>{errors(`kind.${session.kind}`)}</p>
                {session.correlationId !== "" && (
                    <p className="text-sm text-zinc-500">
                        {t("reference")}: {session.correlationId}
                    </p>
                )}
                <button type="button" onClick={reload} className="rounded border px-4 py-2">
                    {t("retry")}
                </button>
            </div>
        );
    }

    if (session.status === "loading" || target !== "") {
        return (
            <p role="status" className="flex flex-1 items-center justify-center p-8">
                {t("loading")}
            </p>
        );
    }

    return <>{children}</>;
}