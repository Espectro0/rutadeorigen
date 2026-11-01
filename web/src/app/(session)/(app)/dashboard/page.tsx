import { getTranslations } from "next-intl/server";
import { DashboardWelcome } from "@/components/dashboard-welcome";
import { DashboardStats } from "@/features/dashboard/dashboard-stats";

export default async function DashboardPage() {
    const t = await getTranslations("Dashboard");

    return (
        <div className="flex flex-col gap-6">
            <DashboardWelcome />
            <DashboardStats />
        </div>
    );
}
