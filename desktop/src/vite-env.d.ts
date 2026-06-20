/// <reference types="vite/client" />

declare module "*.svg" {
  const src: string
  export default src
}

interface DirListing {
  path: string
  parent: string | null
  entries: { name: string; path: string }[]
  home: string
  error?: string
}

interface HollowBridge {
  isElectron: true
  dispatch: (cmd: Record<string, unknown>) => Promise<{ ok: boolean; data?: unknown; error?: string }>
  onEvent: (cb: (msg: unknown) => void) => () => void
  setZoom: (factor: number) => void
  pickDirectory: () => Promise<string | null>
  listDir: (path?: string) => Promise<DirListing>
}

interface Window {
  hollowDesktop?: HollowBridge
}
