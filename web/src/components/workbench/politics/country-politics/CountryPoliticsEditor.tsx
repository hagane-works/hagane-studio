"use client";

import { useMemo } from "react";
import { AlertTriangle, CheckCircle2, Landmark, UserRound } from "lucide-react";
import { useTranslations } from "use-intl";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import type { CountryRow } from "@/lib/api";
import type { IdeologyOption } from "@/lib/ideology/registry";
import type { Character } from "@/types";

import {
  classifyCountryLeaders,
  isCurrentLeaderRulingMismatch,
  popularityPieSegments,
  subtypeRoot,
  validatePopularities,
  type CountryPoliticsDraft,
} from "./model";

export type CountryPoliticsSaving = "politics" | `leader-${number}` | null;

export interface CountryPoliticsEditorProps {
  country: Pick<CountryRow, "tag" | "name" | "name_en">;
  roots: readonly IdeologyOption[];
  subtypes: readonly IdeologyOption[];

  savedRulingParty: string;
  draft: CountryPoliticsDraft;
  onEdit: (
    update: (draft: CountryPoliticsDraft) => CountryPoliticsDraft,
  ) => void;
  leaders: readonly Character[];

  leaderDrafts: Readonly<Record<number, string>>;
  onEditLeaders: (
    update: (drafts: Record<number, string>) => Record<number, string>,
  ) => void;
  saving: CountryPoliticsSaving;
  onSavePolitics: () => void;
  onSaveLeader: (leader: Character) => void;

  onOpenIdeologies?: () => void;
  status: "" | "saved";

  error: string | null;
}

export function CountryPoliticsEditor({
  country,
  roots,
  subtypes,
  savedRulingParty,
  draft,
  onEdit,
  leaders,
  leaderDrafts,
  onEditLeaders,
  saving,
  onSavePolitics,
  onSaveLeader,
  onOpenIdeologies,
  status,
  error,
}: CountryPoliticsEditorProps) {
  const t = useTranslations("countrySettingsDialog.workbench.politics");

  const popularity = useMemo(
    () => validatePopularities(draft.popularities),
    [draft.popularities],
  );
  const pieSegments = useMemo(
    () =>
      popularityPieSegments(
        draft.popularities,
        roots.map((root) => root.value),
      ),
    [draft.popularities, roots],
  );
  const dominantSegment = useMemo(
    () =>
      pieSegments.reduce(
        (best, segment) => (segment.value > best.value ? segment : best),
        pieSegments[0] ?? { id: "", value: 0, share: 0, offset: 0 },
      ),
    [pieSegments],
  );
  const dominantRoot = roots.find((root) => root.value === dominantSegment.id);
  const leaderClassification = useMemo(
    () => classifyCountryLeaders(leaders, savedRulingParty, subtypes),
    [leaders, savedRulingParty, subtypes],
  );
  const updateDraft = <K extends keyof CountryPoliticsDraft>(
    key: K,
    value: CountryPoliticsDraft[K],
  ) => onEdit((current) => ({ ...current, [key]: value }));

  return (
    <div className="space-y-4" data-testid="country-politics-workbench">
      <section className="ui-panel flex items-center justify-between gap-4 p-4">
        <div className="flex min-w-0 items-center gap-3">
          <Landmark
            className="size-5 shrink-0 text-primary"
            aria-hidden="true"
          />
          <div className="min-w-0">
            <h2 className="text-lg font-semibold tracking-tight">
              {t("title")}
            </h2>
            <p className="text-xs text-muted-foreground">
              {t("context", {
                tag: country.tag,
                name: country.name_en || country.name,
              })}
            </p>
          </div>
        </div>
        <Badge variant="outline">{t("canonicalHistory")}</Badge>
      </section>

      <section className="ui-panel p-5">
        <div className="mb-4 flex items-start justify-between gap-3">
          <div>
            <h3 className="text-base font-semibold tracking-tight">
              {t("openingPolitics")}
            </h3>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("openingPoliticsHint")}
            </p>
          </div>
          <span
            className={
              popularity.valid
                ? "text-xs text-success"
                : "text-xs text-destructive"
            }
          >
            {t("supportTotal", { total: popularity.total })}
          </span>
        </div>

        <div className="grid gap-5 lg:grid-cols-[15rem_minmax(0,1fr)] lg:items-center">
          <div className="rounded-md border border-border bg-muted/20 p-4">
            <div className="relative mx-auto size-44">
              <svg
                viewBox="0 0 100 100"
                role="img"
                aria-label={t("supportChartLabel")}
                className="size-full -rotate-90"
              >
                <circle
                  cx="50"
                  cy="50"
                  r="38"
                  pathLength="100"
                  fill="none"
                  stroke="var(--border)"
                  strokeWidth="18"
                />
                {pieSegments.map((segment, index) => (
                  <circle
                    key={segment.id}
                    cx="50"
                    cy="50"
                    r="38"
                    pathLength="100"
                    fill="none"
                    stroke={`var(--chart-${(index % 5) + 1})`}
                    strokeWidth="18"
                    strokeDasharray={`${segment.share} ${100 - segment.share}`}
                    strokeDashoffset={-segment.offset}
                  />
                ))}
              </svg>
              <div className="pointer-events-none absolute inset-0 grid place-content-center text-center">
                <strong className="text-xl font-semibold tabular-nums">
                  {dominantSegment.value}%
                </strong>
                <span className="mt-1 max-w-24 text-micro text-muted-foreground">
                  {dominantRoot?.label ?? t("supportChartEmpty")}
                </span>
              </div>
            </div>
            <p className="mt-3 text-center text-xs text-muted-foreground">
              {t("supportChartHint")}
            </p>
          </div>

          <div className="grid grid-cols-2 gap-3 xl:grid-cols-3">
            {roots.map((root, index) => (
              <label key={root.value} className="ui-card block p-3">
                <span className="mb-2 flex items-center justify-between gap-2 text-xs">
                  <span className="flex min-w-0 items-center gap-2">
                    <span
                      className="size-2.5 shrink-0 rounded-full"
                      style={{
                        backgroundColor: `var(--chart-${(index % 5) + 1})`,
                      }}
                      aria-hidden="true"
                    />
                    <span className="truncate font-medium">{root.label}</span>
                  </span>
                  <code className="text-micro text-muted-foreground">
                    {root.value}
                  </code>
                </span>
                <Input
                  type="number"
                  min={0}
                  max={100}
                  step={1}
                  value={draft.popularities[root.value] ?? 0}
                  aria-label={t("supportAria", { root: root.label })}
                  onChange={(event) =>
                    updateDraft("popularities", {
                      ...draft.popularities,
                      [root.value]: Number(event.target.value),
                    })
                  }
                />
              </label>
            ))}
          </div>
        </div>

        {!popularity.valid && (
          <p
            role="alert"
            className="mt-3 flex items-center gap-2 text-xs text-destructive"
          >
            <AlertTriangle className="size-4" aria-hidden="true" />
            {popularity.reason === "range"
              ? t("supportRangeError")
              : t("supportTotalError")}
          </p>
        )}

        <div className="mt-5 grid gap-4 border-t border-border pt-4 md:grid-cols-2">
          <div className="space-y-1">
            <Label htmlFor="country-politics-ruling">{t("rulingParty")}</Label>
            <Select
              id="country-politics-ruling"
              value={draft.rulingParty}
              onChange={(event) =>
                updateDraft("rulingParty", event.target.value)
              }
            >
              <option value="">{t("chooseRoot")}</option>
              {roots.map((root) => (
                <option key={root.value} value={root.value}>
                  {root.label}
                </option>
              ))}
            </Select>
          </div>
          <div className="space-y-1">
            <Label htmlFor="country-politics-election-mode">
              {t("elections")}
            </Label>
            <Select
              id="country-politics-election-mode"
              value={
                draft.electionsAllowed === null
                  ? "inherit"
                  : String(draft.electionsAllowed)
              }
              onChange={(event) =>
                updateDraft(
                  "electionsAllowed",
                  event.target.value === "inherit"
                    ? null
                    : event.target.value === "true",
                )
              }
            >
              <option value="inherit">{t("electionsInherit")}</option>
              <option value="true">{t("electionsAllowed")}</option>
              <option value="false">{t("electionsBlocked")}</option>
            </Select>
          </div>
          <div className="space-y-1">
            <Label htmlFor="country-politics-last-election">
              {t("lastElection")}
            </Label>
            <Input
              id="country-politics-last-election"
              value={draft.lastElection}
              placeholder="1932.11.8"
              onChange={(event) =>
                updateDraft("lastElection", event.target.value)
              }
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="country-politics-frequency">
              {t("electionFrequency")}
            </Label>
            <Input
              id="country-politics-frequency"
              type="number"
              min={0}
              step={1}
              value={draft.electionFrequency}
              onChange={(event) =>
                updateDraft("electionFrequency", Number(event.target.value))
              }
            />
          </div>
        </div>

        <div className="mt-4 flex items-center justify-between gap-3 border-t border-border pt-4">
          <p className="text-xs text-muted-foreground">
            {t("explicitSaveHint")}
          </p>
          <Button
            onClick={onSavePolitics}
            disabled={!popularity.valid || saving !== null}
          >
            {saving === "politics" ? t("saving") : t("savePolitics")}
          </Button>
        </div>
      </section>

      <section className="ui-panel p-5">
        <div className="mb-4 flex items-start justify-between gap-3">
          <div>
            <h3 className="text-base font-semibold tracking-tight">
              {t("leaders")}
            </h3>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("leadersHint")}
            </p>
          </div>
          {onOpenIdeologies && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={onOpenIdeologies}
            >
              {t("openIdeologies")}
            </Button>
          )}
        </div>

        {leaders.length === 0 ? (
          <div className="ui-card p-4 text-sm text-muted-foreground">
            {t("noLeaders")}
          </div>
        ) : (
          <div className="space-y-3">
            {leaders.map((leader) => {
              const selectedSubtype =
                leaderDrafts[leader.id] ?? leader.ideology;
              const leaderRoot = subtypeRoot(selectedSubtype, subtypes);
              const isCurrentLeader =
                leaderClassification.roleByLeaderId[leader.id] === "current";
              const mismatch = isCurrentLeaderRulingMismatch(
                leaderClassification,
                leader.id,
                leaderRoot,
                draft.rulingParty,
              );
              const rulingSubtype = subtypes.find(
                (subtype) => subtype.group === draft.rulingParty,
              );
              return (
                <article key={leader.id} className="ui-card p-4">
                  <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(240px,.8fr)_auto] lg:items-end">
                    <div className="flex min-w-0 items-center gap-3">
                      <div className="flex size-10 shrink-0 items-center justify-center rounded-md border border-border bg-muted">
                        <UserRound
                          className="size-5 text-muted-foreground"
                          aria-hidden="true"
                        />
                      </div>
                      <div className="min-w-0">
                        <h4 className="truncate text-sm font-semibold">
                          {leader.name_en || leader.name}
                        </h4>
                        <p className="truncate text-xs text-muted-foreground">
                          <code>{leader.tag}</code>
                        </p>
                        <p className="text-micro text-muted-foreground">
                          {isCurrentLeader
                            ? t("currentLeader")
                            : t("candidateLeader")}
                        </p>
                      </div>
                    </div>
                    <div className="space-y-1">
                      <Label htmlFor={`leader-ideology-${leader.id}`}>
                        {t("leaderSubtype")}
                      </Label>
                      <Select
                        id={`leader-ideology-${leader.id}`}
                        value={selectedSubtype}
                        onChange={(event) => {
                          const value = event.target.value;
                          onEditLeaders((current) => ({
                            ...current,
                            [leader.id]: value,
                          }));
                        }}
                      >
                        <option value="">{t("chooseSubtype")}</option>
                        {subtypes.map((subtype) => (
                          <option key={subtype.value} value={subtype.value}>
                            {subtype.label}
                          </option>
                        ))}
                      </Select>
                    </div>
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      disabled={
                        saving !== null || selectedSubtype === leader.ideology
                      }
                      onClick={() => onSaveLeader(leader)}
                    >
                      {saving === `leader-${leader.id}`
                        ? t("saving")
                        : t("saveLeader")}
                    </Button>
                  </div>
                  <div
                    className={`mt-3 flex items-center gap-2 text-xs ${mismatch ? "text-warning" : "text-success"}`}
                  >
                    {mismatch ? (
                      <AlertTriangle className="size-4" aria-hidden="true" />
                    ) : (
                      <CheckCircle2 className="size-4" aria-hidden="true" />
                    )}
                    <span>
                      {!isCurrentLeader
                        ? t("candidateDifferentNormal")
                        : mismatch
                          ? t("leaderMismatch", { root: leaderRoot })
                          : t("leaderMatched")}
                    </span>
                    {mismatch && (
                      <span className="ml-auto flex flex-wrap justify-end gap-1">
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="h-7 text-xs"
                          onClick={() => updateDraft("rulingParty", leaderRoot)}
                        >
                          {t("prepareRulingRepair")}
                        </Button>
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="h-7 text-xs"
                          disabled={!rulingSubtype}
                          onClick={() => {
                            if (rulingSubtype) {
                              onEditLeaders((current) => ({
                                ...current,
                                [leader.id]: rulingSubtype.value,
                              }));
                            }
                          }}
                        >
                          {t("prepareLeaderRepair")}
                        </Button>
                      </span>
                    )}
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </section>

      {(status || error) && (
        <div
          role="status"
          aria-live="polite"
          className={`sticky bottom-0 rounded-md border p-3 text-sm ${
            error
              ? "border-destructive/30 bg-destructive/10 text-destructive"
              : "border-success/30 bg-success/10 text-success"
          }`}
        >
          {error ?? t("savedReadback")}
        </div>
      )}
    </div>
  );
}
