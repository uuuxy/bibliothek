import { appState } from "../../inventur/lib/store.svelte.js";

class AuthStore {
    isLoggedIn = $state(false);
    currentUser = $state(/** @type {any} */ (null));
    heartbeatOk = $state(true);
    lastHeartbeatTime = $state(Date.now());
    loginBarcode = $state("");
    sseSource = $state(/** @type {any} */ (null));
    loginError = $state(/** @type {string | null} */ (null));

    connectSSE() {
        if (this.sseSource) this.sseSource.close();
        const source = new EventSource("/events");
        this.sseSource = source;
        this.lastHeartbeatTime = Date.now();
        this.heartbeatOk = true;
        source.addEventListener("ping", () => {
            this.lastHeartbeatTime = Date.now();
            this.heartbeatOk = true;
        });
        source.onerror = () => { this.heartbeatOk = false; };
    }

    /** 
     * @param {Event|null} e 
     * @param {Function} [onRoleCallback]
     */
    async handleLogin(e, onRoleCallback) {
        if (e) e.preventDefault();
        if (!this.loginBarcode.trim()) return;
        this.loginError = null;

        try {
            const res = await fetch("/login/barcode", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ barcode_id: this.loginBarcode })
            });
            if (!res.ok) {
                let msg = "Login fehlgeschlagen";
                try { const d = await res.json(); msg = d.error || msg; } catch { try { msg = (await res.text()) || msg; } catch {} }
                throw new Error(msg);
            }
            this.currentUser = await res.json();
            this.isLoggedIn = true;
            this.loginBarcode = "";
            this.connectSSE();

            if (this.currentUser && (this.currentUser.rolle === "admin" || this.currentUser.rolle === "mitarbeiter")) {
                appState.adminAuthenticated = true;
                appState.guestAuthenticated = true;
                if (onRoleCallback) onRoleCallback(this.currentUser.rolle);
            } else if (this.currentUser && this.currentUser.rolle === "lehrer") {
                appState.guestAuthenticated = true;
                if (onRoleCallback) onRoleCallback("lehrer");
            } else {
                if (onRoleCallback) onRoleCallback(this.currentUser?.rolle || "");
            }
        } catch (err) {
            const errorMessage = /** @type {any} */ (err).message || String(err);
            this.loginError = errorMessage;
            this.loginBarcode = "";
            setTimeout(() => { this.loginError = null; }, 4000);
        }
    }

    handleLogout(onLogoutCallback) {
        this.isLoggedIn = false;
        this.currentUser = null;
        this.loginBarcode = "";
        this.loginError = null;
        appState.adminAuthenticated = false;
        appState.guestAuthenticated = false;
        if (this.sseSource) {
            this.sseSource.close();
            this.sseSource = null;
        }
        if (onLogoutCallback) onLogoutCallback();
    }
}

export const authStore = new AuthStore();
