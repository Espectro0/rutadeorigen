import { request } from "@/lib/http-client";
import type { ScanStats } from "./types";

type ScanStatsDto = Readonly<{
    lot_id: string;
    total: number;
    first_scan_at: string | null;
    last_scan_at: string | null;
    days: number;
    daily: { date: string; count: number }[] | null;
}>;

export async function getScanStats(lotId: string, days: number): Promise<ScanStats> {
    const query = new URLSearchParams({ days: String(days) });
    const dto = await request<ScanStatsDto>(`/lots/${encodeURIComponent(lotId)}/scans/stats?${query}`);
    return {
        lotId: dto.lot_id,
        total: dto.total,
        firstScanAt: dto.first_scan_at ?? "",
        lastScanAt: dto.last_scan_at ?? "",
        days: dto.days,
        daily: dto.daily ?? [],
    };
}
