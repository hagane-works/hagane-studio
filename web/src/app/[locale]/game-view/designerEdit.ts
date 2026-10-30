import type {
  GuiDesignerCosts,
  GuiDesignerSaveInput,
  GuiDesignerUnit,
} from "@/lib/guiRenderApi";

export interface DesignerCell {
  type: string;
  x: number;
  y: number;
}

export interface DesignerDraft {
  regiments: DesignerCell[];
  support: DesignerCell[];
  regimentalSupport?: DesignerCell[];

  namesGroup?: string | null;

  priority?: number;

  name?: string;
}

export type DesignerUnit = GuiDesignerUnit;

export const EMPTY_DRAFT: DesignerDraft = { regiments: [], support: [] };

export const NEW_TEMPLATE = "__new_division__";

export const REGIMENT_COLUMNS = 5;
export const REGIMENT_ROWS = 5;
export const OPEN_REGIMENT_ROWS = 4;
export const SUPPORT_ROWS = 5;

export const REGIMENTAL_SUPPORT_BATTALIONS = 3;

export type SlotKind = "unit" | "add" | "locked" | "blocked";

export interface DesignerSlot {
  x: number;
  y: number;
  kind: SlotKind;
  cell?: DesignerCell;

  at?: { x: number; y: number };
}

export const HQ_BATTALIONS = 4;
export const HQ_SUPPORT = 4;

export function hqRegimentSlots(draft: DesignerDraft): {
  lead: DesignerSlot[];
  row: DesignerSlot[];
} {
  const cells = column(draft, 0);
  const lead: DesignerSlot[] = cells[0]
    ? [{ x: 0, y: 0, kind: "unit", cell: cells[0] }]
    : [{ x: 0, y: 0, kind: "add" }];
  const row: DesignerSlot[] = [];
  for (let y = 1; y < HQ_BATTALIONS; y += 1) {
    const cell = cells[y];
    const kind: SlotKind = cell
      ? "unit"
      : y === cells.length && cells.length > 0
        ? "add"
        : "locked";
    row.push({
      x: 0,
      y,
      kind,
      ...(cell ? { cell } : {}),
      at: { x: y - 1, y: 0 },
    });
  }
  return { lead, row };
}

export function hqSupportSlots(draft: DesignerDraft): DesignerSlot[] {
  const cells = [...draft.support].sort(byRow);
  const slots: DesignerSlot[] = [];
  for (let y = 0; y < HQ_SUPPORT; y += 1) {
    const cell = cells[y];
    const kind: SlotKind = cell
      ? "unit"
      : y === cells.length
        ? "add"
        : "locked";
    slots.push({
      x: 0,
      y,
      kind,
      ...(cell ? { cell } : {}),
      at: { x: y, y: 0 },
    });
  }
  return slots;
}

export function encodeCells(draft: DesignerDraft): string {
  const parts = [
    ...draft.regiments.map((cell) => `r:${cell.x}:${cell.y}:${cell.type}`),
    ...draft.support.map((cell) => `s:${cell.x}:${cell.y}:${cell.type}`),
    ...(draft.regimentalSupport ?? []).map(
      (cell) => `rs:${cell.x}:${cell.y}:${cell.type}`,
    ),
  ];
  return parts.length ? parts.join(",") : "-";
}

export function sameDraft(a: DesignerDraft, b: DesignerDraft): boolean {
  const cells = (draft: DesignerDraft) =>
    encodeCells(draft).split(",").sort().join(",");
  return (
    cells(a) === cells(b) &&
    (a.namesGroup ?? null) === (b.namesGroup ?? null) &&
    (a.priority ?? 1) === (b.priority ?? 1) &&
    (a.name ?? null) === (b.name ?? null)
  );
}

export function sameEntry(a: PickerEntry, b: PickerEntry): boolean {
  if (a.kind !== b.kind) return false;
  if (a.kind === "group" && b.kind === "group") return a.group === b.group;
  if (a.kind === "unit" && b.kind === "unit")
    return a.unit.type === b.unit.type;
  return true;
}

export function designerButtonsOff(opts: {
  dirty: boolean;
  fresh: boolean;
  cost: number;
  experience: number;
  empty?: boolean;

  editing?: boolean;
}): string[] {
  const off: string[] = [];
  if (opts.empty) off.push("division_equipment_btn");
  if (!opts.dirty) off.push("reset_button");
  if (opts.fresh) off.push("duplicate_button");
  if (!opts.dirty || (!opts.editing && opts.cost > opts.experience))
    off.push("save_button");
  return off;
}

export function designerSavePayload(
  draft: DesignerDraft,
  own: {
    tag: string;
    date?: string | null;
    template: string;
    name: string;
    namesGroup?: string | null;
    priority?: number;
  },
): GuiDesignerSaveInput {
  const cells = (list: readonly DesignerCell[]) =>
    list.map((cell) => ({ type: cell.type, x: cell.x, y: cell.y }));
  return {
    tag: own.tag,
    ...(own.date ? { date: own.date } : {}),
    template: own.template,
    name: (draft.name ?? own.name).trim(),
    names_group:
      (draft.namesGroup === undefined ? own.namesGroup : draft.namesGroup) ??
      "",
    priority: draft.priority ?? own.priority ?? 1,
    regiments: cells(draft.regiments),
    support: cells(draft.support),
    regimental_support: cells(draft.regimentalSupport ?? []),
  };
}

const byRow = (a: DesignerCell, b: DesignerCell) => a.y - b.y;

export function column(draft: DesignerDraft, x: number): DesignerCell[] {
  return draft.regiments.filter((cell) => cell.x === x).sort(byRow);
}

export function regimentSlots(draft: DesignerDraft): DesignerSlot[] {
  const slots: DesignerSlot[] = [];
  for (let x = 0; x < REGIMENT_COLUMNS; x += 1) {
    const cells = column(draft, x);
    const open = x === 0 || column(draft, x - 1).length > 0;
    if (!open) {
      slots.push({ x, y: 0, kind: "locked" });
      continue;
    }
    cells.forEach((cell, at) => slots.push({ x, y: at, kind: "unit", cell }));
    const next = cells.length;
    if (next < OPEN_REGIMENT_ROWS) {
      slots.push({ x, y: next, kind: "add" });
      if (next + 1 < OPEN_REGIMENT_ROWS)
        slots.push({ x, y: next + 1, kind: "locked" });
      else if (next + 1 < REGIMENT_ROWS)
        slots.push({ x, y: next + 1, kind: "blocked" });
    } else if (next < REGIMENT_ROWS) {
      slots.push({ x, y: next, kind: "blocked" });
    }
  }
  return slots;
}

export function supportSlots(draft: DesignerDraft): DesignerSlot[] {
  const cells = [...draft.support].sort(byRow);
  const slots: DesignerSlot[] = cells.map((cell, at) => ({
    x: 0,
    y: at,
    kind: "unit",
    cell,
  }));
  const next = cells.length;
  if (next < SUPPORT_ROWS) slots.push({ x: 0, y: next, kind: "add" });
  if (next + 1 < SUPPORT_ROWS)
    slots.push({ x: 0, y: next + 1, kind: "locked" });
  return slots;
}

export function regimentalSupportSlots(draft: DesignerDraft): DesignerSlot[] {
  const slots: DesignerSlot[] = [];
  for (let x = 0; x < REGIMENT_COLUMNS; x += 1) {
    const cell = draft.regimentalSupport?.find((cell) => cell.x === x);
    const kind = cell
      ? "unit"
      : column(draft, x).length >= REGIMENTAL_SUPPORT_BATTALIONS
        ? "add"
        : "locked";
    slots.push({ x, y: 0, kind, ...(cell ? { cell } : {}) });
  }
  return slots;
}

export function groupOf(
  draft: DesignerDraft,
  x: number,
  units: readonly DesignerUnit[],
): string | null {
  const first = column(draft, x)[0];
  if (!first) return null;
  return units.find((unit) => unit.type === first.type)?.group ?? null;
}

export function lineGroups(units: readonly DesignerUnit[]): string[] {
  const out: string[] = [];
  for (const unit of units) {
    if (unit.group !== "support" && !out.includes(unit.group))
      out.push(unit.group);
  }
  return out;
}

export interface PickerTarget {
  grid: "regiment" | "support" | "regimental";
  x: number;
  y: number;
}

export interface PickerState extends PickerTarget {
  group: string | null;
  fill?: boolean;
}

export type Costs = GuiDesignerCosts;

export type PickerEntry =
  | { kind: "remove"; cost: number }
  | { kind: "group"; group: string; cost: number | null }
  | {
      kind: "unit";
      unit: DesignerUnit;
      line: "add" | "replace" | "selected";
      cost: number;
    };

export function pickerEntries(
  draft: DesignerDraft,
  state: PickerState,
  units: readonly DesignerUnit[],
  costs: Costs,
  original: DesignerDraft = EMPTY_DRAFT,
): PickerEntry[] {
  const now = saveCost(original, draft, units, costs);
  const after = (next: DesignerDraft) =>
    saveCost(original, next, units, costs) - now;
  const removal = (): PickerEntry => ({
    kind: "remove",
    cost: after(removed(draft, state)),
  });
  const pick = (
    unit: DesignerUnit,
    line: "add" | "replace" | "selected",
  ): PickerEntry => {
    if (line === "selected") return { kind: "unit", unit, line, cost: 0 };
    const next =
      state.fill && state.grid === "regiment"
        ? filledRegiment(draft, state.x, unit.type, units)
        : placed(draft, state, unit.type, units);
    return { kind: "unit", unit, line, cost: Math.max(0, after(next)) };
  };
  if (state.grid === "regimental") {
    const current = draft.regimentalSupport?.find((cell) => cell.x === state.x);
    if (
      !current &&
      column(draft, state.x).length < REGIMENTAL_SUPPORT_BATTALIONS
    )
      return [];
    const companies = units.filter(
      (unit) => unit.group === "support" && unit.regimental !== false,
    );
    return [
      ...(current ? [removal()] : []),
      ...companies.map((unit) =>
        pick(
          unit,
          !current
            ? "add"
            : unit.type === current.type
              ? "selected"
              : "replace",
        ),
      ),
    ];
  }
  if (state.grid === "support") {
    const current = [...draft.support].sort(byRow)[state.y];
    const companies = units.filter(
      (unit) => unit.group === "support" && unit.divisional !== false,
    );
    if (!current) {
      const taken = new Set(draft.support.map((cell) => cell.type));
      return companies
        .filter((unit) => !taken.has(unit.type))
        .map((unit) => pick(unit, "add"));
    }
    return [
      removal(),
      ...companies.map((unit) =>
        pick(unit, unit.type === current.type ? "selected" : "replace"),
      ),
    ];
  }
  const cells = column(draft, state.x);
  const current = cells[state.y];
  const own = groupOf(draft, state.x, units);
  const group = state.group ?? (!current && own ? own : null);
  if (!group) {
    const groups = lineGroups(units);
    if (!current)
      return groups.map((name) => ({ kind: "group", group: name, cost: null }));
    return [
      removal(),
      ...groups.map((name): PickerEntry => ({
        kind: "group",
        group: name,
        cost: name === own ? null : costs.group + costs.change,
      })),
    ];
  }
  const members = units.filter((unit) => unit.group === group);
  if (!current) return members.map((unit) => pick(unit, "add"));
  return [
    removal(),
    ...members.map((unit) =>
      pick(unit, unit.type === current.type ? "selected" : "replace"),
    ),
  ];
}

export function placed(
  draft: DesignerDraft,
  target: PickerTarget,
  type: string,
  units: readonly DesignerUnit[],
): DesignerDraft {
  if (target.grid === "regimental") {
    const unit = units.find((unit) => unit.type === type);
    if (
      target.x < 0 ||
      target.x >= REGIMENT_COLUMNS ||
      target.y !== 0 ||
      column(draft, target.x).length < REGIMENTAL_SUPPORT_BATTALIONS ||
      unit?.group !== "support" ||
      unit.regimental === false
    )
      return draft;
    return {
      ...draft,
      regimentalSupport: [
        ...(draft.regimentalSupport ?? []).filter(
          (cell) => cell.x !== target.x,
        ),
        { type, x: target.x, y: 0 },
      ].sort((a, b) => a.x - b.x),
    };
  }
  if (target.grid === "support") {
    const cells = [...draft.support].sort(byRow);
    const at = Math.min(target.y, cells.length);
    const next = [...cells];
    next[at] = { type, x: 0, y: at };
    return { ...draft, support: next.map((cell, y) => ({ ...cell, y })) };
  }
  const cells = column(draft, target.x);
  const group = units.find((unit) => unit.type === type)?.group;
  const own = groupOf(draft, target.x, units);
  const at = Math.min(target.y, cells.length);
  let next = [...cells];
  next[at] = { type, x: target.x, y: at };
  if (own && group && own !== group)
    next = next.map((cell) => ({ ...cell, type }));
  const others = draft.regiments.filter((cell) => cell.x !== target.x);
  return {
    ...draft,
    regiments: [...others, ...next.map((cell, y) => ({ ...cell, y }))],
  };
}

export function swappedAlone(
  draft: DesignerDraft,
  target: PickerTarget,
  type: string,
): DesignerDraft {
  if (target.grid !== "regiment") return draft;
  const cells = column(draft, target.x);
  if (!cells[target.y]) return draft;
  const others = draft.regiments.filter((cell) => cell.x !== target.x);
  return {
    ...draft,
    regiments: [
      ...others,
      ...cells.map((cell, y) => (y === target.y ? { ...cell, type } : cell)),
    ],
  };
}

export function removed(
  draft: DesignerDraft,
  target: PickerTarget,
): DesignerDraft {
  if (target.grid === "regimental") {
    return {
      ...draft,
      regimentalSupport: (draft.regimentalSupport ?? []).filter(
        (cell) => cell.x !== target.x,
      ),
    };
  }
  if (target.grid === "support") {
    const cells = [...draft.support]
      .sort(byRow)
      .filter((_, y) => y !== target.y);
    return { ...draft, support: cells.map((cell, y) => ({ ...cell, y })) };
  }
  const columns: DesignerCell[][] = [];
  const regimentalSupport: DesignerCell[] = [];
  for (let x = 0; x < REGIMENT_COLUMNS; x += 1) {
    const cells = column(draft, x).filter(
      (_, y) => !(x === target.x && y === target.y),
    );
    if (cells.length) {
      const company = draft.regimentalSupport?.find((cell) => cell.x === x);
      if (company && cells.length >= REGIMENTAL_SUPPORT_BATTALIONS) {
        regimentalSupport.push({ ...company, x: columns.length, y: 0 });
      }
      columns.push(cells);
    }
  }
  return {
    ...draft,
    regiments: columns.flatMap((cells, x) =>
      cells.map((cell, y) => ({ ...cell, x, y })),
    ),
    ...(draft.regimentalSupport ? { regimentalSupport } : {}),
  };
}

export function removedRegiment(
  draft: DesignerDraft,
  x: number,
): DesignerDraft {
  let next = draft;
  for (let left = column(draft, x).length; left > 0; left -= 1)
    next = removed(next, { grid: "regiment", x, y: 0 });
  return next;
}

export function filledRegiment(
  draft: DesignerDraft,
  x: number,
  type: string,
  units: readonly DesignerUnit[],
  from = 0,
): DesignerDraft {
  let next = draft;
  const keep = column(draft, x).slice(0, from).length;
  for (let y = keep; y < OPEN_REGIMENT_ROWS; y += 1)
    next = placed(next, { grid: "regiment", x, y }, type, units);

  return next;
}

export function saveCost(
  original: DesignerDraft,
  draft: DesignerDraft,
  units: readonly DesignerUnit[],
  costs: Costs,
): number {
  let total = 0;
  for (let x = 0; x < REGIMENT_COLUMNS; x += 1) {
    const before = column(original, x);
    const after = column(draft, x);
    if (!before.length && !after.length) continue;
    const was = groupOf(original, x, units);
    const now = groupOf(draft, x, units);
    if (was && now && was !== now)
      total +=
        costs.group + costs.change * Math.max(before.length, after.length);
    else total += costs.change * changes(before, after);
  }
  total += costs.support * changes(original.support, draft.support);

  for (let x = 0; x < REGIMENT_COLUMNS; x += 1) {
    total +=
      costs.support *
      changes(
        (original.regimentalSupport ?? []).filter((cell) => cell.x === x),
        (draft.regimentalSupport ?? []).filter((cell) => cell.x === x),
      );
  }
  return total;
}

function changes(
  before: readonly DesignerCell[],
  after: readonly DesignerCell[],
): number {
  const left = new Map<string, number>();
  for (const cell of before)
    left.set(cell.type, (left.get(cell.type) ?? 0) + 1);
  let added = 0;
  for (const cell of after) {
    const have = left.get(cell.type) ?? 0;
    if (have > 0) left.set(cell.type, have - 1);
    else added += 1;
  }
  let taken = 0;
  for (const count of left.values()) taken += count;
  return added + taken;
}

export function draftOf(data: {
  regiments: readonly { type: string; x?: number; y?: number }[];
  support: readonly { type: string; x?: number; y?: number }[];
  regimental_support?: readonly { type: string; x?: number; y?: number }[];
  division_names_group?: string;
  priority?: number;
}): DesignerDraft {
  return {
    regiments: data.regiments.map((cell) => ({
      type: cell.type,
      x: cell.x ?? 0,
      y: cell.y ?? 0,
    })),
    support: data.support.map((cell) => ({
      type: cell.type,
      x: cell.x ?? 0,
      y: cell.y ?? 0,
    })),
    regimentalSupport: (data.regimental_support ?? []).map((cell) => ({
      type: cell.type,
      x: cell.x ?? 0,
      y: cell.y ?? 0,
    })),
    namesGroup: data.division_names_group || null,
    priority: data.priority ?? 1,
  };
}
