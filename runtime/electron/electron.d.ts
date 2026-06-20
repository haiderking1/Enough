// PORT STATUS: active
// Minimal Electron module declaration so hollow typechecks without a hard
// electron install. runtime/electron/main.ts imports { app, BrowserWindow, ipcMain }
// and preload.cjs is CommonJS (excluded from tsconfig). When electron is actually
// installed (dev/electron:dev), Bun resolves the real module at runtime.

declare module "electron" {
  export interface BrowserWindow {
    loadURL(url: string): void;
    loadFile(path: string): void;
    setMenu(menu: unknown): void;
    minimize(): void;
    unmaximize(): void;
    maximize(): void;
    isMaximized(): boolean;
    close(): void;
    readonly webContents: WebContents;
  }

  export interface WebContents {
    send(channel: string, payload: unknown): void;
    isDestroyed(): boolean;
  }

  export const app: {
    commandLine: { appendSwitch(switch_: string, value?: string): void };
    isPackaged: boolean;
    whenReady(): Promise<void>;
    on(event: string, listener: (...args: unknown[]) => void): void;
    quit(): void;
  };

  export const BrowserWindow: {
    fromWebContents(wc: WebContents): BrowserWindow | null;
    getAllWindows(): BrowserWindow[];
    new(opts?: unknown): BrowserWindow;
  };

  export interface IpcMainEvent {
    sender: WebContents;
  }

  export const ipcMain: {
    on(channel: string, listener: (event: IpcMainEvent, ...args: unknown[]) => void): void;
    handle(
      channel: string,
      handler: (event: IpcMainEvent, payload: any) => Promise<any> | any,
    ): void;
  };

  export const dialog: {
    showOpenDialog(
      browserWindow?: BrowserWindow,
      options?: { properties: string[] },
    ): Promise<{ canceled: boolean; filePaths: string[] }>;
  };

  export const contextBridge: {
    exposeInMainWorld(name: string, api: unknown): void;
  };

  export const ipcRenderer: {
    send(channel: string, ...args: unknown[]): void;
    invoke(channel: string, ...args: unknown[]): Promise<unknown>;
    on(channel: string, listener: (...args: unknown[]) => void): void;
    removeAllListeners(channel: string): void;
  };

  export const webFrame: {
    setZoomFactor(factor: number): void;
  };
}
