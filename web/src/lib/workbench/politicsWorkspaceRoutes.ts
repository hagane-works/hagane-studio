export const POLITICS_WORKSPACE_ROUTES = {
  countryPolitics: "country-politics",
  ideologies: "ideologies",
  countryProfile: "country-profile",
} as const;

export type PoliticsWorkspaceRoute =
  (typeof POLITICS_WORKSPACE_ROUTES)[keyof typeof POLITICS_WORKSPACE_ROUTES];

export function politicsWorkspaceHref(
  projectId: number,
  route: PoliticsWorkspaceRoute,
): string {
  return `/project/${projectId}/${route}`;
}

export function isPoliticsWorkspaceRoute(route: string): boolean {
  return (
    route === POLITICS_WORKSPACE_ROUTES.countryPolitics ||
    route === POLITICS_WORKSPACE_ROUTES.ideologies
  );
}
