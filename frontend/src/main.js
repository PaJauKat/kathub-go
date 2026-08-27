// ==========================================
// KatHub - Frontend logic with Wails v2 bindings
// ==========================================

let currentKatPluginsState = null;
let currentLolConfigState = null;
let currentUpdateInfo = null;

// Safe Wails API wrapper
const GoApp = {
    getKatPluginsState: async () => {
        if (window.go && window.go.main && window.go.main.App) {
            return await window.go.main.App.GetKatPluginsState();
        }
        console.warn("Wails runtime not loaded, using fallback mockup");
        return {
            state: "Install",
            statusText: "Plugin not detected",
            statusColor: "#f08080",
            buttonText: "Install",
            buttonColor: "#e63946",
            mainButtonVisible: true,
            uninstallVisible: false
        };
    },
    handleKatPluginsMainAction: async () => {
        if (window.go && window.go.main && window.go.main.App) {
            return await window.go.main.App.HandleKatPluginsMainAction();
        }
        return true;
    },
    uninstallKatPlugins: async () => {
        if (window.go && window.go.main && window.go.main.App) {
            return await window.go.main.App.UninstallKatPlugins();
        }
        return true;
    },
    getLolConfigState: async () => {
        if (window.go && window.go.main && window.go.main.App) {
            return await window.go.main.App.GetLolConfigState();
        }
        return {
            state: "Unlocked",
            statusText: "Unlocked ❌",
            statusColor: "#f08080",
            buttonText: "Lock",
            canToggle: true
        };
    },
    toggleLolConfig: async (lock) => {
        if (window.go && window.go.main && window.go.main.App) {
            return await window.go.main.App.ToggleLolConfig(lock);
        }
        return true;
    },
    openAccountManager: async () => {
        if (window.go && window.go.main && window.go.main.App) {
            return await window.go.main.App.OpenAccountManager();
        }
    },
    installAccountManager: async () => {
        if (window.go && window.go.main && window.go.main.App) {
            return await window.go.main.App.InstallAccountManager();
        }
        return true;
    },
    checkForUpdates: async () => {
        if (window.go && window.go.main && window.go.main.App) {
            return await window.go.main.App.CheckForUpdates();
        }
        return { currentVersion: "1.1.2", updateAvailable: false };
    },
    downloadAndApplyUpdate: async (url) => {
        if (window.go && window.go.main && window.go.main.App) {
            return await window.go.main.App.DownloadAndApplyUpdate(url);
        }
    }
};

// UI Helper: Modal / Alert system
function showCustomModal({ title = "KatHub", message, buttons = [{ label: "OK", value: true, primary: true }] }) {
    return new Promise((resolve) => {
        const modal = document.getElementById("customModal");
        const titleEl = document.getElementById("modalTitle");
        const msgEl = document.getElementById("modalMessage");
        const buttonsContainer = document.getElementById("modalButtons");

        titleEl.textContent = title;
        msgEl.textContent = message;
        buttonsContainer.innerHTML = "";

        buttons.forEach((btn) => {
            const buttonEl = document.createElement("button");
            buttonEl.textContent = btn.label;
            buttonEl.className = btn.primary
                ? "px-4 py-1.5 rounded-md text-sm font-medium bg-[#e63946] hover:bg-[#d62839] text-white transition-colors cursor-pointer"
                : "px-4 py-1.5 rounded-md text-sm font-medium bg-[#2a2a35] hover:bg-[#32323d] text-gray-300 transition-colors cursor-pointer";
            
            buttonEl.onclick = () => {
                modal.classList.add("hidden");
                resolve(btn.value);
            };
            buttonsContainer.appendChild(buttonEl);
        });

        modal.classList.remove("hidden");
    });
}

// ----------------------------------------------------
// UI Renderers & State Updaters
// ----------------------------------------------------

function updateKatPluginsUI(info) {
    currentKatPluginsState = info;
    const statusLabel = document.getElementById("katPluginsStatusLabel");
    const mainBtn = document.getElementById("katPluginsMainButton");
    const uninstallBtn = document.getElementById("uninstallKatButton");

    statusLabel.textContent = info.statusText;
    statusLabel.style.color = info.statusColor;

    mainBtn.textContent = info.buttonText;
    mainBtn.style.backgroundColor = info.buttonColor;
    mainBtn.style.display = info.mainButtonVisible ? "flex" : "none";
    mainBtn.disabled = false;

    if (info.uninstallVisible) {
        uninstallBtn.classList.remove("hidden");
        uninstallBtn.classList.add("flex");
    } else {
        uninstallBtn.classList.add("hidden");
        uninstallBtn.classList.remove("flex");
    }
}

function updateLolConfigUI(info) {
    currentLolConfigState = info;
    const statusLabel = document.getElementById("lolConfigStatusLabel");
    const button = document.getElementById("lolConfigButton");

    statusLabel.textContent = info.statusText;
    statusLabel.style.color = info.statusColor;

    button.textContent = info.buttonText;
    button.disabled = !info.canToggle;
    
    if (info.buttonText === "Unlock") {
        button.style.backgroundColor = "#2a2a35";
        button.style.color = "#ffffff";
    } else {
        button.style.backgroundColor = "#e63946";
        button.style.color = "#ffffff";
    }
}

// ----------------------------------------------------
// Actions & Event Handlers
// ----------------------------------------------------

async function handleKatPluginsMainAction() {
    const mainBtn = document.getElementById("katPluginsMainButton");
    const progressContainer = document.getElementById("katProgressBarContainer");
    const progressBar = document.getElementById("katProgressBar");

    mainBtn.disabled = true;
    mainBtn.textContent = "Processing...";

    try {
        const ok = await GoApp.handleKatPluginsMainAction();
        if (!ok) {
            await showCustomModal({
                title: "KatHub",
                message: "Hubo un error al ejecutar la acción de plugins.",
                buttons: [{ label: "OK", value: true, primary: true }]
            });
        }
    } catch (err) {
        await showCustomModal({
            title: "KatHub Error",
            message: "Exception: " + (err.message || err),
            buttons: [{ label: "OK", value: true, primary: true }]
        });
    } finally {
        progressContainer.classList.add("hidden");
        progressBar.style.width = "0%";
        await refreshKatPluginsState();
    }
}

async function handleUninstallKatPlugins() {
    const confirmed = await showCustomModal({
        title: "Uninstall KatPlugins",
        message: "Are you sure you want to uninstall KatPlugins?",
        buttons: [
            { label: "Cancel", value: false, primary: false },
            { label: "Yes, Uninstall", value: true, primary: true }
        ]
    });

    if (!confirmed) return;

    const uninstallBtn = document.getElementById("uninstallKatButton");
    const mainBtn = document.getElementById("katPluginsMainButton");
    uninstallBtn.disabled = true;
    mainBtn.disabled = true;

    try {
        const ok = await GoApp.uninstallKatPlugins();
        if (ok) {
            await showCustomModal({
                title: "KatHub",
                message: "Plugin uninstalled successfully.",
                buttons: [{ label: "OK", value: true, primary: true }]
            });
        } else {
            await showCustomModal({
                title: "KatHub",
                message: "Error uninstalling the plugin.",
                buttons: [{ label: "OK", value: true, primary: true }]
            });
        }
    } catch (err) {
        await showCustomModal({
            title: "KatHub",
            message: "Error: " + err,
            buttons: [{ label: "OK", value: true, primary: true }]
        });
    } finally {
        uninstallBtn.disabled = false;
        mainBtn.disabled = false;
        await refreshKatPluginsState();
    }
}

async function handleLolConfigToggle() {
    const button = document.getElementById("lolConfigButton");
    button.disabled = true;

    try {
        const shouldLock = button.textContent.trim() === "Lock";
        await GoApp.toggleLolConfig(shouldLock);
        await refreshLolConfigState();
    } catch (err) {
        await showCustomModal({
            title: "KatHub",
            message: "Error modifying config attributes: " + err,
            buttons: [{ label: "OK", value: true, primary: true }]
        });
    } finally {
        button.disabled = false;
    }
}

async function handleAccountManagerClick() {
    try {
        await GoApp.openAccountManager();
    } catch (err) {
        console.error("Account manager open error:", err);
    }
}

async function handleUpdateClick() {
    if (currentUpdateInfo && currentUpdateInfo.updateAvailable && currentUpdateInfo.downloadUrl) {
        try {
            await GoApp.downloadAndApplyUpdate(currentUpdateInfo.downloadUrl);
        } catch (err) {
            await showCustomModal({
                title: "Update Error",
                message: "No se pudo completar la descarga: " + err,
                buttons: [{ label: "OK", value: true, primary: true }]
            });
        }
    } else if (currentUpdateInfo && currentUpdateInfo.releaseUrl) {
        window.open(currentUpdateInfo.releaseUrl, "_blank");
    }
}

// ----------------------------------------------------
// Refresh Loop & Listeners
// ----------------------------------------------------

async function refreshKatPluginsState() {
    try {
        const info = await GoApp.getKatPluginsState();
        updateKatPluginsUI(info);
    } catch (e) {
        console.error("Error checking KatPlugins state:", e);
    }
}

async function refreshLolConfigState() {
    try {
        const info = await GoApp.getLolConfigState();
        updateLolConfigUI(info);
    } catch (e) {
        console.error("Error checking LoL config state:", e);
    }
}

async function checkAppUpdates() {
    try {
        const update = await GoApp.checkForUpdates();
        currentUpdateInfo = update;

        const lblVersion = document.getElementById("lblVersion");
        const lblUpdate = document.getElementById("lblUpdate");

        if (update && update.currentVersion) {
            lblVersion.textContent = `v${update.currentVersion}`;
        }

        if (update && update.updateAvailable) {
            lblUpdate.classList.remove("hidden");
            lblUpdate.textContent = "¡Actualización disponible! (Click aquí)";
        } else {
            lblUpdate.classList.add("hidden");
        }
    } catch (e) {
        console.warn("Check updates error:", e);
    }
}

// Wails Event Listeners
function setupWailsListeners() {
    if (window.runtime && window.runtime.EventsOn) {
        window.runtime.EventsOn("plugin-install-progress", (progress) => {
            const container = document.getElementById("katProgressBarContainer");
            const bar = document.getElementById("katProgressBar");
            const mainBtn = document.getElementById("katPluginsMainButton");

            container.classList.remove("hidden");
            const pct = Math.min(Math.max(progress, 0), 100);
            bar.style.width = `${pct}%`;
            mainBtn.textContent = `Installing... ${Math.round(pct)}%`;
        });
    }
}

// Initialization on DOM ready
document.addEventListener("DOMContentLoaded", async () => {
    setupWailsListeners();
    await Promise.all([
        refreshKatPluginsState(),
        refreshLolConfigState(),
        checkAppUpdates()
    ]);
});
