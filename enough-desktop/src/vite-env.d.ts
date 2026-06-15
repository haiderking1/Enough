/// <reference types="vite/client" />

interface DirListing {
  path: string
  parent: string | null
  entries: { name: string; path: string }[]
  home: string
  error?: string
}

interface EnoughBridge {
  isElectron: true
  setZoom: (factor: number) => void
  pickDirectory: () => Promise<string | null>
  listDir: (path?: string) => Promise<DirListing>
}

interface Window {
  enough?: EnoughBridge
}
