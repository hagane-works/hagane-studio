import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath } from "node:url";

const devApi = process.env.HAGANEWORKS_API ?? "http://127.0.0.1:7330";

let devSession = "";
async function refreshDevSession() {
  try {
    const res = await fetch(`${devApi}/`);
    const cookie = res.headers.get("set-cookie");
    if (cookie) devSession = cookie.split(";")[0];
  } catch {}
}

export default defineConfig(({ command }) => {
  if (command === "serve") void refreshDevSession();
  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) },
    },
    server: {
      proxy: {
        "/api": {
          target: devApi,
          changeOrigin: true,
          configure(proxy) {
            proxy.on("proxyReq", (req) => {
              req.setHeader("origin", devApi);
              if (devSession) req.setHeader("cookie", devSession);
            });
            proxy.on("proxyRes", (res) => {
              if (res.statusCode === 401) void refreshDevSession();
            });
          },
        },
      },
    },
    build: {
      outDir: "dist",
      emptyOutDir: true,
      chunkSizeWarningLimit: 4096,
      sourcemap: false,
    },
    test: {
      projects: [
        {
          extends: true,
          test: {
            name: "node",
            environment: "node",
            setupFiles: ["src/test/setup.ts"],
            include: ["src/**/*.test.ts"],
            exclude: [
              "src/components/workbench/import-wizard/localScanResponsiveness.contract.test.ts",
              "src/lib/api/_client.warnings.test.ts",
              "src/lib/chunkLoadRecovery.test.ts",
            ],
          },
        },
        {
          extends: true,
          test: {
            name: "browser",
            environment: "jsdom",
            setupFiles: ["src/test/setup.ts"],
            include: [
              "src/**/*.test.tsx",
              "src/components/workbench/import-wizard/localScanResponsiveness.contract.test.ts",
              "src/lib/api/_client.warnings.test.ts",
              "src/lib/chunkLoadRecovery.test.ts",
            ],
          },
        },
      ],
    },
  };
});
