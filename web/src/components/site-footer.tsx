import { getTranslations } from "next-intl/server";

const REPO_URL = "https://github.com/Espectro0/rutadeorigen";
const linkClass = "underline underline-offset-4 hover:text-terracotta";

export async function SiteFooter() {
    const t = await getTranslations("Footer");

    return (
        <footer className="border-t border-hairline px-6 py-4 text-center text-sm text-ink-dim">
            <p>
                {t("openSource")}{" "}
                <a href={`${REPO_URL}/blob/main/LICENSE`} target="_blank" rel="noopener noreferrer" className={linkClass}>
                    AGPL-3.0
                </a>
                {" · "}
                <a href={REPO_URL} target="_blank" rel="noopener noreferrer" className={linkClass}>
                    {t("repository")}
                </a>
            </p>
        </footer>
    );
}