import { useTranslations } from "next-intl";
import { LOT_STAGES, type LotStage } from "../types";

const PROGRESS_STAGES = LOT_STAGES.filter((stage) => stage !== "none");

export function StageProgress({ stage: current }: Readonly<{ stage: LotStage }>) {
    const t = useTranslations("Lots");
    const currentIndex = PROGRESS_STAGES.findIndex((stage) => stage === current);

    return (
        <section aria-label={t("detail.progress")} className="card p-5">
            <h2 className="mb-6 text-lg font-semibold">{t("detail.progress")}</h2>
            <ol className="relative flex flex-col gap-6 sm:flex-row sm:justify-between sm:gap-0">
                <div
                    aria-hidden
                    className="absolute hidden sm:block top-3.5 left-4 right-4 h-0.5 bg-ground-raised -z-0"
                />

                {PROGRESS_STAGES.map((stage, index) => {
                    const done = index < currentIndex;
                    const isCurrent = index === currentIndex;

                    return (
                        <li
                            key={stage}
                            aria-current={isCurrent ? "step" : undefined}
                            className="relative z-10 flex items-center gap-3 sm:flex-col sm:items-center sm:gap-2 sm:text-center"
                        >
                            <span
                                aria-hidden
                                className={`flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-semibold transition-colors ${
                                    done
                                        ? "bg-verified text-white"
                                        : isCurrent
                                            ? "bg-terracotta text-on-accent ring-4 ring-terracotta/20"
                                            : "bg-ground-raised text-ink-dim"
                                }`}
                            >
                                {done ? "✓" : index + 1}
                            </span>
                            <span className={`text-sm ${isCurrent ? "font-semibold text-ink" : "text-ink-dim"}`}>
                                {t(`stage.${stage}`)}
                            </span>
                        </li>
                    );
                })}
            </ol>
        </section>
    );
}
