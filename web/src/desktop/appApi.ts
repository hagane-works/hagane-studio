import { request } from "@/lib/api/_client";

export interface GameInstall {
  dir: string;
  version: string;
}

export interface CwtoolsStatus {
  dir: string;

  modifiers: number;

  error: string;
}

export type CpuSetting = "1" | "50%" | "100%";

export interface AppStatus {
  version: string;

  settings: {
    projects_dir: string;
    game_dir: string;
    mod_dir?: string;
    cwtools_dir?: string;
    port?: number;
    cpu?: CpuSetting;
  };
  game: GameInstall | null;
  mod_dir: string;
  cwtools: CwtoolsStatus;

  port: number;

  configured_port: number;

  cpu: CpuSetting;
  cpu_cores: number;
  cores: number;
  ready: boolean;
}

export interface GameIndexStatus {
  state: "missing" | "building" | "ready" | "failed";
  game_version: string;

  step: string;
  done: number;
  total: number;
  error: string;

  built_at: string;
}

export interface FolderEntry {
  name: string;
  path: string;
}

export interface FolderListing {
  path: string;

  parent: string | null;
  entries: FolderEntry[];
}

export const appApi = {
  status: () => request<AppStatus>("/app/status"),
  updateSettings: (patch: Partial<AppStatus["settings"]>) =>
    request<AppStatus>("/app/settings", {
      method: "PUT",
      body: JSON.stringify(patch),
    }),
  folders: (path: string) =>
    request<FolderListing>(`/app/folders?path=${encodeURIComponent(path)}`),
  gameIndex: () => request<GameIndexStatus>("/app/game-index"),
  rebuildGameIndex: () =>
    request<GameIndexStatus>("/app/game-index/rebuild", { method: "POST" }),
  quit: () => request<{ ok: boolean }>("/app/quit", { method: "POST" }),
};
