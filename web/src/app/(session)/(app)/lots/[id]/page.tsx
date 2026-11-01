import { LotDetail } from "@/features/lots/components/lot-detail";

export default async function LotDetailPage({ params }: { params: Promise<{ id: string }> }) {
    const { id } = await params;

    return <LotDetail id={id} />;
}
