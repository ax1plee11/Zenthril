/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE?: string;
  readonly VITE_TENOR_KEY?: string;
  readonly VITE_GIPHY_KEY?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
