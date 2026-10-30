"use client";

import { create } from "zustand";
import { ApiError, countriesApi } from "@/lib/api";

interface GameViewCountry {
  projectId: number | null;

  tag: string | null;
}

export const useGameViewCountry = create<GameViewCountry>(() => ({
  projectId: null,
  tag: null,
}));

export function speakFor(
  projectId: number,
  tag: string | null | undefined,
): void {
  const next = tag?.trim().toUpperCase() || null;
  const now = useGameViewCountry.getState();
  if (now.projectId !== projectId || now.tag !== next)
    useGameViewCountry.setState({ projectId, tag: next });
}

export function chosenCountry(projectId: number): string | null {
  const now = useGameViewCountry.getState();
  return now.projectId === projectId ? now.tag : null;
}

type TakenIn = (projectId: number, tag: string) => void;
const listeners = new Set<TakenIn>();

export function onCountryTakenIn(listener: TakenIn): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

const asked = new Map<string, Promise<boolean>>();

export function takeInCountry(
  projectId: number,
  tag: string,
): Promise<boolean> {
  const country = tag.trim().toUpperCase();
  const key = `${projectId}:${country}`;
  let answer = asked.get(key);
  if (!answer) {
    answer = countriesApi.takeIn(projectId, country).then(
      ({ created }) => {
        if (created)
          for (const listener of listeners) listener(projectId, country);
        return true;
      },
      (err: unknown) => {
        if (!(err instanceof ApiError && err.status === 422)) asked.delete(key);
        return false;
      },
    );
    asked.set(key, answer);
  }
  return answer;
}

export function noteEdited(projectId: number): void {
  const tag = chosenCountry(projectId);
  if (tag) void takeInCountry(projectId, tag);
}

export function resetCountryTakeIn(): void {
  asked.clear();
  listeners.clear();
  useGameViewCountry.setState({ projectId: null, tag: null });
}
