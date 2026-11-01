import Image from "next/image";
import { getTranslations } from "next-intl/server";
import { DashboardStats } from "@/features/dashboard/dashboard-stats";

export default async function DashboardPage() {
    const t = await getTranslations("Dashboard");

    return (
        <div className="flex flex-col gap-6">
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
                <p className="text-ink-dim">{t("subtitle")}</p>
            </div>
            <DashboardStats />
        </div>
    );
}
