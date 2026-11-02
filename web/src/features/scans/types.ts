export const STATS_DAYS = [7, 30, 90, 365] as const;
export const DEFAULT_STATS_DAYS = 30;

export type DailyScans = Readonly<{ date: string; count: number }>;

export type ScanStats = Readonly<{
    lotId: string;
    total: number;
    firstScanAt: string;
    lastScanAt: string;
    days: number;
    daily: readonly DailyScans[];
}>;
