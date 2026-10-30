"use client";

import { createContext, useContext } from "react";

export const FocusInGamePreviewContext = createContext(false);

export const useFocusInGamePreview = () =>
  useContext(FocusInGamePreviewContext);
