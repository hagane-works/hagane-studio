import { useLocaleStore } from "@/lib/store/localeStore";
import en from "../../../../messages/en/saveConflicts.json";
import zh from "../../../../messages/zh/saveConflicts.json";
import de from "../../../../messages/de/saveConflicts.json";
import es from "../../../../messages/es/saveConflicts.json";
import fr from "../../../../messages/fr/saveConflicts.json";
import ru from "../../../../messages/ru/saveConflicts.json";
import ja from "../../../../messages/ja/saveConflicts.json";
import ko from "../../../../messages/ko/saveConflicts.json";

const copy = { en, zh, de, es, fr, ru, ja, ko };
export function editRevisionErrorCode(
  value: unknown,
  depth = 0,
): keyof typeof en | null {
  if (
    value === "save_conflict" ||
    value === "save_precondition_required" ||
    value === "save_committed_host_publish_deferred" ||
    value === "save_committed_host_publish_failed"
  )
    return value;
  if (!value || typeof value !== "object" || depth > 3) return null;
  const item = value as Record<string, unknown>;
  for (const key of ["code", "error", "detail", "payload"]) {
    const code = editRevisionErrorCode(item[key], depth + 1);
    if (code) return code;
  }
  return null;
}

export function localizedEditRevisionError(error: unknown): unknown {
  const code = editRevisionErrorCode(error);
  if (!code) return error;
  const result = new Error(copy[useLocaleStore.getState().locale][code]);
  const committed =
    code === "save_committed_host_publish_deferred" ||
    code === "save_committed_host_publish_failed";
  return Object.assign(result, {
    code,
    cause: error,
    status: committed ? 500 : code === "save_conflict" ? 409 : 428,
    ...(committed ? { committed: true } : {}),
  });
}
