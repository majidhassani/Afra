import { useState, type FormEvent } from "react";
import { useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Pencil, Trash2, X } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { journalApi } from "@/shared/api/endpoints";
import { errorKey } from "@/shared/api/client";
import { EmptyState, ErrorState, SkeletonRows } from "@/shared/ui/states";
import { GuidancePanel } from "@/features/guidance/GuidancePanel";
import { toast } from "@/shared/ui/toast";
import type { JournalNote } from "@/shared/types/api";

export function JournalPage() {
  const { missionId } = useParams<{ missionId: string }>();
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState<JournalNote | "new" | null>(null);
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");

  const notes = useQuery({
    queryKey: ["mission", missionId, "journal"],
    queryFn: () => journalApi.list(missionId!),
    enabled: !!missionId,
  });

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: ["mission", missionId, "journal"] });

  const save = useMutation({
    mutationFn: () => {
      const input = { title: title.trim(), content: content.trim() };
      return editing === "new"
        ? journalApi.create(missionId!, input)
        : journalApi.update(missionId!, (editing as JournalNote).id, input);
    },
    onSuccess: () => {
      toast("success", editing === "new" ? t("journal.created") : t("journal.updated"));
      setEditing(null);
      void invalidate();
    },
  });

  const remove = useMutation({
    mutationFn: (noteId: string) => journalApi.remove(missionId!, noteId),
    onSuccess: () => {
      toast("success", t("journal.deleted"));
      void invalidate();
    },
  });

  const openEditor = (note: JournalNote | "new") => {
    setEditing(note);
    setTitle(note === "new" ? "" : note.title);
    setContent(note === "new" ? "" : note.content);
  };

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (content.trim() && !save.isPending) save.mutate();
  };

  return (
    <div className="page">
      <header className="page-header">
        <h1>{t("journal.title")}</h1>
        <button className="btn btn-primary" onClick={() => openEditor("new")}>
          <Plus size={15} aria-hidden />
          {t("journal.new")}
        </button>
      </header>

      <div className="dash-grid">
        <div className="col-8 stack">
          {editing !== null && (
            <form className="panel stack" style={{ padding: 16 }} onSubmit={onSubmit}>
              <div className="spread">
                <h3>{editing === "new" ? t("journal.new") : t("common.edit")}</h3>
                <button
                  type="button"
                  className="btn btn-ghost btn-icon"
                  aria-label={t("common.close")}
                  onClick={() => setEditing(null)}
                >
                  <X size={15} aria-hidden />
                </button>
              </div>
              <input
                className="input"
                value={title}
                maxLength={200}
                placeholder={t("journal.noteTitle")}
                aria-label={t("journal.noteTitle")}
                onChange={(e) => setTitle(e.target.value)}
              />
              <textarea
                className="textarea"
                rows={6}
                maxLength={10000}
                value={content}
                placeholder={t("journal.noteContent")}
                aria-label={t("journal.noteContent")}
                onChange={(e) => setContent(e.target.value)}
                style={{ unicodeBidi: "plaintext" }}
              />
              {save.isError && (
                <p className="field-error" role="alert">
                  {t(errorKey(save.error))}
                </p>
              )}
              <div className="row">
                <button
                  className="btn btn-primary"
                  type="submit"
                  disabled={save.isPending || !content.trim()}
                >
                  {t("common.save")}
                </button>
                <button
                  className="btn btn-ghost"
                  type="button"
                  onClick={() => setEditing(null)}
                >
                  {t("common.cancel")}
                </button>
              </div>
            </form>
          )}

          <div className="panel">
            {notes.isPending && <SkeletonRows rows={4} />}
            {notes.isError && (
              <ErrorState error={notes.error} onRetry={() => notes.refetch()} />
            )}
            {notes.isSuccess && notes.data.length === 0 && (
              <EmptyState
                title={t("journal.empty")}
                body={t("journal.empty.body")}
              />
            )}
            <div className="item-list">
              {notes.data?.map((note) => (
                <div key={note.id} className="item-row" style={{ alignItems: "flex-start" }}>
                  <div className="grow">
                    {note.title && <div className="title">{note.title}</div>}
                    <p
                      className="muted"
                      style={{ whiteSpace: "pre-wrap", unicodeBidi: "plaintext" }}
                    >
                      {note.content}
                    </p>
                    <span className="faint mono-num">
                      {new Date(note.updated_at).toLocaleString()}
                    </span>
                  </div>
                  <button
                    className="btn btn-ghost btn-icon"
                    aria-label={t("common.edit")}
                    title={t("common.edit")}
                    onClick={() => openEditor(note)}
                  >
                    <Pencil size={14} aria-hidden />
                  </button>
                  <button
                    className="btn btn-ghost btn-icon"
                    aria-label={t("common.delete")}
                    title={t("common.delete")}
                    disabled={remove.isPending}
                    onClick={() => {
                      if (window.confirm(t("journal.confirmDelete"))) {
                        remove.mutate(note.id);
                      }
                    }}
                  >
                    <Trash2 size={14} aria-hidden />
                  </button>
                </div>
              ))}
            </div>
          </div>
        </div>

        <section className="panel col-4" style={{ alignSelf: "flex-start" }}>
          <div style={{ padding: 16 }}>
            <GuidancePanel missionId={missionId!} screen="journal" />
          </div>
        </section>
      </div>
    </div>
  );
}
