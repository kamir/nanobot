import { app, BrowserWindow, shell } from 'electron';
import * as path from 'path';
import { SidecarManager } from './sidecar';
import { RemoteClient } from './remote-client';
import { registerIpcHandlers } from './ipc-handlers';
import { resolveMode, loadElectronConfig, saveElectronConfig, AppMode } from './mode-resolver';

let mainWindow: BrowserWindow | null = null;
const sidecar = new SidecarManager();
const remoteClient = new RemoteClient();

function createWindow(): BrowserWindow {
  const cfg = loadElectronConfig();
  const win = new BrowserWindow({
    width: cfg.windowState.width,
    height: cfg.windowState.height,
    x: cfg.windowState.x,
    y: cfg.windowState.y,
    title: 'GoMikroBot',
    webPreferences: {
      preload: path.join(__dirname, '..', 'preload', 'index.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  });

  // Save window state on close
  win.on('close', () => {
    const bounds = win.getBounds();
    const cfg = loadElectronConfig();
    cfg.windowState = { width: bounds.width, height: bounds.height, x: bounds.x, y: bounds.y };
    saveElectronConfig(cfg);
  });

  // Open external links in default browser
  win.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url);
    return { action: 'deny' };
  });

  return win;
}

async function loadApp(win: BrowserWindow): Promise<void> {
  // In dev, load from Vite dev server; in prod, load built files
  const isDev = !app.isPackaged;
  if (isDev) {
    await win.loadURL('http://localhost:5173');
  } else {
    await win.loadFile(path.join(__dirname, '..', 'renderer', 'index.html'));
  }
}

app.whenReady().then(async () => {
  registerIpcHandlers(sidecar, remoteClient);

  // Load saved remote connections
  const cfg = loadElectronConfig();
  for (const conn of cfg.remoteConnections) {
    remoteClient.addConnection(conn);
  }

  mainWindow = createWindow();

  // Broadcast sidecar status to renderer
  sidecar.setStatusCallback((status) => {
    mainWindow?.webContents.send('sidecar:statusChanged', status);
  });

  await loadApp(mainWindow);

  // Resolve mode and auto-start sidecar for local modes
  const mode = resolveMode(process.argv);
  if (mode === 'full' || mode === 'standalone') {
    sidecar.start(mode, cfg.sidecarPath || undefined).catch((err) => {
      console.error('Sidecar start failed:', err);
    });
  }
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('activate', () => {
  if (BrowserWindow.getAllWindows().length === 0) {
    mainWindow = createWindow();
    loadApp(mainWindow);
  }
});

app.on('before-quit', async () => {
  await sidecar.stop();
});
