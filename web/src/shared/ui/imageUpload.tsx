import { useCallback, useRef, useState, type DragEvent } from "react";
import { ImagePlus, X } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import type { ImagePayload } from "@/shared/types/api";

const ACCEPTED = ["image/png", "image/jpeg", "image/webp"];
const MAX_BYTES = 4 * 1024 * 1024;
export const MAX_IMAGES = 4;

export interface Attachment {
  payload: ImagePayload;
  previewUrl: string;
  name: string;
}

async function fileToAttachment(file: File): Promise<Attachment> {
  const mime = file.type === "image/jpg" ? "image/jpeg" : file.type;
  const buf = await file.arrayBuffer();
  let binary = "";
  const bytes = new Uint8Array(buf);
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunk));
  }
  return {
    payload: { data: btoa(binary), mime },
    previewUrl: URL.createObjectURL(file),
    name: file.name,
  };
}

/**
 * Attachment state for vision-capable requests: validates type/size/count
 * client-side and produces base64 payloads plus object-URL previews.
 */
export function useImageAttachments(max = MAX_IMAGES) {
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [error, setError] = useState<"type" | "size" | "count" | null>(null);

  const addFiles = useCallback(
    async (files: FileList | File[]) => {
      setError(null);
      const list = Array.from(files);
      const next: Attachment[] = [];
      for (const file of list) {
        if (!ACCEPTED.includes(file.type) && file.type !== "image/jpg") {
          setError("type");
          continue;
        }
        if (file.size > MAX_BYTES) {
          setError("size");
          continue;
        }
        next.push(await fileToAttachment(file));
      }
      setAttachments((prev) => {
        const merged = [...prev, ...next];
        if (merged.length > max) {
          setError("count");
          merged.slice(max).forEach((a) => URL.revokeObjectURL(a.previewUrl));
          return merged.slice(0, max);
        }
        return merged;
      });
    },
    [max],
  );

  const remove = useCallback((index: number) => {
    setAttachments((prev) => {
      const target = prev[index];
      if (target) URL.revokeObjectURL(target.previewUrl);
      return prev.filter((_, i) => i !== index);
    });
    setError(null);
  }, []);

  const clear = useCallback(() => {
    setAttachments((prev) => {
      prev.forEach((a) => URL.revokeObjectURL(a.previewUrl));
      return [];
    });
    setError(null);
  }, []);

  return { attachments, addFiles, remove, clear, error };
}

interface ImageAttachButtonProps {
  onFiles: (files: FileList) => void;
  disabled?: boolean;
}

/** Paper-clip style button that opens the image file picker. */
export function ImageAttachButton({ onFiles, disabled }: ImageAttachButtonProps) {
  const { t } = useI18n();
  const inputRef = useRef<HTMLInputElement>(null);
  return (
    <>
      <input
        ref={inputRef}
        type="file"
        accept={ACCEPTED.join(",")}
        multiple
        hidden
        onChange={(e) => {
          if (e.target.files?.length) onFiles(e.target.files);
          e.target.value = "";
        }}
      />
      <button
        type="button"
        className="btn btn-ghost btn-icon"
        aria-label={t("images.attach")}
        title={t("images.attach")}
        disabled={disabled}
        onClick={() => inputRef.current?.click()}
      >
        <ImagePlus size={15} aria-hidden />
      </button>
    </>
  );
}

interface AttachmentStripProps {
  attachments: Attachment[];
  onRemove: (index: number) => void;
  error: "type" | "size" | "count" | null;
}

/** Thumbnails of pending attachments with remove buttons. */
export function AttachmentStrip({ attachments, onRemove, error }: AttachmentStripProps) {
  const { t } = useI18n();
  if (attachments.length === 0 && !error) return null;
  return (
    <div className="row" style={{ flexWrap: "wrap", gap: 8, padding: "6px 0" }}>
      {attachments.map((a, i) => (
        <span key={a.previewUrl} className="chip" style={{ gap: 6, padding: 4 }}>
          <img
            src={a.previewUrl}
            alt={a.name}
            style={{ width: 36, height: 36, objectFit: "cover", borderRadius: 6 }}
          />
          <button
            type="button"
            className="btn btn-ghost btn-icon"
            aria-label={t("common.delete")}
            onClick={() => onRemove(i)}
          >
            <X size={12} aria-hidden />
          </button>
        </span>
      ))}
      {error && (
        <span className="field-error" role="alert">
          {t(
            error === "type"
              ? "images.error.type"
              : error === "size"
                ? "images.error.size"
                : "images.error.count",
          )}
        </span>
      )}
    </div>
  );
}

/** Wires drag & drop of image files onto any container. */
export function useImageDrop(onFiles: (files: FileList) => void) {
  const [dragging, setDragging] = useState(false);
  const onDragOver = useCallback((e: DragEvent) => {
    e.preventDefault();
    setDragging(true);
  }, []);
  const onDragLeave = useCallback((e: DragEvent) => {
    e.preventDefault();
    setDragging(false);
  }, []);
  const onDrop = useCallback(
    (e: DragEvent) => {
      e.preventDefault();
      setDragging(false);
      if (e.dataTransfer.files?.length) onFiles(e.dataTransfer.files);
    },
    [onFiles],
  );
  return { dragging, dropProps: { onDragOver, onDragLeave, onDrop } };
}
