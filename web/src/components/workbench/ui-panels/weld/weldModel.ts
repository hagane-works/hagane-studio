import type { UiPanel, UiPanelControl } from "@/types";
import type { PanelContractFinding } from "@/lib/api/panelContract";

export type WeldRowKind =
  | "window"
  | "panelVisible"
  | "visible"
  | "loc"
  | "sprite"
  | "raw"
  | "click"
  | "clickEnabled"
  | "tooltip";

export interface WeldRow {
  id: string;
  kind: WeldRowKind;

  label: string;

  controlName: string;

  file: string;
  findings: PanelContractFinding[];

  worst: "error" | "warning" | null;

  fixableLocKey: string | null;

  pendingSave: boolean;

  infoNote?: string;
}

function worstOf(findings: PanelContractFinding[]): "error" | "warning" | null {
  if (findings.some((f) => f.severity === "error")) return "error";
  if (findings.some((f) => f.severity === "warning")) return "warning";
  return null;
}

function makeRow(
  id: string,
  kind: WeldRowKind,
  label: string,
  findings: PanelContractFinding[],
  extra?: Partial<
    Pick<
      WeldRow,
      "controlName" | "file" | "fixableLocKey" | "pendingSave" | "infoNote"
    >
  >,
): WeldRow {
  return {
    id,
    kind,
    label,
    controlName: extra?.controlName ?? "",
    file: extra?.file ?? "",
    findings,
    worst: worstOf(findings),
    fixableLocKey: extra?.fixableLocKey ?? null,
    pendingSave: extra?.pendingSave ?? false,
    infoNote: extra?.infoNote,
  };
}

export function flattenControls(controls: UiPanelControl[]): UiPanelControl[] {
  const out: UiPanelControl[] = [];
  const visit = (list?: UiPanelControl[]) => {
    for (const c of list ?? []) {
      out.push(c);
      if (c.kind === "container") visit(c.children);
    }
  };
  visit(controls);
  return out;
}

export function findingsForPanel(
  findings: PanelContractFinding[] | undefined,
  panel: UiPanel,
): PanelContractFinding[] {
  return (findings ?? []).filter(
    (f) => f.entity_ref?.entity_id === panel.token,
  );
}

export function buildWeldRows(
  panel: UiPanel,
  controls: UiPanelControl[],
  findings: PanelContractFinding[],

  tooltipEmptyNote?: string,
): WeldRow[] {
  const derived = panel.derived;
  const rows: WeldRow[] = [];

  rows.push(
    makeRow(
      "window",
      "window",
      derived?.window_name ?? panel.token,
      findings.filter(
        (f) =>
          f.code === "panel.token_collision" ||
          f.code === "panel.control_collision",
      ),
      { file: derived?.gui_path ?? "" },
    ),
  );

  if ((panel.visible_trigger ?? "").trim()) {
    rows.push(
      makeRow(
        "panel-visible",
        "panelVisible",
        "visible",
        findings.filter(
          (f) =>
            f.code === "panel.unknown_scripted" && f.detail.field === "visible",
        ),
        { file: derived?.txt_path ?? "" },
      ),
    );
  }

  for (const ctrl of flattenControls(controls)) {
    const short = ctrl.name;
    const dc = derived?.controls?.[short];

    if ((ctrl.visible_trigger ?? "").trim()) {
      const key = dc?.visible_trigger_key;
      rows.push(
        makeRow(
          `visible:${short}`,
          "visible",
          key ?? short,
          key
            ? findings.filter(
                (f) =>
                  f.code === "panel.unknown_scripted" &&
                  f.detail.field === `triggers.${key}`,
              )
            : [],
          {
            controlName: short,
            file: derived?.txt_path ?? "",
            pendingSave: !key,
          },
        ),
      );
    }

    if (ctrl.kind === "text" || ctrl.kind === "button") {
      const key = dc?.text_loc_key;
      const blank = key
        ? findings.filter(
            (f) => f.code === "panel.loc_blank" && f.detail.loc_key === key,
          )
        : [];
      rows.push(
        makeRow(`loc:${short}`, "loc", key ?? short, blank, {
          controlName: short,
          fixableLocKey: blank.length > 0 ? (key ?? null) : null,
          pendingSave: !key,
        }),
      );
    }

    if (ctrl.kind === "button") {
      const clickKey = dc?.click_key;
      rows.push(
        makeRow(
          `click:${short}`,
          "click",
          clickKey ?? short,
          clickKey
            ? findings.filter(
                (f) =>
                  f.code === "panel.unknown_scripted" &&
                  f.detail.field === `effects.${clickKey}`,
              )
            : [],
          {
            controlName: short,
            file: derived?.txt_path ?? "",
            pendingSave: !clickKey,
          },
        ),
      );

      if ((ctrl.click_enabled_trigger ?? "").trim()) {
        const enabledKey = dc?.click_enabled_key;
        rows.push(
          makeRow(
            `click-enabled:${short}`,
            "clickEnabled",
            enabledKey ?? short,
            enabledKey
              ? findings.filter(
                  (f) =>
                    f.code === "panel.unknown_scripted" &&
                    f.detail.field === `triggers.${enabledKey}`,
                )
              : [],
            {
              controlName: short,
              file: derived?.txt_path ?? "",
              pendingSave: !enabledKey,
            },
          ),
        );
      }

      const tooltipPresent =
        (ctrl.tooltip ?? "").trim() !== "" ||
        Object.values(ctrl.tooltip_locs ?? {}).some(
          (v) => (v ?? "").trim() !== "",
        );

      const ttKey = dc?.tooltip_loc_key;
      const ttBlank = ttKey
        ? findings.filter(
            (f) => f.code === "panel.loc_blank" && f.detail.loc_key === ttKey,
          )
        : [];
      rows.push(
        makeRow(`tooltip:${short}`, "tooltip", ttKey ?? short, ttBlank, {
          controlName: short,
          fixableLocKey: ttBlank.length > 0 ? (ttKey ?? null) : null,
          pendingSave: tooltipPresent ? !dc : false,
          infoNote: tooltipPresent ? undefined : tooltipEmptyNote,
        }),
      );
    }

    if (
      ctrl.kind === "icon" ||
      ctrl.kind === "progressbar" ||
      ctrl.kind === "button"
    ) {
      rows.push(
        makeRow(
          `sprite:${short}`,
          "sprite",
          dc?.control_name ?? short,
          findings.filter(
            (f) =>
              f.code === "panel.sprite_empty" && f.detail.control === short,
          ),
          {
            controlName: short,
            file: derived?.gui_path ?? "",
            pendingSave: !dc,
          },
        ),
      );
    }
  }

  if (Object.keys(panel.raw_extras ?? {}).length > 0) {
    rows.push(
      makeRow(
        "raw",
        "raw",
        "raw_extras",
        findings.filter(
          (f) =>
            f.code === "panel.raw_dirty_invalid" ||
            (f.code === "panel.unknown_scripted" &&
              typeof f.detail.field === "string" &&
              (f.detail.field as string).startsWith("raw_extras.")),
        ),
        { file: derived?.txt_path ?? "" },
      ),
    );
  }

  return rows;
}

export function summarizeRows(rows: WeldRow[]): {
  welded: number;
  total: number;
  pending: number;
} {
  const pending = rows.filter((r) => r.worst !== null).length;
  return { welded: rows.length - pending, total: rows.length, pending };
}

export function fillBlankTextByName(
  controls: UiPanelControl[],
  short: string,
  placeholder: string,
): UiPanelControl[] {
  const visit = (list: UiPanelControl[]): UiPanelControl[] =>
    list.map((c) => {
      if ((c.kind === "text" || c.kind === "button") && c.name === short)
        return { ...c, text: placeholder };
      if (c.kind === "container" && c.children)
        return { ...c, children: visit(c.children) };
      return c;
    });
  return visit(controls);
}

export function fillBlankTooltipByName(
  controls: UiPanelControl[],
  short: string,
  placeholder: string,
): UiPanelControl[] {
  const visit = (list: UiPanelControl[]): UiPanelControl[] =>
    list.map((c) => {
      if (c.kind === "button" && c.name === short)
        return { ...c, tooltip: placeholder };
      if (c.kind === "container" && c.children)
        return { ...c, children: visit(c.children) };
      return c;
    });
  return visit(controls);
}
