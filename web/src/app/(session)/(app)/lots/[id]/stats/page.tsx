import { ScanStats } from "@/features/scans/components/scan-stats";

export default async function LotStatsPage({ params }: { params: Promise<{ id: string }> }) {
    const { id } = await params;

    return <ScanStats lotId={id} />;
}
