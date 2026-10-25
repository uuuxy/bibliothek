import { untrack } from 'svelte';
import { apiFetch, apiClient, leseApiFehler } from './apiFetch.js';
import { istKollegium } from './leserArt.js';
import { snackbars } from './stores/snackbars.svelte.js';
import { fehlertext } from './utils/fehlertext.js';

export function useLeserakte() {
	/** @type {any} */
	let akte = $state(null);
	/** @type {any[]} */
	let vormerkungen = $state([]);
	/** @type {any[]} */
	let gebuehren = $state([]);
	/** @type {any[]} */
	let bescheide = $state([]);
	// Listen, deren Abruf gescheitert ist, wie in der Buch-Akte (useBuchakte): Ihre Karte ist
	// leer, weil nichts ankam, nicht weil nichts vorläge („bei uns liegt kein Bescheid vor").
	let fehlendeListen = $state(/** @type {string[]} */ ([]));
	let laedt = $state(true);
	let kameraOffen = $state(false);
	let fotoStand = $state(Date.now());
	let loeschfrageOffen = $state(false);
	let aktiverReiter = $state('ausleihen');
	let bearbeitenOffen = $state(false);
	let schadenOffen = $state(false);
	let sperreOffen = $state(false);
	let schadenExemplar = $state(/** @type {any} */ (null));
	let meldetSchaden = $state(false);
	let rechnungLaedt = $state(false);
	let kontoauszugLaedt = $state(false);

	// Sequenznummer wie im bestellStore: Zwei Akten kurz hintereinander geöffnet, und die
	// langsamere Antwort gewinnt. `laufNr` verwirft, was überholt wurde.
	let laufNr = 0;

	// Solange „Stammdaten bearbeiten" offen ist, lädt die Akte desselben Lesers nicht neu (Scan,
	// Live-Leitung): Das Neuladen baute die Maske samt Eingaben ab. „Zurück" holt es nach.
	// Gelesen ohne Abhängigkeit: Der $effect der Leserakte, der ladeAkte ruft, schlösse die Maske
	// sonst bei jedem Öffnen wieder (schliesseAlleBlaetter).
	let nachladen = false;

	async function ladeAkte(studentId) {
		if (!studentId) return;
		if (untrack(() => bearbeitenOffen && studentId === akte?.id)) {
			nachladen = true;
			return;
		}
		nachladen = false;
		const meine = ++laufNr;
		laedt = true;
		try {
			const [resProfile, resVormerkungen, resGebuehren, resBescheide] = await Promise.all([
				apiFetch(`/api/schueler/${studentId}`),
				apiFetch(`/api/vormerkungen?schueler_id=${studentId}`),
				apiFetch(`/api/schueler/${studentId}/schadensfaelle`),
				apiFetch(`/api/schueler/${studentId}/bescheide`)
			]);
			// Jede der vier Antworten wird zugewiesen, auch wenn sie scheitert, und erst nach dem Lesen
			// aller Körper (wie useBuchakte): Sonst stünden Teile des vorher geöffneten Lesers da, und
			// „Zahlung verbucht" schriebe auf dessen Schadensfall. 403 heißt nur, keine Liste zu zeigen.
			const fehlend = /** @type {string[]} */ ([]);
			/** @param {Response} res @param {string} name */
			const listeOderVermerk = async (res, name) => {
				if (res.ok) return (await res.json()) || [];
				if (res.status !== 403) fehlend.push(name);
				return [];
			};
			const neu = await Promise.all([
				resProfile.ok ? resProfile.json() : null,
				listeOderVermerk(resVormerkungen, 'Vormerkungen'),
				listeOderVermerk(resGebuehren, 'Gebühren'),
				listeOderVermerk(resBescheide, 'Bescheide')
			]);
			if (meine !== laufNr) return;
			[akte, vormerkungen, gebuehren, bescheide] = neu;
			fehlendeListen = fehlend;
		} catch (err) {
			if (meine !== laufNr) return;
			console.error('Fehler beim Laden des Schüler-Profils:', err);
			[akte, vormerkungen, gebuehren, bescheide] = [null, [], [], []];
			fehlendeListen = ['Vormerkungen', 'Gebühren', 'Bescheide'];
		} finally {
			if (meine === laufNr) laedt = false;
		}
	}

	// Alles, was über einem Leser offen stehen kann, beim Wechsel schließen; wer ein weiteres
	// Blatt baut, trägt es hier ein. Ein offenes Bearbeiten-Blatt schickte „Speichern" sonst an
	// den neuen Leser.
	function schliesseAlleBlaetter() {
		bearbeitenOffen = false;
		schadenOffen = false;
		sperreOffen = false;
		loeschfrageOffen = false;
		kameraOffen = false;
		schadenExemplar = null;
	}

	function nachLoeschen(onDeselect, id) {
		if (akte?.id !== id) return;
		loeschfrageOffen = false;
		onDeselect?.();
	}

	function nachBearbeiten(studentId) {
		bearbeitenOffen = false;
		ladeAkte(studentId);
	}

	function schliesseBearbeiten() {
		bearbeitenOffen = false;
		if (nachladen) ladeAkte(akte?.id);
	}

	function nachFoto(studentId) {
		fotoStand = Date.now();
		kameraOffen = false;
		ladeAkte(studentId);
	}

	async function ladeRechnungHerunter() {
		if (!akte) return;
		rechnungLaedt = true;
		try {
			const res = await apiFetch(`/api/print/rechnung/${akte.id}`);
			if (!res.ok) {
				// leseApiFehler packt die Meldung aus dem JSON des Servers; roh stünden die
				// Klammern im Toast.
				throw new Error((await leseApiFehler(res)) || 'Keine ausstehenden Rechnungen gefunden');
			}
			const blob = await res.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `Rechnung_${akte.vorname}_${akte.nachname}.pdf`;
			a.click();
			URL.revokeObjectURL(url);
		} catch (e) {
			snackbars.addToast(fehlertext(e), 'error');
		} finally {
			rechnungLaedt = false;
		}
	}

	async function ladeKontoauszugHerunter() {
		if (!akte) return;
		kontoauszugLaedt = true;
		try {
			const res = await apiFetch(`/api/print/kontoauszug/${akte.id}`);
			if (!res.ok) {
				throw new Error((await leseApiFehler(res)) || 'Keine aktiven Ausleihen gefunden');
			}
			const blob = await res.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `Kontoauszug_${akte.vorname}_${akte.nachname}.pdf`;
			a.click();
			URL.revokeObjectURL(url);
		} catch (e) {
			snackbars.addToast(fehlertext(e), 'error');
		} finally {
			kontoauszugLaedt = false;
		}
	}

	// Der Dialog bekommt die Daten der Ausleihe und die Auskunft, ob eine Forderung
	// entsteht: Bei einem Kollegen entsteht keine (Begründung in
	// repository/schaden_melden.go). Der Server entscheidet das selbst; hier steht es, damit
	// der Dialog nicht nach einem Betrag fragt, den niemand fordern wird.
	function oeffneSchaden(book) {
		schadenExemplar = { ...book, ohneForderung: istKollegium(akte) };
		schadenOffen = true;
	}

	async function meldeSchaden(studentId, reason, amount, art) {
		// Festgehalten: Schließt der Dialog vor der Antwort, steht hier ein anderes Buch oder keins.
		const exemplar = schadenExemplar;
		if (!exemplar) return;
		meldetSchaden = true;
		try {
			const res = await apiClient.post(`/api/damage/report`, {
				loan_id: exemplar.ausleihe_id,
				schueler_id: studentId,
				// LaufendeAusleihe liefert die Exemplar-ID als `id` — ein
				// `exemplar_id`-Feld gibt es dort nicht (500: leere UUID).
				copy_id: exemplar.id,
				beschreibung: reason,
				art,
				betrag: amount
			});
			if (res.ok) {
				// Der Bescheid ist ein eigener Schritt: „Bescheid erstellen" an der Gebühren-Karte.
				snackbars.addToast(
					exemplar.ohneForderung
						? 'Verlust/Schaden gebucht, das Exemplar ist ausgesondert. Eine Forderung entsteht bei einem Kollegen nicht.'
						: 'Verlust/Schaden gebucht. Die Forderung steht unter „Gebühren & Schäden"; der Bescheid ist ein eigener Schritt.',
					'success'
				);
				if (schadenExemplar === exemplar) schadenOffen = false;
				if (akte?.id === studentId) ladeAkte(studentId);
			} else {
				snackbars.addToast(await leseApiFehler(res, 'Fehler beim Melden.'), 'error');
			}
		} catch (e) {
			console.error('Schaden melden fehlgeschlagen:', e);
			snackbars.addToast('Netzwerkfehler.', 'error');
		} finally {
			meldetSchaden = false;
		}
	}

	function nachSperre(updated) {
		if (akte?.id === updated.id) {
			// Sofortiges Feedback: die zwei Sperren so, wie der Server sie nach dem Umschalten
			// meldet (Aufheben nimmt auch die der Ehemaligen weg), ohne eigene Formel (sperrStatus.js).
			akte.is_manually_blocked = updated.is_manually_blocked;
			akte.ist_gesperrt = updated.ist_gesperrt;
			// Dann die eine Wahrheit vom Server holen, wie nach Bearbeiten und Foto: Der Sperrgrund
			// steht nicht in der Antwort auf das Sperren (Stufe 1 hinter edit_students, der Grund
			// Stufe 2 hinter view_students), und die Behalten/Löschen-Logik des Servers hier
			// nachzubauen wäre eine zweite Wahrheitsquelle.
			ladeAkte(akte.id);
		}
	}

	return /** @satisfies {object} */ ({
		get akte() {
			return akte;
		},
		set akte(v) {
			akte = v;
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
		get laedt() {
			return laedt;
		},
		get kameraOffen() {
			return kameraOffen;
		},
		set kameraOffen(v) {
			kameraOffen = v;
		},
		get fotoStand() {
			return fotoStand;
		},
		get loeschfrageOffen() {
			return loeschfrageOffen;
		},
		set loeschfrageOffen(v) {
			loeschfrageOffen = v;
		},
		get aktiverReiter() {
			return aktiverReiter;
		},
		set aktiverReiter(v) {
			aktiverReiter = v;
		},
		get bearbeitenOffen() {
			return bearbeitenOffen;
		},
		set bearbeitenOffen(v) {
			bearbeitenOffen = v;
		},
		get schadenOffen() {
			return schadenOffen;
		},
		set schadenOffen(v) {
			schadenOffen = v;
		},
		get sperreOffen() {
			return sperreOffen;
		},
		set sperreOffen(v) {
			sperreOffen = v;
		},
		get schadenExemplar() {
			return schadenExemplar;
		},
		get meldetSchaden() {
			return meldetSchaden;
		},
		get rechnungLaedt() {
			return rechnungLaedt;
		},
		get kontoauszugLaedt() {
			return kontoauszugLaedt;
		},
		ladeAkte,
		schliesseAlleBlaetter,
		nachLoeschen,
		nachBearbeiten,
		schliesseBearbeiten,
		nachFoto,
		ladeRechnungHerunter,
		ladeKontoauszugHerunter,
		oeffneSchaden,
		meldeSchaden,
		nachSperre
	});
}
