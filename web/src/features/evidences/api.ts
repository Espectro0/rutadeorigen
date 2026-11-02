import { ApiError } from "@/lib/api-error";
import { request } from "@/lib/http-client";
import { parseEvidenceStage, parseEvidenceStatus, type Evidence, type UploadTarget } from "./types";

type EvidenceDto = Readonly<{
    id: string;
    lot_id: string;
    stage: string;
    file_name: string;
    content_type: string;
    size_bytes: number;
    status: string;
    rejection_reason?: string;
    created_at: string;
    thumbnail_url?: string;
    image_url?: string;
    url_expires_at?: string;
}>;

type UploadDto = Readonly<{
    evidence: EvidenceDto;
    upload: Readonly<{ url: string; method: string; headers?: Record<string, string> | null }>;
}>;

type EvidenceListDto = Readonly<{ items: EvidenceDto[] | null }>;

function toEvidence(dto: EvidenceDto): Evidence {
    return {
        id: dto.id,
        lotId: dto.lot_id,
        stage: parseEvidenceStage(dto.stage),
        fileName: dto.file_name,
        contentType: dto.content_type,
        sizeBytes: dto.size_bytes,
        status: parseEvidenceStatus(dto.status),
        rejectionReason: dto.rejection_reason ?? "",
        createdAt: dto.created_at,
        thumbnailUrl: dto.thumbnail_url ?? "",
        imageUrl: dto.image_url ?? "",
        urlExpiresAt: dto.url_expires_at ?? "",
    };
}

function toTarget(dto: UploadDto["upload"]): UploadTarget {
    return { url: dto.url, method: dto.method, headers: dto.headers ?? {} };
}

export async function listEvidences(lotId: string): Promise<readonly Evidence[]> {
    const result = await request<EvidenceListDto>(`/lots/${encodeURIComponent(lotId)}/evidences`);
    return (result.items ?? []).map(toEvidence);
}

async function requestUpload(lotId: string, file: File): Promise<Readonly<{ evidence: Evidence; target: UploadTarget }>> {
    const dto = await request<UploadDto>(`/lots/${encodeURIComponent(lotId)}/evidences`, {
        method: "POST",
        body: JSON.stringify({ file_name: file.name, content_type: file.type, size_bytes: file.size }),
    });
    return { evidence: toEvidence(dto.evidence), target: toTarget(dto.upload) };
}

async function putToStorage(target: UploadTarget, file: File): Promise<void> {
    let response: Response;
    try {
        response = await fetch(target.url, { method: target.method, headers: target.headers, body: file });
    } catch {
        throw new ApiError({ code: "STORAGE_UNREACHABLE", kind: "unavailable", message: "storage unreachable", status: 0 });
    }

    if (!response.ok) {
        throw new ApiError({
            code: `STORAGE_${response.status}`,
            kind: "unavailable",
            message: "storage rejected the upload",
            status: response.status,
        });
    }
}

async function completeUpload(lotId: string, evidenceId: string): Promise<Evidence> {
    const dto = await request<EvidenceDto>(
        `/lots/${encodeURIComponent(lotId)}/evidences/${encodeURIComponent(evidenceId)}/complete`,
        { method: "POST" },
    );
    return toEvidence(dto);
}

export async function uploadEvidence(lotId: string, file: File): Promise<Evidence> {
    const { evidence, target } = await requestUpload(lotId, file);
    await putToStorage(target, file);
    return completeUpload(lotId, evidence.id);
}
