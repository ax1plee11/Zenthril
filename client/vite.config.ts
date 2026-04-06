/// <reference types="vitest" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vitejs.dev/config/
export default defineConfig(async () => ({
  plugins: [react()],

  // Vitest configuration
  test: {
    environment: "node",
    globals: true,
    setupFiles: ["./src/crypto/test-setup.ts"],
  },

  // Vite options tailored for Tauri development and only applied in `tauri dev` or `tauri build`
  clearScreen: false,
  server: {
    port: 1420,
    strictPort: true,
    host: "0.0.0.0",
    watch: {
      ignored: ["**/src-tauri/**"],
    },
  },
}));
