import { ipcMain } from 'electron';
import { SidecarManager } from './sidecar';
import { RemoteClient, RemoteConnection } from './remote-client';
import {
  AppMode,
  loadElectronConfig,
  saveElectronConfig,
} from './mode-resolver';

export function registerIpcHandlers(
  sidecar: SidecarManager,
  remoteClient: RemoteClient,
): void {
  // Sidecar
  ipcMain.handle('sidecar:status', () => sidecar.getStatus());
  ipcMain.handle('sidecar:logs', () => sidecar.getLogs());
  ipcMain.handle('sidecar:start', async (_event, mode: AppMode) => {
    const cfg = loadElectronConfig();
    await sidecar.start(mode, cfg.sidecarPath || undefined);
    return sidecar.getStatus();
  });
  ipcMain.handle('sidecar:stop', async () => {
    await sidecar.stop();
    return sidecar.getStatus();
  });

  // Mode
  ipcMain.handle('mode:get', () => {
    const cfg = loadElectronConfig();
    return cfg.mode;
  });
  ipcMain.handle('mode:set', (_event, mode: AppMode) => {
    const cfg = loadElectronConfig();
    cfg.mode = mode;
    saveElectronConfig(cfg);
    return mode;
  });

  // Config
  ipcMain.handle('config:get', () => loadElectronConfig());
  ipcMain.handle('config:save', (_event, partial: Record<string, any>) => {
    const cfg = { ...loadElectronConfig(), ...partial };
    saveElectronConfig(cfg);
    return cfg;
  });

  // Remote connections
  ipcMain.handle('remote:list', () => {
    const cfg = loadElectronConfig();
    return cfg.remoteConnections;
  });
  ipcMain.handle('remote:add', (_event, conn: RemoteConnection) => {
    const cfg = loadElectronConfig();
    cfg.remoteConnections = cfg.remoteConnections.filter((c) => c.id !== conn.id);
    cfg.remoteConnections.push(conn);
    saveElectronConfig(cfg);
    remoteClient.addConnection(conn);
    return cfg.remoteConnections;
  });
  ipcMain.handle('remote:remove', (_event, id: string) => {
    const cfg = loadElectronConfig();
    cfg.remoteConnections = cfg.remoteConnections.filter((c) => c.id !== id);
    saveElectronConfig(cfg);
    remoteClient.removeConnection(id);
    return cfg.remoteConnections;
  });
  ipcMain.handle('remote:verify', async (_event, conn: RemoteConnection) => {
    return remoteClient.verify(conn);
  });
  ipcMain.handle('remote:setActive', (_event, id: string) => {
    remoteClient.setActive(id);
    return true;
  });
  ipcMain.handle('remote:getActive', () => {
    return remoteClient.getActive();
  });
}
