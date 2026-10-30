import { useRef, useState, type ChangeEvent } from "react";
import { Check, Upload } from "lucide-react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  PET_AVATARS,
  avatarSource,
  imageFileToAvatarDataUrl,
  petAvatarSource,
  useAvatarStore,
  type PetAvatar,
} from "./avatar";

const MAX_FILE_SIZE = 10 * 1024 * 1024;
const acceptedTypes = new Set(["image/png", "image/jpeg", "image/webp"]);

export function AvatarSettings() {
  const t = useTranslations("desktop.settings.avatar");
  const inputRef = useRef<HTMLInputElement>(null);
  const selection = useAvatarStore((state) => state.selection);
  const choosePet = useAvatarStore((state) => state.choosePet);
  const chooseCustom = useAvatarStore((state) => state.chooseCustom);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function upload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    if (!acceptedTypes.has(file.type.toLowerCase())) {
      setError(t("invalidType"));
      return;
    }
    if (file.size > MAX_FILE_SIZE) {
      setError(t("tooLarge"));
      return;
    }
    setBusy(true);
    setError("");
    try {
      chooseCustom(await imageFileToAvatarDataUrl(file));
    } catch {
      setError(t("failed"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="space-y-3 border-b border-border pb-4">
      <div className="flex items-center gap-3">
        <img
          src={avatarSource(selection)}
          alt={t("currentAlt")}
          className={cn(
            "size-16 shrink-0 rounded-full border border-border bg-muted",
            selection.kind === "pet" ? "object-contain p-1" : "object-cover",
          )}
        />
        <div className="min-w-0 space-y-1">
          <h3 className="text-sm font-medium">{t("title")}</h3>
          <p className="text-xs text-muted-foreground">{t("about")}</p>
        </div>
      </div>
      <div className="grid grid-cols-4 gap-2 sm:grid-cols-8">
        {PET_AVATARS.map((id) => (
          <PetChoice
            key={id}
            id={id}
            name={t(`pets.${id}`)}
            selected={selection.kind === "pet" && selection.id === id}
            onChoose={() => {
              setError("");
              choosePet(id);
            }}
          />
        ))}
      </div>
      <input
        ref={inputRef}
        type="file"
        accept="image/png,image/jpeg,image/webp"
        className="hidden"
        aria-label={t("upload")}
        onChange={(event) => void upload(event)}
      />
      <div className="flex flex-wrap items-center gap-3">
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={busy}
          onClick={() => inputRef.current?.click()}
        >
          <Upload className="size-4" />
          {busy ? t("processing") : t("upload")}
        </Button>
        <span className="text-xs text-muted-foreground">{t("uploadHelp")}</span>
      </div>
      {error ? (
        <p role="alert" className="text-xs text-destructive">
          {error}
        </p>
      ) : null}
    </section>
  );
}

function PetChoice({
  id,
  name,
  selected,
  onChoose,
}: {
  id: PetAvatar;
  name: string;
  selected: boolean;
  onChoose: () => void;
}) {
  return (
    <button
      type="button"
      aria-label={name}
      aria-pressed={selected}
      title={name}
      onClick={onChoose}
      className={cn(
        "relative aspect-square rounded-lg border bg-muted/50 p-1 ui-interactive ui-focus-ring",
        selected
          ? "border-accent ring-2 ring-accent/30"
          : "border-border hover:bg-muted",
      )}
    >
      <img
        src={petAvatarSource(id)}
        alt=""
        className="size-full object-contain"
      />
      {selected ? (
        <span className="absolute right-0.5 bottom-0.5 grid size-4 place-items-center rounded-full bg-accent text-accent-foreground">
          <Check className="size-3" aria-hidden />
        </span>
      ) : null}
    </button>
  );
}
