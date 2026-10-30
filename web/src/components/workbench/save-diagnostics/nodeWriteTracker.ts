export interface NodeWriteTicket {
  readonly key: string;
  readonly generation: number;
  readonly sequence: number;
}

export type NodeWriteOutcome<T> = { ok: true; node: T } | { ok: false };
export type NodeWriteSettlement<T> =
  | { apply: true; canonical: T; latest: boolean; pending: number }
  | { apply: false; latest: boolean; pending: number };

interface WriteGroup<T> {
  generation: number;
  latest: number;
  confirmed: number;
  canonical: T;
  appliedFor?: number;
  pending: Set<number>;
}

export function createNodeWriteTracker<T>() {
  const groups = new Map<string, WriteGroup<T>>();
  let sequence = 0;

  return {
    begin(key: string, before: T): NodeWriteTicket {
      const id = ++sequence;
      let group = groups.get(key);
      if (!group || group.pending.size === 0) {
        group = {
          generation: id,
          latest: id,
          confirmed: 0,
          canonical: before,
          pending: new Set(),
        };
        groups.set(key, group);
      }
      group.latest = id;
      group.pending.add(id);
      return { key, generation: group.generation, sequence: id };
    },

    settle(
      ticket: NodeWriteTicket,
      outcome: NodeWriteOutcome<T>,
    ): NodeWriteSettlement<T> {
      const group = groups.get(ticket.key);
      if (
        !group ||
        group.generation !== ticket.generation ||
        !group.pending.has(ticket.sequence)
      ) {
        return {
          apply: false,
          latest: false,
          pending: group?.pending.size ?? 0,
        };
      }
      group.pending.delete(ticket.sequence);
      const latest = ticket.sequence === group.latest;
      if (outcome.ok && ticket.sequence > group.confirmed) {
        group.confirmed = ticket.sequence;
        group.canonical = outcome.node;
      }

      if (
        (latest && outcome.ok) ||
        (group.pending.size === 0 && group.appliedFor !== group.latest)
      ) {
        group.appliedFor = group.latest;
        return {
          apply: true,
          canonical: group.canonical,
          latest,
          pending: group.pending.size,
        };
      }
      return { apply: false, latest, pending: group.pending.size };
    },
  };
}
