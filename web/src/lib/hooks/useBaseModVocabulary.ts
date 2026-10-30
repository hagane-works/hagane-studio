import { useEffect, useState } from "react";
import { projectsApi, type BaseModVocabulary } from "@/lib/api";

const EMPTY_VOCAB: BaseModVocabulary = { advisor_slots: [], idea_types: [] };

const cache = new Map<number, Promise<BaseModVocabulary>>();

function loadVocabulary(projectId: number): Promise<BaseModVocabulary> {
  const cached = cache.get(projectId);
  if (cached) return cached;
  const p = projectsApi
    .baseModVocabulary(projectId)
    .then((v) => ({
      advisor_slots: Array.isArray(v?.advisor_slots) ? v.advisor_slots : [],
      idea_types: Array.isArray(v?.idea_types) ? v.idea_types : [],
    }))

    .catch(() => EMPTY_VOCAB);
  cache.set(projectId, p);
  return p;
}

export function useBaseModVocabulary(projectId: number): BaseModVocabulary {
  const [vocab, setVocab] = useState<BaseModVocabulary>(EMPTY_VOCAB);

  useEffect(() => {
    let active = true;
    loadVocabulary(projectId).then((v) => {
      if (active) setVocab(v);
    });
    return () => {
      active = false;
    };
  }, [projectId]);

  return vocab;
}
