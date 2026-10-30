import type { GuiDivisions } from "@/lib/guiRenderApi";
import { tagFromOob } from "@/lib/oob/oobDefaults";
import type { DivisionDeployment, DivisionTemplate } from "@/types";

export function countryDeployments(
  rows: readonly DivisionDeployment[] | undefined,
  tag: string,
): DivisionDeployment[] {
  const want = tag.toUpperCase();
  return (rows ?? []).filter(
    (row) =>
      (row.country_tag || tagFromOob(row.oob_file)).toUpperCase() === want,
  );
}

export function newDeploymentTemplate(
  templates: readonly DivisionTemplate[] | undefined,
  oobFile: string,
  divisions: GuiDivisions | undefined,
): string {
  return (
    templates?.find((template) => template.oob_file === oobFile)
      ?.template_name ||
    divisions?.templates[0]?.name ||
    ""
  );
}

export function deploymentLabel(
  row: Pick<DivisionDeployment, "count" | "division_template">,
  noTemplate: string,
): string {
  return `${row.count} × ${row.division_template || noTemplate}`;
}
