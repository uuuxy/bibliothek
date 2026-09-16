import { apiFetch, apiClient } from './apiFetch.js';
import { istKollegium } from './leserArt.js';
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
	// Listen, deren Abruf gescheitert ist — wie fehlendeListen in der Buch-Akte (useBookAkte).
	// Ihre Karte ist leer, weil nichts ankam, nicht weil nichts vorläge: Eltern mit dem Brief
	// in der Bibliothek, GET …/bescheide in 503 — und die Auskunft „bei uns liegt kein
	// Bescheid vor" (Review 14.09.2026, OFFEN.md 1.5).
	let fehlendeListen = $state(/** @type {string[]} */ ([]));
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
			// Jede der vier Antworten wird ZUGEWIESEN, auch wenn sie scheitert. Bis zum
			// Rasterdurchgang am 06.09.2026 stand hier dreimal `if (ok)` ohne `else`: Fiel
			// genau eine Anfrage aus (500, oder 429 vom Rate-Limiter — es sind vier
			// parallele Anfragen je Akte), behielt dieser Teil die Werte des VORHER
			// geöffneten Schülers, während Kopf und Ausleihen schon zum neuen gehörten.
			// Bei den Gebühren ist das nicht nur Anzeige: Die Karte schreibt auf die
			// Fall-ID der ZEILE („Zahlung verbucht", „Storno") — ein Klick hätte die
			// Zahlung einem fremden Schadensfall gutgeschrieben.
			//
			// Und seit dem 15.09.2026 wird der Ausfall VERMERKT: Eine leere Liste nach 503 sah
			// bis dahin genauso aus wie „nichts offen". 403 (z. B. Kiosk-Rolle ohne
			// view_students) heißt dagegen schlicht: keine Liste zeigen — kein Ausfall.
			const fehlend = /** @type {string[]} */ ([]);
			/** @param {Response} res @param {string} name @param {(json: any) => any[]} auspacken */
			const listeOderVermerk = async (res, name, auspacken) => {
				if (res.ok) return auspacken(await res.json()) || [];
				if (res.status !== 403) fehlend.push(name);
				return [];
			};
			profile = resProfile.ok ? await resProfile.json() : null;
			vormerkungen = await listeOderVermerk(resVormerkungen, 'Vormerkungen', (j) => j);
			gebuehren = await listeOderVermerk(resGebuehren, 'Gebühren', (j) => j.data);
			bescheide = await listeOderVermerk(resBescheide, 'Bescheide', (j) => j.data);
			fehlendeListen = fehlend;
		} catch (err) {
			if (meine !== laufNr) return;
			console.error('Fehler beim Laden des Schüler-Profils:', err);
			profile = null;
			vormerkungen = [];
			gebuehren = [];
			bescheide = [];
			fehlendeListen = ['Vormerkungen', 'Gebühren', 'Bescheide'];
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

	// Der Dialog bekommt die Daten der Ausleihe UND die Auskunft, ob eine Forderung
	// entsteht: Bei einem Kollegen entsteht keine (entschieden am 16.09.2026, Begründung in
	// repository/schaden_melden.go). Der Server entscheidet das selbst; hier steht es, damit
	// der Dialog nicht nach einem Betrag fragt, den niemand fordern wird.
	function openDamageModal(book) {
		damageBook = { ...book, ohneForderung: istKollegium(profile) };
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
				// Kein Brief-Fenster mehr: Der frühere Elternbrief verlangte Barzahlung in der
				// Bibliothek und widersprach dem Bescheid des Landes (OFFEN.md 5.2). Der Brief
				// ist ein eigener Schritt — „Bescheid erstellen" an der Gebühren-Karte.
				toastStore.addToast(
					damageBook.ohneForderung
						? 'Verlust/Schaden gebucht, das Exemplar ist ausgesondert. Eine Forderung entsteht bei einem Kollegen nicht.'
						: 'Verlust/Schaden gebucht. Die Forderung steht unter „Gebühren & Schäden"; der Bescheid ist ein eigener Schritt.',
					'success'
				);
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
		get fehlendeListen() {
			return fehlendeListen;
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
