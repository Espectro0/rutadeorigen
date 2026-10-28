import Image from "next/image";
import { getTranslations } from "next-intl/server";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { Link } from "@/i18n/navigation";

export async function SiteHeader() {
    const t = await getTranslations("Header");

    return (
        <header className="flex items-center justify-between border-b border-hairline px-6 py-3">
            <Link href="/" className="flex items-center gap-2 font-display text-lg font-semibold text-terracotta">
                <Image src="/logo-without-bg.svg" alt="" width={28} height={28} />
                {t("brand")}
            </Link>
            <LocaleSwitcher />
        </header>
    );
}