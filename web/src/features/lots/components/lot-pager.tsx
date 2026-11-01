"use client";

import { useTranslations } from "next-intl";

type Props = Readonly<{
    pages: number;
    current: number;
    hasNext: boolean;
    disabled: boolean;
    onSelect: (index: number) => void;
}>;

export function LotPager({ pages, current, hasNext, disabled, onSelect }: Props) {
    const t = useTranslations("Lots.pagination");

    return (
        <nav aria-label={t("label")} className="flex flex-wrap items-center justify-center gap-2">
            <button
                type="button"
                onClick={() => onSelect(current - 1)}
                disabled={disabled || current === 0}
                className="btn-secondary disabled:opacity-50"
            >
                {t("previous")}
            </button>

            {Array.from({ length: pages }, (_, index) => {
                const active = index === current;
                return (
                    <button
                        key={index}
                        type="button"
                        onClick={() => onSelect(index)}
                        disabled={disabled}
                        aria-current={active ? "page" : undefined}
                        aria-label={t("goTo", { page: index + 1 })}
                        className={
                            active
                                ? "btn-primary min-w-10 px-3 py-2"
                                : "btn-secondary min-w-10 px-3 py-2 disabled:opacity-50"
                        }
                    >
                        {index + 1}
                    </button>
                );
            })}

            <button
                type="button"
                onClick={() => onSelect(current + 1)}
                disabled={disabled || !hasNext}
                className="btn-secondary disabled:opacity-50"
            >
                {t("next")}
            </button>
        </nav>
    );
}
