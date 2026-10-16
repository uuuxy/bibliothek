import { apiClient } from './apiFetch.js';
import { istKollegium, artMitKonto } from './leserArt.js';
import { fehlertext } from './utils/fehlertext.js';
import { nurGeaendertes } from './utils/geaendert.js';

/**
 * Klasse, Abgangsjahr und LUSD-ID gehören dem Schüler, die Schul-Adresse dem Kollegium:
 * Die Felder der anderen Seite gehen nicht mit, auch nicht leer. Ein leerer String hieße
 * am Server „räum das weg" und käme als 400 zurück.
 * @param {any} daten Werte der Maske
 * @param {any} leser der geladene Leser
 * @param {boolean} hatKonto
 * @returns {Record<string, unknown>}
 */
function schulfelder(daten, leser, hatKonto) {
	const ausweis = { barcode_id: daten.barcode_id };
	if (istKollegium({ art: daten.art })) {
		// Ohne Zugang zur Art und ohne Konto ist das Feld verschlossen: Eine davor getippte
		// Adresse geht nicht mit.
		return { ...ausweis, email: artMitKonto(daten.art) || hatKonto ? daten.email : '' };
	}
	// Das Abgangsjahr geht nicht mit, wenn die Klasse wechselt und niemand es angefasst
	// hat: Dann leitet der Server es aus der neuen Klasse ab (repository.AbgaengerJahr).
	const geladenesJahr = leser?.abgaenger_jahr ? String(leser.abgaenger_jahr) : '';
	const jahrAngefasst = daten.abgaenger_jahr !== geladenesJahr;
	const klasseGeaendert = daten.klasse !== (leser?.klasse || '');

	/** @type {Record<string, unknown>} */
	const felder = { ...ausweis, lusd_id: daten.lusd_id, klasse: daten.klasse };
	if (!(klasseGeaendert && !jahrAngefasst)) {
		felder.abgaenger_jahr = daten.abgaenger_jahr ? Number.parseInt(daten.abgaenger_jahr, 10) : null;
	}
	return felder;
}

/**
 * Die Maske in der Form, in der sie an den Server geht. Geräumte Felder sind leere Strings:
 * JSON-null hieße dort „nicht mitgeschickt", und Löschen wäre nicht möglich. Nur das
 * Geburtsdatum geht leer als null hinaus, der Server lässt es sich nicht leeren.
 * @param {any} daten
 * @param {any} leser
 * @param {boolean} hatKonto
 * @returns {Record<string, unknown>}
 */
function nutzlast(daten, leser, hatKonto) {
	return {
		vorname: daten.vorname,
		nachname: daten.nachname,
		art: daten.art,
		geburtsdatum: daten.geburtsdatum || null,
		strasse: daten.strasse,
		hausnummer: daten.hausnummer,
		plz: daten.plz,
		ort: daten.ort,
		eltern_email: daten.eltern_email,
		...schulfelder(daten, leser, hatKonto)
	};
}

/**
 * Custom hook to manage the state and submission of the student edit form.
 *
 * `getStudent` ist bewusst ein GETTER und kein Wert. Vorher stand hier
 * `{ student, … }`: Das Destrukturieren nimmt einen Schnappschuss des Props, und der
 * Hook arbeitete danach dauerhaft mit dem Schüler, der beim Aufbau der Komponente
 * gerade aktuell war. Zwei Folgen, beide still:
 *
 *   1. `uebernimmLeser()` füllte das Formular erneut mit den ALTEN Daten, wenn der Aufrufer
 *      dieselbe Komponente mit einem anderen Schüler weiterverwendete.
 *   2. `save()` schickte das PATCH an `/api/schueler/<alte-id>` — die Eingaben landeten
 *      am falschen Datensatz, mit Erfolgsmeldung.
 *
 * Der Svelte-Compiler warnte darauf hin ("This reference only captures the initial
 * value of `student`"). Über einen Getter liest der Hook bei jedem Zugriff den
 * aktuellen Wert, und ein `$effect`, der `uebernimmLeser()` aufruft, verfolgt das Prop
 * dadurch auch richtig.
 *
 * @param {Object} props
 * @param {() => any} props.getStudent - Liefert den aktuell bearbeiteten Schüler
 * @param {() => void} props.onSave - Callback when the save is successful
 * @param {(msg: string, type: 'success' | 'error') => void} props.showSnackbar - Callback to show notifications
 * @returns {{ eingaben: any, speichert: boolean, kontoVorhanden: boolean, uebernimmLeser: () => void, speichere: () => Promise<void> }}
 */
export function useLeserakteFormular({ getStudent, onSave, showSnackbar }) {
	let speichert = $state(false);

	let eingaben = $state({
		vorname: '',
		nachname: '',
		art: 'schueler',
		geburtsdatum: '',
		lusd_id: '',
		klasse: '',
		barcode_id: '',
		abgaenger_jahr: '',
		strasse: '',
		hausnummer: '',
		plz: '',
		ort: '',
		eltern_email: '',
		// Die SCHUL-Adresse (benutzer.email), nicht die der Eltern. Sie steht nicht an der
		// Leserzeile, sondern am Konto — die Akte zeigt und trägt sie trotzdem, weil sie
		// die Kennung ist, an der die Anmeldung die Person erkennt (wo ein Zugang zur Art gehört).
		email: ''
	});

	// Ob zu dieser Leserzeile bereits ein Konto besteht. Es ist nicht dasselbe wie „das
	// Feld ist ausgefüllt": Nachgetragen wird nur in eine LEERE Adresse, danach ist das
	// Feld eine Anzeige (geändert wird in der Benutzerverwaltung, api/leser/leser_schul_email.go).
	let kontoVorhanden = $state(false);

	// Die Nutzlast, wie die Maske sie direkt nach dem Füllen schickte. Das Speichern schickt
	// nur, was davon abweicht.
	/** @type {Record<string, unknown>} */
	let geladen = {};

	/**
	 * Syncs the form data with the provided student object.
	 * Call this in an $effect when the student prop changes.
	 */
	function uebernimmLeser() {
		const leser = getStudent();
		if (!leser) return;
		// Erst ein schlichtes Objekt: Der Aufrufer ruft uebernimmLeser in einem $effect, und wer dort
		// eingaben läse, füllte die Maske bei jedem Tastendruck neu.
		const werte = {
			vorname: leser.vorname || '',
			nachname: leser.nachname || '',
			// Eine Zeile ohne Art ist ein Schüler, wie in der Spalte und in leserArtText().
			art: leser.art || 'schueler',
			geburtsdatum: leser.geburtsdatum ? leser.geburtsdatum.slice(0, 10) : '',
			lusd_id: leser.lusd_id || '',
			klasse: leser.klasse || '',
			barcode_id: leser.barcode_id || '',
			// Die Abfrage liefert für eine leere Spalte die 0, und "0" wäre ein wahrer String.
			abgaenger_jahr: leser.abgaenger_jahr ? String(leser.abgaenger_jahr) : '',
			strasse: leser.strasse || '',
			hausnummer: leser.hausnummer || '',
			plz: leser.plz || '',
			ort: leser.ort || '',
			eltern_email: leser.eltern_email || '',
			email: leser.email || ''
		};
		Object.assign(eingaben, werte);
		kontoVorhanden = !!leser.email;
		geladen = nutzlast(werte, leser, !!leser.email);
	}

	/**
	 * Speichert die Felder, die seit dem Laden der Akte geändert wurden. Der Server schreibt
	 * jedes Feld, das der Rumpf nennt: Mit allen Feldern schriebe die Maske zurück, was ein
	 * anderer Platz, der LUSD-Import oder die Versetzung inzwischen geändert hat.
	 */
	async function speichere() {
		const leser = getStudent();
		if (!leser?.id) {
			showSnackbar('Kein Schüler ausgewählt.', 'error');
			return;
		}
		const payload = nurGeaendertes(geladen, nutzlast(eingaben, leser, kontoVorhanden));
		// Ohne Änderung gibt es nichts zu schicken; der Server wiese den leeren Rumpf ab.
		if (Object.keys(payload).length === 0) {
			showSnackbar('Änderungen gespeichert.', 'success');
			onSave();
			return;
		}
		speichert = true;
		try {
			const res = await apiClient.patch(`/api/schueler/${leser.id}`, payload);
			if (!res.ok) {
				const antwort = await res.json().catch(() => ({}));
				throw new Error(antwort.error || 'Speichern fehlgeschlagen');
			}
			showSnackbar('Änderungen gespeichert.', 'success');
			onSave();
		} catch (e) {
			showSnackbar(fehlertext(e), 'error');
		} finally {
			speichert = false;
		}
	}

	return /** @satisfies {object} */ ({
		get eingaben() {
			return eingaben;
		},
		get kontoVorhanden() {
			return kontoVorhanden;
		},
		get speichert() {
			return speichert;
		},
		uebernimmLeser,
		speichere
	});
}
