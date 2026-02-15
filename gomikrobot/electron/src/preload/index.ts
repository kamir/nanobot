import { contextBridge, ipcRenderer } from 'electron';

contextBridge.exposeInMainWorld('electronAPI', {
  // Sidecar
  sidecar: {
    getStatus: () => ipcRenderer.invoke('sidecar:status'),
    getLogs: () => ipcRenderer.invoke('sidecar:logs'),
    start: (mode: string) => ipcRenderer.invoke('sidecar:start', mode),
    stop: () => ipcRenderer.invoke('sidecar:stop'),
    onStatusChanged: (cb: (status: string) => void) => {
      ipcRenderer.on('sidecar:statusChanged', (_event, status) => cb(status));
    },
  },

  // Mode
  mode: {
    get: () => ipcRenderer.invoke('mode:get'),
    set: (mode: string) => ipcRenderer.invoke('mode:set', mode),
  },

  // Config
  config: {
    get: () => ipcRenderer.invoke('config:get'),
    save: (partial: Record<string, any>) => ipcRenderer.invoke('config:save', partial),
  },

  // Remote
  remote: {
    list: () => ipcRenderer.invoke('remote:list'),
    add: (conn: any) => ipcRenderer.invoke('remote:add', conn),
    remove: (id: string) => ipcRenderer.invoke('remote:remove', id),
    verify: (conn: any) => ipcRenderer.invoke('remote:verify', conn),
    setActive: (id: string) => ipcRenderer.invoke('remote:setActive', id),
    getActive: () => ipcRenderer.invoke('remote:getActive'),
  },
});
