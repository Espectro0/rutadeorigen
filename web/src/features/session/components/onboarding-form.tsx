"use client";

import { useState, type FormEvent } from "react";
import { useTranslations } from "next-intl";
import { isApiError } from "@/lib/api-error";
import { registerActor } from "../api";
import { useSession } from "../session-provider";
import { ACTOR_TYPES, type ValidActorType } from "../types";

export function OnboardingForm() {
    const t = useTranslations("Onboarding");
    const types = useTranslations("ActorTypes");
    const errors = useTranslations("Errors");
    const { reload } = useSession();

    const [type, setType] = useState<ValidActorType>("producer");
    const [name, setName] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [failure, setFailure] = useState("");

    function describe(error: unknown): string {
        if (isApiError(error)) {
            if (error.kind === "validation" || error.kind === "conflict") {
                return error.message;
            }
            return errors(`kind.${error.kind}`);
        }
        return errors("kind.unknown");
    }

    async function onSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setSubmitting(true);
        setFailure("");
        try {
            await registerActor({ type, name: name.trim() });
            reload();
        } catch (error) {
            setFailure(describe(error));
            setSubmitting(false);
        }
    }

    return (
        <form onSubmit={onSubmit} className="flex w-full max-w-sm flex-col gap-4 text-left">
            <div className="flex flex-col gap-1">
                <label htmlFor="actor-type" className="font-medium">
                    {t("typeLabel")}
                </label>
                <select
                    id="actor-type"
                    value={type}
                    onChange={(event) => setType(event.target.value as ValidActorType)}
                    className="rounded border px-3 py-2"
                >
                    {ACTOR_TYPES.map((option) => (
                        <option key={option} value={option}>
                            {types(option)}
                        </option>
                    ))}
                </select>
            </div>

            <div className="flex flex-col gap-1">
                <label htmlFor="actor-name" className="font-medium">
                    {t("nameLabel")}
                </label>
                <input
                    id="actor-name"
                    type="text"
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                    required
                    minLength={3}
                    maxLength={100}
                    autoComplete="organization"
                    className="rounded border px-3 py-2"
                />
            </div>

            {failure !== "" && (
                <p role="alert" className="text-red-600">
                    {failure}
                </p>
            )}

            <button
                type="submit"
                disabled={submitting}
                className="rounded bg-black px-6 py-3 text-white disabled:opacity-60 dark:bg-white dark:text-black"
            >
                {submitting ? t("submitting") : t("submit")}
            </button>
        </form>
    );
}