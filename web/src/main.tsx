import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "@fontsource-variable/geist";
import "@fontsource-variable/geist-mono";
import "@/app/globals.css";
import "@/desktop/desktop.css";
import { App } from "@/desktop/App";
import { installProgramWindow } from "@/desktop/programWindow";

installProgramWindow();
createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
