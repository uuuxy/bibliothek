import { apiFetch } from '../apiFetch.js';

class UIStore {
	activeTab = $state('kiosk');
	selectedBook = $state(/** @type {any} */ (null));
	isSidebarCollapsed = $state(false);
	pendingReservierungen = $state(0);
	/**
	 * Exemplare ohne gedrucktes Etikett. Liegt hier und nicht im Bestellwesen, weil die
	 * Zahl die SEITENLEISTE speist: Die Arbeit wartet im Druck-Center, also gehoert der
	 * Zaehler an dessen Navigationsziel. Vorher stand sie als Banner ueber dem
	 * Bestellbedarf — einer Seite, die damit nichts zu tun hat.
	 */
	offeneEtiketten = $state(0);
	// Offene Lehrer-Anliegen (Wünsche & Meldungen). Bis 24.08.2026 zählte sie niemand —
	// wer nicht ohnehin bestellen wollte, sah den dritten Reiter nie.
	offeneAnliegen = $state(0);
	isInitialRouteMatched = $state(false);
	/** Welche Statistik-Detailliste die stats_detail-Seite zeigt (deep-linkbar via URL). */
	statsDetailKind = $state(/** @type {'renner' | 'ladenhueter'} */ ('renner'));
	/**
	 * Aus einer anderen Ansicht (Mahnwesen/Abgänger) angefordertes Schülerprofil.
	 * Zentral im Store, bewusst NICHT localStorage: welches Profil DU gerade ansiehst,
	 * ist Session-lokal und nicht PC-übergreifend zu teilen (Multi-PC). StudentDirectory
	 * greift die ID auf, öffnet das Profil und setzt sie zurück.
	 */
	requestedStudentId = $state(/** @type {string | null} */ (null));
	/**
	 * Aus einem System-Alert angeforderte KATEGORIE der Einstellungen (ihre id, z. B.
	 * 'erreichbarkeit'). Gleiche Mechanik wie requestedStudentId: SystemSettings greift
	 * den Wert auf und setzt ihn zurück. Damit verweist ein Alert direkt dorthin, wo
	 * sich das Problem beheben lässt, statt den Nutzer auf der ersten Kategorie
	 * abzusetzen. Bis zum 23.08.2026 stand hier ein Reiter-NAME ("Allgemein").
	 */
	requestedSettingsTab = $state(/** @type {string | null} */ (null));
	/**
	 * Angeforderter Reiter des Druck-Centers, gleiche Mechanik. Das Bestellwesen weist
	 * auf offene Etiketten hin und schickt von dort direkt in die Nachdruck-Liste —
	 * ohne den Hinweis "wechseln Sie bitte ins Druck-Center und dann auf den dritten
	 * Reiter", der eine Arbeit beschreibt, die das Programm selbst erledigen kann.
	 */
	requestedDruckCenterTab = $state(/** @type {string | null} */ (null));
	/**
	 * Vorbelegung für den Filter der Nachdruck-Liste. Die Bestellhistorie verweist damit
	 * auf genau den Titel, den man dort gerade ansieht — statt den Benutzer die Liste
	 * noch einmal von Hand durchsuchen zu lassen, obwohl das Programm den Titel kennt.
	 */
	requestedEtikettenFilter = $state(/** @type {string | null} */ (null));
	/**
	 * Aus dem Druck-Center angeforderter Klassen-Stapeldruck: Die Schülerdatei greift
	 * den Klassennamen auf, sucht ihn und markiert alle Treffer — der Druck selbst
	 * bleibt dort (EIN Druckpfad; Warnung bei fehlendem Ablaufdatum und die
	 * Etiketten-Startposition stehen davor, nicht dahinter).
	 */
	requestedKlassenDruck = $state(/** @type {string | null} */ (null));

	async fetchPendingReservierungen() {
		try {
			const res = await apiFetch('/api/reservierungen/klassensatz/anzahl');
			if (res.ok) {
				const data = await res.json();
				this.pendingReservierungen = data.anzahl ?? 0;
			}
		} catch {
			/* ignore */
		}
	}

	async fetchOffeneAnliegen() {
		try {
			const res = await apiFetch('/api/anliegen/anzahl');
			if (res.ok) {
				const data = await res.json();
				this.offeneAnliegen = data.anzahl ?? 0;
			}
		} catch {
			/* ignore */
		}
	}

	async fetchOffeneEtiketten() {
		try {
			const res = await apiFetch('/api/exemplare/etiketten-offen/anzahl');
			if (res.ok) {
				const data = await res.json();
				this.offeneEtiketten = data.anzahl ?? 0;
			}
		} catch {
			/* Ein fehlender Zaehler darf die Navigation nicht aufhalten. */
		}
	}
}

export const uiStore = new UIStore();
