import { lazy, type ComponentType, type LazyExoticComponent } from "react";

type Page = LazyExoticComponent<ComponentType>;

export const projectPages: Record<string, Page> = {
  start: lazy(() => import("@/app/[locale]/project/[id]/start/page")),
  characters: lazy(() => import("@/app/[locale]/project/[id]/characters/page")),
  "country-politics": lazy(
    () => import("@/app/[locale]/project/[id]/country-politics/page"),
  ),
  "country-profile": lazy(
    () => import("@/app/[locale]/project/[id]/country-profile/page"),
  ),
  ideas: lazy(() => import("@/app/[locale]/project/[id]/ideas/page")),
  "focus-tree": lazy(
    () => import("@/app/[locale]/project/[id]/focus-tree/page"),
  ),
  events: lazy(() => import("@/app/[locale]/project/[id]/events/page")),
  decisions: lazy(() => import("@/app/[locale]/project/[id]/decisions/page")),
  technology: lazy(() => import("@/app/[locale]/project/[id]/technology/page")),
  states: lazy(() => import("@/app/[locale]/project/[id]/states/page")),
  "world-map": lazy(() => import("@/app/[locale]/project/[id]/world-map/page")),
  bookmarks: lazy(() => import("@/app/[locale]/project/[id]/bookmarks/page")),
  "history-files": lazy(
    () => import("@/app/[locale]/project/[id]/history-files/page"),
  ),
  ideologies: lazy(() => import("@/app/[locale]/project/[id]/ideologies/page")),
  "super-events": lazy(
    () => import("@/app/[locale]/project/[id]/super-events/page"),
  ),
  "autonomy-states": lazy(
    () => import("@/app/[locale]/project/[id]/autonomy-states/page"),
  ),
  wargoals: lazy(() => import("@/app/[locale]/project/[id]/wargoals/page")),
  "dynamic-modifiers": lazy(
    () => import("@/app/[locale]/project/[id]/dynamic-modifiers/page"),
  ),
  "modifier-definitions": lazy(
    () => import("@/app/[locale]/project/[id]/modifier-definitions/page"),
  ),
  "scripted-snippets": lazy(
    () => import("@/app/[locale]/project/[id]/scripted-snippets/page"),
  ),
  "scripted-localisations": lazy(
    () => import("@/app/[locale]/project/[id]/scripted-localisations/page"),
  ),
  equipment: lazy(() => import("@/app/[locale]/project/[id]/equipment/page")),
  "sub-units": lazy(() => import("@/app/[locale]/project/[id]/sub-units/page")),
  "division-templates": lazy(
    () => import("@/app/[locale]/project/[id]/division-templates/page"),
  ),
  deployments: lazy(
    () => import("@/app/[locale]/project/[id]/deployments/page"),
  ),
  "oob-units": lazy(() => import("@/app/[locale]/project/[id]/oob-units/page")),
  doctrine: lazy(() => import("@/app/[locale]/project/[id]/doctrine/page")),
  mio: lazy(() => import("@/app/[locale]/project/[id]/mio/page")),
  intelligence: lazy(
    () => import("@/app/[locale]/project/[id]/intelligence/page"),
  ),
  "special-projects": lazy(
    () => import("@/app/[locale]/project/[id]/special-projects/page"),
  ),
  "ai-strategy-plans": lazy(
    () => import("@/app/[locale]/project/[id]/ai-strategy-plans/page"),
  ),
  "ai-templates": lazy(
    () => import("@/app/[locale]/project/[id]/ai-templates/page"),
  ),
  "ui-panels": lazy(() => import("@/app/[locale]/project/[id]/ui-panels/page")),
  "image-batch": lazy(
    () => import("@/app/[locale]/project/[id]/image-batch/page"),
  ),
  export: lazy(() => import("@/app/[locale]/project/[id]/export/page")),
  "export/repair": lazy(
    () => import("@/app/[locale]/project/[id]/export/repair/page"),
  ),
};
