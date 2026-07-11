import { useEffect, useRef, useState, type FormEvent } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, SendHorizonal, ImagePlus } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { useLanguageGuard } from "@/shared/i18n/languageGuard";
import { announceTimeUpdate } from "@/features/game/gameEvents";
import { charactersApi, walletApi } from "@/shared/api/endpoints";
import { errorKey } from "@/shared/api/client";
import { ErrorState, SkeletonRows } from "@/shared/ui/states";
import { CostBadge, Meter } from "@/shared/ui/badges";
import { Avatar } from "@/shared/ui/Avatar";
import {
  AttachmentStrip,
  ImageAttachButton,
  useImageAttachments,
  useImageDrop,
} from "@/shared/ui/imageUpload";
import { Button } from "@/shared/ui/Button";
import type { ChatResult, ImagePayload } from "@/shared/types/api";

interface ThreadEntry {
  id: string;
  sender: "player" | "npc";
  content: string;
  imagePreviews?: string[];
  result?: ChatResult;
}

export function CharacterChatPage() {
  const { missionId, characterId } = useParams<{
    missionId: string;
    characterId: string;
  }>();
  const { t } = useI18n();
  const guardLanguage = useLanguageGuard();
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState("");
  const [thread, setThread] = useState<ThreadEntry[]>([]);
  const threadRef = useRef<HTMLDivElement>(null);
  const {
    attachments,
    addFiles,
    remove: removeAttachment,
    clear: clearAttachments,
    error: attachError,
  } = useImageAttachments();
  const { dragging, dropProps } = useImageDrop((files) => void addFiles(files));

  const detail = useQuery({
    queryKey: ["mission", missionId, "character", characterId],
    queryFn: () => charactersApi.detail(missionId!, characterId!),
    enabled: !!missionId && !!characterId,
  });

  const pricing = useQuery({
    queryKey: ["wallet", "pricing"],
    queryFn: walletApi.pricing,
  });
  const chatCost = pricing.data?.character_chat ?? null;

  const generateAvatar = useMutation({
    mutationFn: () => charactersApi.generateAvatar(missionId!, characterId!),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["mission", missionId, "character", characterId],
      });
    },
  });

  // Seed the visible thread from the persisted transcript once loaded.
  useEffect(() => {
    if (detail.data && thread.length === 0 && detail.data.messages.length > 0) {
      setThread(
        detail.data.messages.map((m) => ({
          id: m.id,
          sender: m.sender === "player" || m.sender === "user" ? "player" : "npc",
          content: m.content,
        })),
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [detail.data]);

  useEffect(() => {
    threadRef.current?.scrollTo({ top: threadRef.current.scrollHeight });
  }, [thread.length]);

  const send = useMutation({
    mutationFn: ({ message, images }: { message: string; images: ImagePayload[] }) =>
      charactersApi.chat(missionId!, characterId!, message, images),
    onMutate: ({ message }) => {
      setThread((prev) => [
        ...prev,
        {
          id: `local-${Date.now()}`,
          sender: "player",
          content: message,
          imagePreviews: attachments.map((a) => a.previewUrl),
        },
      ]);
      setDraft("");
      clearAttachments();
    },
    onSuccess: (result) => {
      guardLanguage(result.message);
      setThread((prev) => [
        ...prev,
        {
          id: `npc-${Date.now()}`,
          sender: "npc",
          content: result.message,
          result,
        },
      ]);
      // Talking costs mission time; the interview may also advance a stage.
      announceTimeUpdate(result.time_update);
      void queryClient.invalidateQueries({ queryKey: ["wallet"] });
      void queryClient.invalidateQueries({ queryKey: ["gameplay-status", missionId] });
      void queryClient.invalidateQueries({
        queryKey: ["mission", missionId, "character", characterId],
      });
    },
  });

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    const message = draft.trim();
    if (message && !send.isPending) {
      send.mutate({ message, images: attachments.map((a) => a.payload) });
    }
  };

  if (detail.isPending) {
    return (
      <div className="page">
        <SkeletonRows rows={7} />
      </div>
    );
  }
  if (detail.isError) {
    return (
      <div className="page">
        <ErrorState error={detail.error} onRetry={() => detail.refetch()} />
      </div>
    );
  }

  const { character } = detail.data;

  return (
    <div className="chat-layout">
      <div className="band" style={{ padding: "12px 24px" }}>
        <div className="row" style={{ gap: 12 }}>
          <Button
            to={`/app/missions/${missionId}/characters`}
            ariaLabel={t("common.back")}
            variant="subtle"
            className="btn-icon"
          >
            <ArrowLeft size={16} className="rtl-flip" aria-hidden />
          </Button>
          <div className="portrait-wrap">
            <Avatar
              name={character.name}
              category={character.category}
              imageUrl={character.avatar_url || undefined}
              version={character.avatar_version}
              size="lg"
            />
            <button
              className="portrait-gen"
              title={t("chars.portrait.generate")}
              disabled={generateAvatar.isPending}
              onClick={() => generateAvatar.mutate()}
              aria-label={t("chars.portrait.generate")}
            >
              <ImagePlus size={12} aria-hidden />
            </button>
          </div>
          <div className="grow" style={{ minWidth: 0 }}>
            <h1 className="chat-character-title">{character.name}</h1>
            <p className="sub muted">
              {character.role} · {character.category}
            </p>
          </div>
          <div className="row" style={{ flexWrap: "wrap", justifyContent: "flex-end" }}>
            <span className="row faint" style={{ gap: 6 }}>
              {t("chars.trust")}
              <Meter value={character.trust_level} />
            </span>
            <span className="chip">{t("chars.mood")}: {character.mood}</span>
            {chatCost !== null && <CostBadge coins={chatCost} />}
          </div>
        </div>
        {character.public_profile && (
          <p className="faint" style={{ marginTop: 8, unicodeBidi: "plaintext" }}>
            {character.public_profile}
          </p>
        )}
      </div>

      <div className="chat-thread" ref={threadRef}>
        {thread.length === 0 && (
          <p className="faint" style={{ textAlign: "center" }}>
            {t("chars.chat.hint")}
          </p>
        )}
        {thread.map((entry) => (
          <div key={entry.id} className={`bubble ${entry.sender}`}>
            {entry.imagePreviews && entry.imagePreviews.length > 0 && (
              <span className="row" style={{ flexWrap: "wrap", gap: 6 }}>
                {entry.imagePreviews.map((src, i) => (
                  <img
                    key={src}
                    src={src}
                    // Real content, not decoration — a screen reader with
                    // alt="" would silently drop this attachment entirely.
                    alt={
                      entry.imagePreviews!.length > 1
                        ? t("chars.chat.attachmentOf", {
                            index: i + 1,
                            count: entry.imagePreviews!.length,
                          })
                        : t("chars.chat.attachment")
                    }
                    style={{
                      width: 96,
                      height: 96,
                      objectFit: "cover",
                      borderRadius: 8,
                    }}
                  />
                ))}
              </span>
            )}
            {entry.content}
            {entry.result && (
              <span className="meta row" style={{ flexWrap: "wrap", gap: 6 }}>
                <span>{entry.result.emotion}</span>
                <span className="mono-num">
                  {t("chars.trust")} {entry.result.trust_delta >= 0 ? "+" : ""}
                  {entry.result.trust_delta}
                </span>
                <span className="mono-num">
                  {t("chars.stress")} {entry.result.stress_delta >= 0 ? "+" : ""}
                  {entry.result.stress_delta}
                </span>
                <CostBadge coins={entry.result.cost.coins_charged} />
              </span>
            )}
            {entry.result && entry.result.unlocked_clues.length > 0 && (
              <span className="meta row" style={{ flexWrap: "wrap", gap: 6 }}>
                {t("chars.unlockedClues")}:
                {entry.result.unlocked_clues.map((clue) => (
                  <Link
                    key={clue.id}
                    className="chip chip-wallet"
                    to={`/app/missions/${missionId}/clues/${clue.id}`}
                  >
                    {clue.title}
                  </Link>
                ))}
              </span>
            )}
            {entry.result && entry.result.new_facts.length > 0 && (
              <span className="meta">
                {entry.result.new_facts.join(" · ")}
              </span>
            )}
          </div>
        ))}
        {send.isPending && (
          <div className="bubble npc typing" aria-label={t("chars.typing")}>
            <span className="typing-dots" aria-hidden>
              <i />
              <i />
              <i />
            </span>
            <span className="faint" style={{ fontSize: 11 }}>
              {t("chars.typing", { name: character.name })}
            </span>
          </div>
        )}
        {send.isError && (
          <p className="field-error" role="alert">
            {t(errorKey(send.error))}
          </p>
        )}
      </div>

      <div
        {...dropProps}
        style={dragging ? { outline: "2px dashed var(--accent, #888)", borderRadius: 8 } : undefined}
      >
        <AttachmentStrip
          attachments={attachments}
          onRemove={removeAttachment}
          error={attachError}
        />
        <form className="chat-composer" onSubmit={onSubmit}>
          <ImageAttachButton onFiles={(f) => void addFiles(f)} disabled={send.isPending} />
          <textarea
            className="textarea"
            value={draft}
            placeholder={t("chars.chat.placeholder", { name: character.name })}
            aria-label={t("chars.chat.placeholder", { name: character.name })}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                onSubmit(e);
              }
            }}
          />
          <Button
            type="submit"
            loading={send.isPending}
            disabled={!draft.trim()}
            ariaLabel={t("common.send")}
          >
            <SendHorizonal size={15} className="rtl-flip" aria-hidden />
          </Button>
        </form>
      </div>
    </div>
  );
}
