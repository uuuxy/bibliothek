import { apiFetch, apiClient } from './apiFetch.js';
import { toastStore } from './stores/toastStore.svelte.js';

export function useStudentProfile() {
	/** @type {any} */
	let profile = $state(null);
	/** @type {any[]} */
	let vormerkungen = $state([]);
	/** @type {any[]} */
	let gebuehren = $state([]);
	/** @type {any[]} */
	let bescheide = $state([]);
	let loading = $state(true);
	let showWebcam = $state(false);
	let timestamp = $state(Date.now());
	let showDeleteConfirm = $state(false);
	let activeTab = $state('ausleihen');
	let showEditModal = $state(false);
	let showDamageModal = $state(false);
	let showLockModal = $state(false);
	let damageBook = $state(/** @type {any} */ (null));
	let isSubmittingDamage = $state(false);
	let globalErrorToast = $state(/** @type {string|null} */ (null));
	let rechnungPdfLoading = $state(false);
	let kontoauszugPdfLoading = $state(false);

	// Sequenznummer wie im orderStore: Zwei Akten kurz hintereinander geöffnet, und die
	// langsamere Antwort gewinnt. `laufNr` verwirft, was überholt wurde.
	let laufNr = 0;

	async function fetchProfile(studentId) {
		if (!studentId) return;
		const meine = ++laufNr;
		loading = true;
		try {
			const [resProfile, resVormerkungen, resGebuehren, resBescheide] = await Promise.all([
				apiFetch(`/api/schueler/${studentId}`),
				apiFetch(`/api/vormerkungen?schueler_id=${studentId}`),
				apiFetch(`/api/schueler/${studentId}/schadensfaelle`),
				apiFetch(`/api/schueler/${studentId}/bescheide`)
			]);
			if (meine !== laufNr) return; // eine jüngere Akte ist schon unterwegs oder da
			// Jede der drei Antworten wird ZUGEWIESEN, auch wenn sie scheitert. Bis zum
			// Rasterdurchgang am 06.09.2026 stand hier dreimal `if (ok)` ohne `else`: Fiel
			// genau eine Anfrage aus (500, oder 429 vom Rate-Limiter — es sind drei
			// parallele Anfragen je Akte), behielt dieser Teil die Werte des VORHER
			// geöffneten Schülers, während Kopf und Ausleihen schon zum neuen gehörten.
			// Bei den Gebühren ist das nicht nur Anzeige: Die Karte schreibt auf die
			// Fall-ID der ZEILE („Zahlung verbucht", „Storno") — ein Klick hätte die
			// Zahlung einem fremden Schadensfall gutgeschrieben.
			profile = resProfile.ok ? await resProfile.json() : null;
			vormerkungen = resVormerkungen.ok ? await resVormerkungen.json() : [];
			// 403 (z. B. Kiosk-Rolle ohne view_students) heisst schlicht: keine Liste zeigen.
			gebuehren = resGebuehren.ok ? (await resGebuehren.json()).data || [] : [];
			bescheide = resBescheide.ok ? (await resBescheide.json()).data || [] : [];
		} catch (err) {
			if (meine !== laufNr) return;
			console.error('Fehler beim Laden des Schüler-Profils:', err);
			profile = null;
			vormerkungen = [];
			gebuehren = [];
			bescheide = [];
		} finally {
			if (meine === laufNr) loading = false;
		}
	}

	// Alles, was über einem Schüler offen stehen kann, beim Wechsel schließen. Eine Liste
	// statt fünf Zuweisungen im Aufrufer: Wer ein sechstes Blatt baut, trägt es hier ein.
	//
	// Bis zum Rasterdurchgang am 06.09.2026 setzte der Aufrufer beim Schülerwechsel nur
	// den Reiter zurück: Das Bearbeiten-Blatt überlebte ihn. Wer bei Schüler A die Adresse
	// tippte, ohne zu speichern, und dann B's Ausweis scannte, sah dasselbe Formular mit
	// B's Daten, A's Eingabe war wortlos weg — und „Speichern" ging als PATCH auf B.
	function schliesseAlleBlaetter() {
		showEditModal = false;
		showDamageModal = false;
		showLockModal = false;
		showDeleteConfirm = false;
		showWebcam = false;
		damageBook = null;
		globalErrorToast = null;
	}

	function handleDeleteSuccess(onDeselect) {
		showDeleteConfirm = false;
		if (onDeselect) onDeselect();
	}

	function handleSaveEdit(studentId) {
		showEditModal = false;
		fetchProfile(studentId);
	}

	function handlePhotoCaptured(studentId) {
		timestamp = Date.now();
		showWebcam = false;
		fetchProfile(studentId);
	}

	async function downloadRechnungPDF() {
		if (!profile) return;
		rechnungPdfLoading = true;
		globalErrorToast = null;
		try {
			const res = await apiFetch(`/api/print/rechnung/${profile.id}`);
			if (!res.ok) {
				const errText = await res.text();
				throw new Error(errText || 'Keine ausstehenden Rechnungen gefunden');
			}
			const blob = await res.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `Rechnung_${profile.vorname}_${profile.nachname}.pdf`;
			a.click();
			URL.revokeObjectURL(url);
		} catch (e) {
			globalErrorToast = String(e);
			setTimeout(() => (globalErrorToast = null), 4000);
		} finally {
			rechnungPdfLoading = false;
		}
	}

	async function downloadKontoauszugPDF() {
		if (!profile) return;
		kontoauszugPdfLoading = true;
		globalErrorToast = null;
		try {
			const res = await apiFetch(`/api/print/kontoauszug/${profile.id}`);
			if (!res.ok) {
				const errText = await res.text();
				throw new Error(errText || 'Keine aktiven Ausleihen gefunden');
			}
			const blob = await res.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `Kontoauszug_${profile.vorname}_${profile.nachname}.pdf`;
			a.click();
			URL.revokeObjectURL(url);
		} catch (e) {
			globalErrorToast = String(e);
			setTimeout(() => (globalErrorToast = null), 4000);
		} finally {
			kontoauszugPdfLoading = false;
		}
	}

	function openDamageModal(book) {
		damageBook = book;
		showDamageModal = true;
	}

	async function submitDamageReport(studentId, reason, amount, art) {
		if (!damageBook) return;
		isSubmittingDamage = true;
		try {
			const res = await apiClient.post(`/api/damage/report`, {
				loan_id: damageBook.ausleihe_id,
				schueler_id: studentId,
				// BorrowedBook liefert die Exemplar-ID als `id` — ein
				// `exemplar_id`-Feld gibt es dort nicht (500: leere UUID).
				copy_id: damageBook.id,
				beschreibung: reason,
				art,
				betrag: amount
			});
			if (res.ok) {
				const json = await res.json();
				window.open(`/api/schadensfaelle/${json.schadens_id}/pdf`, '_blank');
				showDamageModal = false;
				fetchProfile(studentId);
			} else {
				const err = await res.json().catch((e) => {
					console.error('Fehler:', e);
					return {};
				});
				toastStore.addToast(err.error || 'Fehler beim Melden.', 'error');
			}
		} catch (e) {
			console.error('Schaden melden fehlgeschlagen:', e);
			toastStore.addToast('Netzwerkfehler.', 'error');
		} finally {
			isSubmittingDamage = false;
		}
	}

	function handleLockSuccess(updated) {
		if (profile) {
			// Sofortiges Feedback: nur das Handschloss lokal übernehmen. ist_gesperrt
			// (Systemsperre) bleibt der Serverwert — die frühere Formel (manuell ||
			// offene Schäden) erfand eine dritte Sperr-Definition (sperrStatus.js).
			profile.is_manually_blocked = updated.is_manually_blocked;
			// Dann die EINE Wahrheit vom Server holen — wie nach Bearbeiten/Foto.
			// Der Sperrgrund (Anzeige seit 01.09.2026) steckt bewusst NICHT in der
			// Lock-Antwort (sie ist Stufe 1 hinter edit_students, der Grund Stufe 2
			// hinter view_students, PII-Matrix); und die Behalten/Löschen-Logik des
			// Servers clientseitig nachzubauen wäre eine doppelte Wahrheitsquelle.
			fetchProfile(profile.id);
		}
	}

	return {
		get profile() {
			return profile;
		},
		set profile(v) {
			profile = v;
		},
		get vormerkungen() {
			return vormerkungen;
		},
		set vormerkungen(v) {
			vormerkungen = v;
		},
		get gebuehren() {
			return gebuehren;
		},
		get bescheide() {
			return bescheide;
		},
		get loading() {
			return loading;
		},
		get showWebcam() {
			return showWebcam;
		},
		set showWebcam(v) {
			showWebcam = v;
		},
		get timestamp() {
			return timestamp;
		},
		get showDeleteConfirm() {
			return showDeleteConfirm;
		},
		set showDeleteConfirm(v) {
			showDeleteConfirm = v;
		},
		get activeTab() {
			return activeTab;
		},
		set activeTab(v) {
			activeTab = v;
		},
		get showEditModal() {
			return showEditModal;
		},
		set showEditModal(v) {
			showEditModal = v;
		},
		get showDamageModal() {
			return showDamageModal;
		},
		set showDamageModal(v) {
			showDamageModal = v;
		},
		get showLockModal() {
			return showLockModal;
		},
		set showLockModal(v) {
			showLockModal = v;
		},
		get damageBook() {
			return damageBook;
		},
		get isSubmittingDamage() {
			return isSubmittingDamage;
		},
		get globalErrorToast() {
			return globalErrorToast;
		},
		get rechnungPdfLoading() {
			return rechnungPdfLoading;
		},
		get kontoauszugPdfLoading() {
			return kontoauszugPdfLoading;
		},
		fetchProfile,
		schliesseAlleBlaetter,
		handleDeleteSuccess,
		handleSaveEdit,
		handlePhotoCaptured,
		downloadRechnungPDF,
		downloadKontoauszugPDF,
		openDamageModal,
		submitDamageReport,
		handleLockSuccess
	};
}
