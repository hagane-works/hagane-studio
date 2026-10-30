"use client";

import { createContext, useContext } from "react";

export const AddCountryDialogContext = createContext<(() => void) | null>(null);

export function useOpenAddCountryDialog() {
  const open = useContext(AddCountryDialogContext);
  if (!open)
    throw new Error("AddCountryDialogContext is required inside a project");
  return open;
}
