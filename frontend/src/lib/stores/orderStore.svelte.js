// stores/orderStore.svelte.js
// Zustand und Logik des Bestellwesens: Lieferanten, Warenkorb, Titel-Suche,
// Zulauf und Bestellbedarf. Die Views (BestellWorkspace & Kinder) bleiben rein
// darstellend.

import { apiGet, apiPost, apiPut, apiDelete } from '../apiFetch.js';
import { toastStore } from './toastStore.svelte.js';

/** @typedef {{ id: string, name: string, email: string, customerNumber: string }} Supplier */
/** @typedef {{ id: string, titel: string, autor: string, isbn: string, verlag: string, cover_url: string, menge: number, preis: number, preis_vorschlag: number, generate_barcodes: boolean }} CartItem */

class OrderStore {
	/** @type {Supplier[]} */
	suppliers = $state([]);
	selectedSupplierId = $state('');
	selectedSupplier = $derived(this.suppliers.find((s) => s.id === this.selectedSupplierId) ?? null);

	/** @type {CartItem[]} */
	cart = $state([]);
	total = $derived(this.cart.reduce((sum, i) => sum + i.menge * (Number(i.preis) || 0), 0));
	totalQty = $derived(this.cart.reduce((sum, i) => sum + i.menge, 0));
	submitting = $state(false);
	/** Globaler Schalter „Barcodes mitschicken" */
	attachBarcodes = $state(true);
	/**
	 * Arbeitet das Bestellwesen mit Preisen? Systemeinstellung, nicht Sitzungswahl —
	 * geladen aus /api/bestellungen/konfiguration.
	 *
	 * Vorgabe AN wie im Backend: Waere sie AUS, verschwaenden Preisfeld und Betraege fuer
	 * einen Wimpernschlag, bis die Antwort da ist — ein Flackern, das wie ein Datenverlust
	 * aussieht.
	 */
	preiseErfassen = $state(true);

	searchQuery = $state('');
	/** @type {any[]} */
	searchResults = $state([]);
	showDropdown = $state(false);
	searchLoading = $state(false);
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	#searchTimeout;
	#searchSeq = 0;

	/** @type {any[]} */
	recommendations = $state([]);
	/** @type {any[]} */
	incomingShipments = $state([]);

	/** Zeitpunkt des letzten vollständigen Ladens (0 = noch nie). */
	#zuletztGeladen = 0;

	/**
	 * Lädt die Arbeitsdaten des Bestellwesens.
	 *
	 * Der Store ist ein Singleton, seine Daten überleben also den Unmount der Ansicht.
	 * Trotzdem lud jeder Mount alles neu — und BestellWorkspace wird bei jedem
	 * Tab-Wechsel neu aufgebaut (Router.svelte nutzt {#if}/{:else if}). Der
	 * Bestellbedarf ist die mit Abstand grösste Liste der Anwendung; genau das war der
	 * spürbare Hänger beim Zurückwechseln auf „Bestellungen".
	 *
	 * Jetzt: vorhandene Daten sofort anzeigen und nur dann auffrischen, wenn sie älter
	 * als frischeDauerMs sind — im Hintergrund, ohne die Ansicht zu blockieren. Nach
	 * jeder verändernden Aktion (Wareneingang, Bestellung) laden die Aufrufer ohnehin
	 * gezielt neu, die Liste ist also nicht auf den Tab-Wechsel als Auslöser angewiesen.
	 */
	async init() {
		const frischeDauerMs = 60_000;
		const alter = Date.now() - this.#zuletztGeladen;

		if (this.#zuletztGeladen !== 0 && alter < frischeDauerMs) {
			return; // Daten sind da und frisch genug — nichts tun.
		}
		if (this.#zuletztGeladen !== 0) {
			// Daten sind da, aber älter: im Hintergrund auffrischen. Bewusst ohne await —
			// die Ansicht zeigt sofort den vorhandenen Stand.
			void this.#ladeAlles();
			return;
		}
		await this.#ladeAlles(); // Erster Aufruf: es gibt noch nichts zu zeigen.
	}

	async #ladeAlles() {
		const ergebnisse = await Promise.all([
			this.loadSuppliers(),
			this.loadIncomingShipments(),
			this.loadRecommendations(),
			this.loadKonfiguration()
		]);
		// Nur vollen Erfolg als "frisch" stempeln: Nach einem Netzwerkfehler soll der
		// nächste Mount sofort erneut laden, nicht 60 Sekunden leere Listen zeigen.
		if (ergebnisse.every(Boolean)) {
			this.#zuletztGeladen = Date.now();
		}
	}

	// Die Loader melden zurück, ob sie durchkamen (Fehler-Toast zeigt apiFetch selbst):
	// #ladeAlles stempelt den Cache nur bei vollem Erfolg. Ein gecachter Fehlschlag
	// hieße sonst: 60 Sekunden leere Listen ohne Retry beim nächsten Mount.
	async loadSuppliers() {
		let ok = true;
		try {
			this.suppliers = (await apiGet('/api/lieferanten')) || [];
		} catch {
			ok = false;
		}
		// Auswahl per ID stabil halten; Index-basierte Auswahl kippt bei Reload/Umsortierung
		if (!this.suppliers.some((s) => s.id === this.selectedSupplierId)) {
			this.selectedSupplierId = this.suppliers[0]?.id ?? '';
		}
		return ok;
	}

	async loadKonfiguration() {
		try {
			const daten = await apiGet('/api/bestellungen/konfiguration');
			this.preiseErfassen = daten?.preise_erfassen ?? true;
			return true;
		} catch {
			return false;
		}
	}

	async loadIncomingShipments() {
		try {
			this.incomingShipments = (await apiGet('/api/bestellungen/zulauf')) || [];
			return true;
		} catch {
			return false;
		}
	}

	async loadRecommendations() {
		try {
			this.recommendations = (await apiGet('/api/bestellungen')) || [];
			return true;
		} catch {
			return false;
		}
	}

	/** @param {string} name @param {string} email @param {string} customerNumber @param {boolean} [liefertMitBarcode] */
	async addSupplier(name, email, customerNumber, liefertMitBarcode = false) {
		if (!name || !email || !customerNumber) return;
		try {
			await apiPost('/api/lieferanten', {
				name,
				email,
				customerNumber,
				liefert_mit_barcode: liefertMitBarcode
			});
			await this.loadSuppliers();
		} catch {
			/* apiFetch zeigt Fehler-Toast */
		}
	}

	/** @param {string} id @param {string} name @param {string} email @param {string} customerNumber @param {boolean} [liefertMitBarcode] */
	async editSupplier(id, name, email, customerNumber, liefertMitBarcode = false) {
		try {
			await apiPut(`/api/lieferanten/${id}`, {
				name,
				email,
				customerNumber,
				liefert_mit_barcode: liefertMitBarcode
			});
			await this.loadSuppliers();
			toastStore.addToast('Lieferant aktualisiert.', 'success');
		} catch {
			/* apiFetch zeigt Fehler-Toast */
		}
	}

	/** @param {string} id */
	async removeSupplier(id) {
		try {
			await apiDelete(`/api/lieferanten/${id}`);
			await this.loadSuppliers();
		} catch {
			/* apiFetch zeigt Fehler-Toast */
		}
	}

	handleSearchInput() {
		clearTimeout(this.#searchTimeout);
		const raw = this.searchQuery.trim();
		if (raw.length < 2) {
			this.searchResults = [];
			this.showDropdown = false;
			return;
		}
		this.#searchTimeout = setTimeout(() => this.#performSearch(raw), 300);
	}

	/** @param {string} query */
	async #performSearch(query) {
		// Sequenznummer verwirft Out-of-Order-Antworten (DNB/Google-Latenzen schwanken stark)
		const seq = ++this.#searchSeq;
		this.searchLoading = true;
		try {
			const data = await apiPost('/api/bestellungen/suche', { query });
			if (seq !== this.#searchSeq) return;
			this.searchResults = data || [];
			this.showDropdown = this.searchResults.length > 0;
		} catch {
			if (seq !== this.#searchSeq) return;
			this.searchResults = [];
			this.showDropdown = false;
		} finally {
			if (seq === this.#searchSeq) this.searchLoading = false;
		}
	}

	resetSearch() {
		this.searchQuery = '';
		this.searchResults = [];
		this.showDropdown = false;
	}

	/**
	 * Ohne dritten Parameter folgt die Barcode-Wahl dem Schalter "Barcodes mitschicken"
	 * im Warenkorb — der Vorgabewert war `false`, und das war der Grund, warum eine
	 * Bestellung ohne Barcodebogen hinausging, obwohl der Schalter an war:
	 *
	 * Der Bestellbedarf (die tägliche Arbeitsfläche) ruft addToCart mit EINEM Argument
	 * auf, die Position landete also immer mit generate_barcodes: false im Warenkorb.
	 * Beim Absenden wird der Schalter nur als Sperre angewandt (`schalter ? wert : false`)
	 * — er konnte damit abschalten, aber niemals einschalten. Wer über den Bestellbedarf
	 * bestellte, bekam nie Barcodes, egal was der Schalter sagte.
	 * @param {any} book
	 * @param {number} menge
	 * @param {boolean} withBarcodes
	 */
	addToCart(book, menge = 1, withBarcodes = this.attachBarcodes) {
		// Im Cart liegt immer die titel_id — der Duplikat-Check muss denselben Schlüssel nutzen
		const key = book.titel_id ?? book.id;
		const isbn = book.isbn ?? book.ISBN ?? '';
		// Der DNB-Ladenpreis (MARC21 020 $c) als VORSCHLAG. Er wird mitgeführt, damit der
		// Warenkorb einen selbst getippten Preis von einem übernommenen unterscheiden kann.
		const vorschlag = Number(book.preis_vorschlag) || 0;
		const existing = this.cart.find((item) => item.id === key || (isbn && item.isbn === isbn));
		if (existing) {
			existing.menge += menge;
			if (withBarcodes) existing.generate_barcodes = true;
			// Einen bereits erfassten Preis NICHT überschreiben: Wer ihn getippt hat, kennt
			// den Schulpreis — der Vorschlag ist nur der Ladenpreis bei Erscheinen.
			if (!existing.preis && vorschlag) {
				existing.preis = vorschlag;
				existing.preis_vorschlag = vorschlag;
			}
		} else {
			this.cart.push({
				id: key,
				titel: book.titel,
				autor: book.autor,
				isbn,
				verlag: book.verlag ?? '',
				cover_url: book.cover_url ?? '',
				menge,
				preis: vorschlag,
				preis_vorschlag: vorschlag,
				generate_barcodes: withBarcodes
			});
		}
		this.resetSearch();
		// Ohne mitgelieferten Vorschlag (Weg ueber den Bestellbedarf) einen nachladen.
		// Bewusst hier und nicht beim Laden der Liste: Der Bestellbedarf umfasst hunderte
		// Titel, das waeren hunderte DNB-Abfragen bei jedem Oeffnen der Ansicht — fuer
		// Preise, die fast alle niemand braucht.
		if (!vorschlag) void this.#ladePreisvorschlag(key, isbn);
	}

	/**
	 * Holt den DNB-Ladenpreis zu einer ISBN nach und traegt ihn nach, falls im Warenkorb
	 * noch kein Preis steht.
	 *
	 * Nutzt die vorhandene Bestellsuche: Sie fragt fuer eine ISBN ohnehin die DNB ab und
	 * liefert preis_vorschlag mit. Ein zweiter Endpunkt fuer dieselbe Auskunft waere eine
	 * Dopplung, die frueher oder spaeter anders antwortet als die erste.
	 *
	 * Still im Fehlerfall: Ein fehlender Vorschlag ist kein Problem, das den Benutzer
	 * etwas angeht — er tippt den Preis dann wie bisher selbst.
	 * @param {string} key @param {string} isbn
	 */
	async #ladePreisvorschlag(key, isbn) {
		if (!isbn || !this.preiseErfassen) return;
		try {
			const treffer = await apiPost('/api/bestellungen/suche', { query: isbn });
			const vorschlag = Number(
				(treffer || []).find((/** @type {any} */ t) => t.preis_vorschlag > 0)?.preis_vorschlag
			);
			if (!vorschlag) return;
			const pos = this.cart.find((i) => i.id === key);
			// Nur eintragen, wenn inzwischen niemand selbst etwas erfasst hat.
			if (pos && !pos.preis) {
				pos.preis = vorschlag;
				pos.preis_vorschlag = vorschlag;
			}
		} catch {
			/* ohne Vorschlag weiterarbeiten */
		}
	}

	/** @param {number} idx */
	removeFromCart(idx) {
		this.cart.splice(idx, 1);
	}

	async submitOrder() {
		const supplier = this.selectedSupplier;
		if (!this.cart.length || !supplier) return;
		this.submitting = true;
		try {
			const data = await apiPost('/api/bestellungen', {
				supplier_id: supplier.id,
				items: this.cart.map((item) => ({
					titel_id: item.id,
					menge: item.menge,
					// Ohne Preiserfassung wird auch nichts erfasst. Sonst wanderte der
					// DNB-Vorschlag in die Bestellhistorie, obwohl das Preisfeld gar nicht
					// sichtbar war — ein Betrag, den nie jemand gesehen oder bestaetigt hat.
					preis: this.preiseErfassen ? Number(item.preis) || 0 : 0,
					generate_barcodes: this.attachBarcodes ? item.generate_barcodes : false
				}))
			});
			this.cart = [];
			const toastType = data?.status === 'warning' ? 'error' : 'success';
			const barcodeInfo =
				data?.ordered_qty != null ? ` (${data.ordered_qty} Barcodes reserviert.)` : '';
			toastStore.addToast((data?.message ?? 'Bestellung ausgelöst.') + barcodeInfo, toastType);
			await this.loadIncomingShipments();
			this.loadRecommendations();
		} catch {
			/* apiFetch zeigt Fehler-Toast */
		} finally {
			this.submitting = false;
		}
	}
}

export const orderStore = new OrderStore();
