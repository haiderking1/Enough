// PORT STATUS: active
// Electron preload — exposes the IPC bridge to the renderer (window.hollowDesktop).
// Loaded with contextIsolation: true; the renderer never touches Node directly.

const { contextBridge, ipcRenderer, webFrame } = require("electron");

const bridge = {
  // Agent ↔ renderer dispatch channel.
  dispatch: (cmd) => ipcRenderer.invoke("hollow:dispatch", cmd),
  // Agent → renderer event stream. cb receives BackendMessage-shaped events.
  onEvent: (cb) => {
    const handler = (_event, e) => cb(e);
    ipcRenderer.on("hollow:event", handler);
    return () => ipcRenderer.removeAllListeners("hollow:event");
  },
  isElectron: true,
  // Frameless title-bar window controls.
  minimize: () => ipcRenderer.send("window-minimize"),
  maximize: () => ipcRenderer.send("window-maximize"),
  close: () => ipcRenderer.send("window-close"),
  setZoom: (factor) => webFrame.setZoomFactor(factor),
};

contextBridge.exposeInMainWorld("hollowDesktop", bridge);
