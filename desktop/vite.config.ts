import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

// Config lives in desktop/ — root must be this directory, not the repo cwd.
const desktopRoot = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  root: desktopRoot,
  plugins: [
    react(),
    tailwindcss(),
  ],
  server: {
    port: 1420,
    strictPort: true,
  },
  base: './',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
});
