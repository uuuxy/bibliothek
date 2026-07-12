// stores/orderStore.svelte.js
// Zustand und Logik des Bestellwesens: Lieferanten, Warenkorb, Titel-Suche,
// Zulauf und Bestellbedarf. Die Views (BestellWorkspace & Kinder) bleiben rein
// darstellend.

import { apiGet, apiPost, apiPut, apiDelete } from '../apiFetch.js';
import { toastStore } from './toastStore.svelte.js';

/** @typedef {{ id: string, name: string, email: string, customerNumber: string }} Supplier */
/** @typedef {{ id: string, titel: string, autor: string, isbn: string, verlag: string, cover_url: string, menge: number, preis: number, generate_barcodes: boolean }} CartItem */

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

	async init() {
		await Promise.all([
			this.loadSuppliers(),
			this.loadIncomingShipments(),
			this.loadRecommendations()
		]);
	}

	async loadSuppliers() {
		try {
			this.suppliers = (await apiGet('/api/lieferanten')) || [];
		} catch {
			/* apiFetch zeigt Fehler-Toast */
		}
		// Auswahl per ID stabil halten; Index-basierte Auswahl kippt bei Reload/Umsortierung
		if (!this.suppliers.some((s) => s.id === this.selectedSupplierId)) {
			this.selectedSupplierId = this.suppliers[0]?.id ?? '';
		}
	}

	async loadIncomingShipments() {
		try {
			this.incomingShipments = (await apiGet('/api/bestellungen/zulauf')) || [];
		} catch {
			/* apiFetch zeigt Fehler-Toast */
		}
	}

	async loadRecommendations() {
		try {
			this.recommendations = (await apiGet('/api/bestellungen')) || [];
		} catch {
			/* apiFetch zeigt Fehler-Toast */
		}
	}

	/** @param {string} name @param {string} email @param {string} customerNumber */
	async addSupplier(name, email, customerNumber) {
		if (!name || !email || !customerNumber) return;
		try {
			await apiPost('/api/lieferanten', { name, email, customerNumber });
			await this.loadSuppliers();
		} catch {
			/* apiFetch zeigt Fehler-Toast */
		}
	}

	/** @param {string} id @param {string} name @param {string} email @param {string} customerNumber */
	async editSupplier(id, name, email, customerNumber) {
		try {
			await apiPut(`/api/lieferanten/${id}`, { name, email, customerNumber });
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

	/** @param {any} book */
	addToCart(book, menge = 1, withBarcodes = false) {
		// Im Cart liegt immer die titel_id — der Duplikat-Check muss denselben Schlüssel nutzen
		const key = book.titel_id ?? book.id;
		const isbn = book.isbn ?? book.ISBN ?? '';
		const existing = this.cart.find((item) => item.id === key || (isbn && item.isbn === isbn));
		if (existing) {
			existing.menge += menge;
			if (withBarcodes) existing.generate_barcodes = true;
		} else {
			this.cart.push({
				id: key,
				titel: book.titel,
				autor: book.autor,
				isbn,
				verlag: book.verlag ?? '',
				cover_url: book.cover_url ?? '',
				menge,
				preis: 0.0,
				generate_barcodes: withBarcodes
			});
		}
		this.resetSearch();
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
					preis: Number(item.preis) || 0,
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
