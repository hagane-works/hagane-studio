import { ApiError } from "@/lib/api/_client";
import type { Character } from "@/types";

export function characterTagOwners(
  characters: readonly Pick<Character, "id" | "tag">[],
): Map<string, number> {
  const owners = new Map<string, number>();
  for (const character of characters) {
    const tag = (character.tag || "").trim();
    if (tag) owners.set(tag, character.id);
  }
  return owners;
}

export function characterTagTaken(
  owners: Map<string, number>,
  tag: string,
  selfId: number,
): boolean {
  const owner = owners.get(tag.trim());
  return owner !== undefined && owner !== selfId;
}

export function isCharacterTagConflict(err: unknown): boolean {
  return err instanceof ApiError && err.status === 409;
}
