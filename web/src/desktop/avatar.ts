import { create } from "zustand";

export const PET_AVATARS = [
  "field_engineer_dog",
  "infantry_rabbit",
  "logistics_panda",
  "air_ace_fox",
  "tank_commander_bear",
  "codebreaker_owl",
  "admiral_penguin",
  "medic_deer",
] as const;

export type PetAvatar = (typeof PET_AVATARS)[number];

export type AvatarSelection =
  | { kind: "pet"; id: PetAvatar }
  | { kind: "custom"; dataUrl: string };

const STORAGE_KEY = "hagane-works-avatar-v1";
const DEFAULT_AVATAR: AvatarSelection = {
  kind: "pet",
  id: "field_engineer_dog",
};
const petAvatarSet = new Set<string>(PET_AVATARS);

export function petAvatarSource(id: PetAvatar): string {
  return `/avatars/pets/${id}.png`;
}

export function avatarSource(selection: AvatarSelection): string {
  return selection.kind === "pet"
    ? petAvatarSource(selection.id)
    : selection.dataUrl;
}

export function parseAvatarSelection(raw: string | null): AvatarSelection {
  if (!raw) return DEFAULT_AVATAR;
  try {
    const value = JSON.parse(raw) as Partial<AvatarSelection>;
    if (
      value.kind === "pet" &&
      typeof value.id === "string" &&
      petAvatarSet.has(value.id)
    ) {
      return { kind: "pet", id: value.id as PetAvatar };
    }
    if (
      value.kind === "custom" &&
      typeof value.dataUrl === "string" &&
      /^data:image\/(?:png|jpeg|webp);base64,/i.test(value.dataUrl)
    ) {
      return { kind: "custom", dataUrl: value.dataUrl };
    }
  } catch {
    return DEFAULT_AVATAR;
  }
  return DEFAULT_AVATAR;
}

function loadAvatar(): AvatarSelection {
  if (typeof window === "undefined") return DEFAULT_AVATAR;
  try {
    return parseAvatarSelection(window.localStorage.getItem(STORAGE_KEY));
  } catch {
    return DEFAULT_AVATAR;
  }
}

function saveAvatar(selection: AvatarSelection): void {
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(selection));
}

interface AvatarStore {
  selection: AvatarSelection;
  choosePet: (id: PetAvatar) => void;
  chooseCustom: (dataUrl: string) => void;
}

export const useAvatarStore = create<AvatarStore>((set) => ({
  selection: loadAvatar(),
  choosePet: (id) => {
    const selection: AvatarSelection = { kind: "pet", id };
    saveAvatar(selection);
    set({ selection });
  },
  chooseCustom: (dataUrl) => {
    const selection: AvatarSelection = { kind: "custom", dataUrl };
    saveAvatar(selection);
    set({ selection });
  },
}));

export async function imageFileToAvatarDataUrl(file: File): Promise<string> {
  const objectUrl = URL.createObjectURL(file);
  try {
    const image = await loadImage(objectUrl);
    const sourceSide = Math.min(image.naturalWidth, image.naturalHeight);
    if (sourceSide < 1) throw new Error("invalid-image");
    const outputSide = Math.min(512, sourceSide);
    const canvas = document.createElement("canvas");
    canvas.width = outputSide;
    canvas.height = outputSide;
    const context = canvas.getContext("2d");
    if (!context) throw new Error("canvas-unavailable");
    const sourceX = (image.naturalWidth - sourceSide) / 2;
    const sourceY = (image.naturalHeight - sourceSide) / 2;
    context.drawImage(
      image,
      sourceX,
      sourceY,
      sourceSide,
      sourceSide,
      0,
      0,
      outputSide,
      outputSide,
    );
    return canvas.toDataURL("image/webp", 0.86);
  } finally {
    URL.revokeObjectURL(objectUrl);
  }
}

function loadImage(source: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error("invalid-image"));
    image.src = source;
  });
}
