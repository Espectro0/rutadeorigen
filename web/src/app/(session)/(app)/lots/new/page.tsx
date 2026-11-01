import { getTranslations } from "next-intl/server";
import { LotForm } from "@/features/lots/components/lot-form";

export default async function NewLotPage() {
    const t = await getTranslations("Lots.form");

    return (
        <div className="flex flex-col gap-6">
            <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
            <LotForm />
        </div>
    );
}
