class UIStore {
    activeTab = $state("kiosk");
    selectedBook = $state(/** @type {any} */ (null));
    isSidebarCollapsed = $state(false);
    pendingReservierungen = $state(0);
    isInitialRouteMatched = $state(false);

    async fetchPendingReservierungen() {
        try {
            const res = await fetch("/api/reservierungen/klassensatz/anzahl");
            if (res.ok) {
                const data = await res.json();
                this.pendingReservierungen = data.anzahl ?? 0;
            }
        } catch { /* ignore */ }
    }
}

export const uiStore = new UIStore();
