export type EditRevisions = Readonly<Record<string, string>>;
export type EditScope = `bookmark:${number}` | `focus_tree:${number}`;

export function withoutEditRevisions<T extends object>(value: T): T {
  const copy = { ...value } as T & {
    edit_revision?: string;
    tree_edit_revision?: string;
  };
  delete copy.edit_revision;
  delete copy.tree_edit_revision;
  return copy;
}

export function validEditRevisions(value: unknown): EditRevisions {
  if (!value || typeof value !== "object" || Array.isArray(value)) return {};
  return Object.fromEntries(
    Object.entries(value).filter(
      ([key, revision]) =>
        /^(bookmark|focus_tree):[1-9]\d*$/.test(key) &&
        typeof revision === "string" &&
        /^[1-9]\d*$/.test(revision),
    ),
  );
}

export function editRevisionSnapshot(
  scope: EditScope,
  revision: string | undefined,
): EditRevisions {
  return validEditRevisions({ [scope]: revision });
}

export function editRevisionHeaders(
  snapshot: EditRevisions,
): Record<string, string> {
  return { "X-Edit-Revisions": JSON.stringify(validEditRevisions(snapshot)) };
}

export function readEditRevisionHeaders(headers: Headers): EditRevisions {
  try {
    return validEditRevisions(
      JSON.parse(headers.get("X-Edit-Revisions") ?? "{}"),
    );
  } catch {
    return {};
  }
}

export function focusEditRevisions(
  trees: readonly { id: number; edit_revision?: string }[],
): EditRevisions {
  return validEditRevisions(
    Object.fromEntries(
      trees.map((tree) => [`focus_tree:${tree.id}`, tree.edit_revision]),
    ),
  );
}

export function createEditRevisionQueue() {
  type Transition = { scope: string; before: string; after: string };
  type Outcome =
    | { ok: true; value: unknown; transitions: Transition[] }
    | { ok: false; error: unknown; transitions: Transition[] };
  const tails = new Map<string, Promise<Outcome>>();
  return {
    async run<T>(
      projectId: number,
      snapshot: EditRevisions,
      execute: (
        revisions: EditRevisions,
      ) => Promise<{ value: T; revisions: EditRevisions }>,
      followPending = false,
    ): Promise<T> {
      const original = { ...validEditRevisions(snapshot) };
      const keys = Object.keys(original).sort();
      const pending = [
        ...new Set(
          keys
            .map((key) => tails.get(`${projectId}:${key}`))
            .filter((item): item is Promise<Outcome> => !!item),
        ),
      ];
      const operation: Promise<Outcome> = Promise.all(pending).then(
        async (settled) => {
          const transitions = settled.flatMap((item) => item.transitions);
          const submitted = { ...original };
          if (followPending) {
            for (const step of transitions) {
              if (submitted[step.scope] === step.before)
                submitted[step.scope] = step.after;
            }
          }
          try {
            const result = await execute(submitted);
            for (const [scope, after] of Object.entries(
              validEditRevisions(result.revisions),
            )) {
              const before = submitted[scope];
              if (before && before !== after)
                transitions.push({ scope, before, after });
            }
            return { ok: true, value: result.value, transitions };
          } catch (error) {
            return { ok: false, error, transitions };
          }
        },
      );
      for (const key of keys) tails.set(`${projectId}:${key}`, operation);
      try {
        const result = await operation;
        if (!result.ok) throw result.error;
        return result.value as T;
      } finally {
        for (const key of keys) {
          const identity = `${projectId}:${key}`;
          if (tails.get(identity) === operation) tails.delete(identity);
        }
      }
    },
  };
}

export const editRevisionQueue = createEditRevisionQueue();
