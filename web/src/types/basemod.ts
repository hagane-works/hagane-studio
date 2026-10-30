export interface BaseModRef {
  id: string;
  display_name: string;
  dependency_key: string;
  dependency_name: string;
}

export interface BaseModRefSearchResult {
  type: string;
  results: BaseModRef[];
}

export interface BaseModIdeologyGroup {
  value: string;
  label: string;
}

export interface BaseModIdeologySub {
  id: string;
  group: string;
  name: { en: string };
}

export interface BaseModIdeologyRegistry {
  groups: BaseModIdeologyGroup[];
  sub_ideologies: BaseModIdeologySub[];
}
