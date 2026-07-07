import { appState } from "../../inventur/lib/store.svelte.js";

class AuthStore {
    isLoggedIn = $state(false);
    currentUser = $state(/** @type {any} */ (null));
    heartbeatOk = $state(true);
    lastHeartbeatTime = $state(Date.now());
    loginEmail = $state("");
    loginPassword = $state("");
    sseSource = $state(/** @type {any} */ (null));
    loginError = $state(/** @type {string | null} */ (null));

    /** @type {ReturnType<typeof setInterval> | null} */
    #refreshTimer = null;

    // Sliding Session: Der Server erneuert das Cookie erst bei <50% Restlaufzeit
    // (12h-Token → Erneuerung frühestens nach 6h). Der 30-Minuten-Tick ist also
    // fast immer ein billiges "skipped" — hält aber Kiosk-Tabs über Nacht am Leben.
    startSessionRefresh() {
        this.stopSessionRefresh();
        this.#refreshTimer = setInterval(async () => {
            try {
                const res = await fetch("/api/auth/refresh", { method: "POST" });
                if (res.status === 401) this.handleLogout();
            } catch { /* offline — der SSE-Heartbeat meldet das bereits */ }
        }, 30 * 60 * 1000);
    }

    stopSessionRefresh() {
        if (this.#refreshTimer) {
            clearInterval(this.#refreshTimer);
            this.#refreshTimer = null;
        }
    }

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
        source.onerror = () => { 
            this.heartbeatOk = false; 
            // Automatisch neu verbinden nach Verbindungsabbruch (z.B. durch Druckdialog oder Sleep)
            setTimeout(() => {
                if (this.isLoggedIn) this.connectSSE();
            }, 2000);
        };
    }

    /** 
     * @param {Event|null} e 
     * @param {Function} [onRoleCallback]
     */
    async handleLogin(e, onRoleCallback) {
        if (e) e.preventDefault();
        if (!this.loginEmail.trim() || !this.loginPassword) return;
        this.loginError = null;

        try {
            const res = await fetch("/login", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ email: this.loginEmail, password: this.loginPassword })
            });
            if (!res.ok) {
                let msg = "Login fehlgeschlagen";
                try { const d = await res.json(); msg = d.error || msg; } catch { try { msg = (await res.text()) || msg; } catch {} }
                throw new Error(msg);
            }
            this.currentUser = await res.json();
            this.isLoggedIn = true;
            this.loginEmail = "";
            this.loginPassword = "";
            this.connectSSE();
            this.startSessionRefresh();

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
            this.loginPassword = "";
            setTimeout(() => { this.loginError = null; }, 4000);
        }
    }

    handleLogout(onLogoutCallback) {
        this.isLoggedIn = false;
        this.currentUser = null;
        this.loginEmail = "";
        this.loginPassword = "";
        this.loginError = null;
        appState.adminAuthenticated = false;
        appState.guestAuthenticated = false;
        this.stopSessionRefresh();
        if (this.sseSource) {
            this.sseSource.close();
            this.sseSource = null;
        }
        if (onLogoutCallback) onLogoutCallback();
    }
}

export const authStore = new AuthStore();
