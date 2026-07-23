// stores/mahnwesen.svelte.js
// Status- und Logikverwaltung für das Mahnwesen (Svelte 5 Runes)

import { apiFetch } from '../apiFetch.js';
import { useMahnwesenPdf } from './mahnwesenPdf.svelte.js';
import { useMahnwesenMail } from './mahnwesenMail.svelte.js';
import { SvelteSet } from 'svelte/reactivity';

/**
 * Höchste Überfälligkeit (Tage) über alle Medien eines Schülers.
 * @param {any[]} medien
 * @returns {number}
 */
function maxTageUeberfaellig(medien) {
	let maxTage = 0;
	for (const m of medien) {
		if (m.tage_ueberfaellig > maxTage) maxTage = m.tage_ueberfaellig;
	}
	return maxTage;
}

/**
 * Mahnstufe eines Schülers: Lehrer → Lehrerkollegium, sonst nach Überfälligkeit.
 * @param {any} schueler
 * @param {number} maxTage
 * @returns {string}
 */
function berechneMahnstufe(schueler, maxTage) {
	if (schueler.klasse?.toLowerCase() === 'lehrer') {
		return 'Lehrerkollegium';
	}
	// Diese Liste enthält ausschließlich überfällige Ausleihen (rueckgabe_frist < now).
	// tage_ueberfaellig kann bei <24 h Überfälligkeit rechnerisch 0 sein — das ist dann
	// „gerade fällig", NICHT „erledigt". Daher gibt es hier bewusst kein 'Erledigt'.
	if (maxTage > 14) return 'Mahnung';
	return '1. Erinnerung';
}

/**
 * Creates the central Mahnwesen store.
 */
export function createMahnwesenStore() {
	let data = $state(/** @type {{ klassen: any[] } | null} */ (null));
	let loading = $state(true);
	let error = $state(/** @type {string|null} */ (null));

	// Ferien-Logik
	let ferienAktiv = $state(false);
	let ferienBezeichnung = $state('');
	let heuteRetourniert = $state(0);

	// Filter und Auswahl
	let expandedKlassen = /** @type {Set<string>} */ (new SvelteSet());
	let mahnMode = $state('datum'); // "datum" oder "jahrgang"
	let selectedKlasse = $state(''); // Klassenfilter (leer = alle); steuert Liste UND Klassen-PDF
	let searchQuery = $state(''); // Freitextsuche über Name/Klasse

	// MD3 Filter & Bulk Actions
	let activeFilter = $state('Alle'); // "Alle", "1. Erinnerung", "Mahnung", "Lehrerkollegium"
	let selectedIds = /** @type {Set<string>} */ (new SvelteSet());

	const pdfStore = useMahnwesenPdf();
	const mailStore = useMahnwesenMail();

	// Abgeleitete Werte
	let klassen = $derived(data?.klassen ?? []);
	let totalOverdue = $derived(
		klassen.reduce(
			(/** @type {number} */ sum, /** @type {any} */ k) =>
				sum +
				k.schueler.reduce(
					(/** @type {number} */ s2, /** @type {any} */ sch) => s2 + sch.medien.length,
					0
				),
			0
		)
	);

	// Flache Liste für MD3 Table
	let flatSchueler = $derived(() => {
		let list = [];
		for (const k of klassen) {
			for (const s of k.schueler) {
				const maxTage = maxTageUeberfaellig(s.medien);
				const mahnstufe = berechneMahnstufe(s, maxTage);
				list.push({ ...s, maxTage, mahnstufe, lehrer_email: k.lehrer_email });
			}
		}
		return list;
	});

	let filteredSchueler = $derived(() => {
		let list = flatSchueler();
		if (activeFilter !== 'Alle')
			list = list.filter((/** @type {any} */ s) => s.mahnstufe === activeFilter);
		if (selectedKlasse) list = list.filter((/** @type {any} */ s) => s.klasse === selectedKlasse);
		const q = searchQuery.trim().toLowerCase();
		if (q) {
			list = list.filter(
				(/** @type {any} */ s) =>
					(s.name || '').toLowerCase().includes(q) || (s.klasse || '').toLowerCase().includes(q)
			);
		}
		// Dringlichkeit zuerst: am längsten überfällig oben, dann alphabetisch. So arbeitet man
		// die Liste von oben nach unten ab.
		return [...list].sort(
			(/** @type {any} */ a, /** @type {any} */ b) =>
				b.maxTage - a.maxTage || String(a.name).localeCompare(String(b.name), 'de')
		);
	});

	/**
	 * Fetches Mahnwesen data from API.
	 */
	async function fetchData() {
		loading = true;
		error = null;
		try {
			const endpoint =
				mahnMode === 'datum' ? '/api/mahnwesen' : '/api/mahnwesen/ueberfaellig_jahrgang';
			const res = await apiFetch(endpoint);
			if (!res.ok) throw new Error((await res.text()) || 'Fehler beim Laden');
			const json = await res.json();
			data = json;
			ferienAktiv = json.ferien_aktiv || false;
			ferienBezeichnung = json.ferien_bezeichnung || '';
			heuteRetourniert = json.heute_retourniert || 0;
			selectedIds.clear();
		} catch (e) {
			error = String(e);
		} finally {
			loading = false;
		}
	}

	/** @param {string} klasse */
	function toggleKlasse(klasse) {
		const s = new SvelteSet(expandedKlassen);
		if (s.has(klasse)) s.delete(klasse);
		else s.add(klasse);
		expandedKlassen = s;
	}

	// WICHTIG: selectedIds ist ein reaktives SvelteSet, aber KEIN $state-Binding. Deshalb muss
	// es IN-PLACE mutiert werden (add/delete/clear) — eine Neuzuweisung (selectedIds = neuesSet)
	// wäre nicht reaktiv, die Checkboxen und die Auswahl-Toolbar würden nicht reagieren.
	/** @param {string} id */
	function toggleSelect(id) {
		if (selectedIds.has(id)) selectedIds.delete(id);
		else selectedIds.add(id);
	}

	function selectAllSchueler() {
		selectedIds.clear();
		for (const s of filteredSchueler()) selectedIds.add(s.schueler_id);
	}

	function deselectAllSchueler() {
		selectedIds.clear();
	}

	/** Wrapper for bulk printing that supplies needed state context. */
	async function printSelectedMahnungenWrapper() {
		await pdfStore.printSelectedMahnungen(selectedIds, filteredSchueler, fetchData);
	}

	return {
		get data() {
			return data;
		},
		get loading() {
			return loading;
		},
		get error() {
			return error;
		},
		get expandedKlassen() {
			return expandedKlassen;
		},
		get mahnMode() {
			return mahnMode;
		},
		set mahnMode(v) {
			mahnMode = v;
		},
		get selectedKlasse() {
			return selectedKlasse;
		},
		set selectedKlasse(v) {
			selectedKlasse = v;
			// Sichtbereich ändert sich → Auswahl leeren, damit sie stets zur Liste passt.
			selectedIds.clear();
		},
		get searchQuery() {
			return searchQuery;
		},
		set searchQuery(v) {
			searchQuery = v;
			selectedIds.clear();
		},
		get klassen() {
			return klassen;
		},
		get totalOverdue() {
			return totalOverdue;
		},
		get ferienAktiv() {
			return ferienAktiv;
		},
		get ferienBezeichnung() {
			return ferienBezeichnung;
		},
		get heuteRetourniert() {
			return heuteRetourniert;
		},
		get activeFilter() {
			return activeFilter;
		},
		set activeFilter(v) {
			activeFilter = v;
			selectedIds.clear();
		},
		get selectedIds() {
			return selectedIds;
		},
		get filteredSchueler() {
			return filteredSchueler();
		},

		fetchData,
		toggleKlasse,
		toggleSelect,
		selectAllSchueler,
		deselectAllSchueler,

		// Expose mail store methods and state directly
		get modalOpen() {
			return mailStore.modalOpen;
		},
		get modalKlasse() {
			return mailStore.modalKlasse;
		},
		get modalEmail() {
			return mailStore.modalEmail;
		},
		set modalEmail(v) {
			mailStore.modalEmail = v;
		},
		get modalSending() {
			return mailStore.modalSending;
		},
		get modalMsg() {
			return mailStore.modalMsg;
		},
		openModal: mailStore.openModal,
		closeModal: mailStore.closeModal,
		sendMahnliste: mailStore.sendMahnliste,
		sendBulkOverdueMails: mailStore.sendBulkOverdueMails,

		// Expose pdf store methods and state directly
		get pdfLoading() {
			return pdfStore.pdfLoading;
		},
		get elternPdfLoading() {
			return pdfStore.elternPdfLoading;
		},
		get klassePdfLoading() {
			return pdfStore.klassePdfLoading;
		},
		get globalErrorToast() {
			return pdfStore.globalErrorToast;
		},
		printSelectedMahnungen: printSelectedMahnungenWrapper,
		downloadPDF: pdfStore.downloadPDF,
		downloadElternPDF: pdfStore.downloadElternPDF,
		downloadKlassePDF: pdfStore.downloadKlassePDF
	};
}

export const mahnwesenStore = createMahnwesenStore();
