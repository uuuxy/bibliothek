import { apiClient } from './apiFetch.js';
import { istKollegium, artMitKonto } from './leserArt.js';
import { fehlertext } from './utils/fehlertext.js';
import { nurGeaendertes } from './utils/geaendert.js';

/**
 * Custom hook to manage the state and submission of the student edit form.
 *
 * `getStudent` ist bewusst ein GETTER und kein Wert. Vorher stand hier
 * `{ student, … }`: Das Destrukturieren nimmt einen Schnappschuss des Props, und der
 * Hook arbeitete danach dauerhaft mit dem Schüler, der beim Aufbau der Komponente
 * gerade aktuell war. Zwei Folgen, beide still:
 *
 *   1. `syncData()` füllte das Formular erneut mit den ALTEN Daten, wenn der Aufrufer
 *      dieselbe Komponente mit einem anderen Schüler weiterverwendete.
 *   2. `save()` schickte das PATCH an `/api/schueler/<alte-id>` — die Eingaben landeten
 *      am falschen Datensatz, mit Erfolgsmeldung.
 *
 * Der Svelte-Compiler warnte darauf hin ("This reference only captures the initial
 * value of `student`"). Über einen Getter liest der Hook bei jedem Zugriff den
 * aktuellen Wert, und ein `$effect`, der `syncData()` aufruft, verfolgt das Prop
 * dadurch auch richtig.
 *
 * @param {Object} props
 * @param {() => any} props.getStudent - Liefert den aktuell bearbeiteten Schüler
 * @param {() => void} props.onSave - Callback when the save is successful
 * @param {(msg: string, type: 'success' | 'error') => void} props.showSnackbar - Callback to show notifications
 * @returns {{ formData: any, saving: boolean, kontoVorhanden: boolean, syncData: () => void, save: () => Promise<void> }}
 */
export function useStudentEditForm({ getStudent, onSave, showSnackbar }) {
	let saving = $state(false);

	let formData = $state({
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
	// Feld eine Anzeige (geändert wird in der Benutzerverwaltung, api/student_schul_email.go).
	let kontoVorhanden = $state(false);

	// Die Nutzlast, wie die Maske sie direkt nach dem Füllen schickte. Das Speichern schickt
	// nur, was davon abweicht.
	/** @type {Record<string, unknown>} */
	let geladen = {};

	/**
	 * Syncs the form data with the provided student object.
	 * Call this in an $effect when the student prop changes.
	 */
	function syncData() {
		const student = getStudent();
		if (!student) return;
		// Erst ein schlichtes Objekt: Der Aufrufer ruft syncData in einem $effect, und wer dort
		// formData läse, füllte die Maske bei jedem Tastendruck neu.
		const werte = {
			vorname: student.vorname || '',
			nachname: student.nachname || '',
			// Eine Zeile ohne Art ist ein Schüler, wie in der Spalte und in leserArtText().
			art: student.art || 'schueler',
			geburtsdatum: student.geburtsdatum ? student.geburtsdatum.slice(0, 10) : '',
			lusd_id: student.lusd_id || '',
			klasse: student.klasse || '',
			barcode_id: student.barcode_id || '',
			// Die Abfrage liefert für eine leere Spalte die 0, und "0" wäre ein wahrer String.
			abgaenger_jahr: student.abgaenger_jahr ? String(student.abgaenger_jahr) : '',
			strasse: student.strasse || '',
			hausnummer: student.hausnummer || '',
			plz: student.plz || '',
			ort: student.ort || '',
			eltern_email: student.eltern_email || '',
			email: student.email || ''
		};
		Object.assign(formData, werte);
		kontoVorhanden = !!student.email;
		geladen = nutzlast(werte, student, !!student.email);
	}

	/**
	 * Klasse, Abgangsjahr und LUSD-ID gehören dem Schüler, die Schul-Adresse dem Kollegium:
	 * Die Felder der anderen Seite gehen nicht mit, auch nicht leer. Ein leerer String hieße
	 * am Server „räum das weg" und käme als 400 zurück.
	 * @param {any} daten Werte der Maske
	 * @param {any} student der geladene Leser
	 * @param {boolean} hatKonto
	 * @returns {Record<string, unknown>}
	 */
	function schulfelder(daten, student, hatKonto) {
		const ausweis = { barcode_id: daten.barcode_id };
		if (istKollegium({ art: daten.art })) {
			// Ohne Zugang zur Art und ohne Konto ist das Feld verschlossen: Eine davor getippte
			// Adresse geht nicht mit.
			return { ...ausweis, email: artMitKonto(daten.art) || hatKonto ? daten.email : '' };
		}
		// Das Abgangsjahr geht nicht mit, wenn die Klasse wechselt und niemand es angefasst
		// hat: Dann leitet der Server es aus der neuen Klasse ab (calculateAbgaengerJahr).
		const geladenesJahr = student?.abgaenger_jahr ? String(student.abgaenger_jahr) : '';
		const jahrAngefasst = daten.abgaenger_jahr !== geladenesJahr;
		const klasseGeaendert = daten.klasse !== (student?.klasse || '');

		/** @type {Record<string, unknown>} */
		const felder = { ...ausweis, lusd_id: daten.lusd_id, klasse: daten.klasse };
		if (!(klasseGeaendert && !jahrAngefasst)) {
			felder.abgaenger_jahr = daten.abgaenger_jahr
				? Number.parseInt(daten.abgaenger_jahr, 10)
				: null;
		}
		return felder;
	}

	/**
	 * Die Maske in der Form, in der sie an den Server geht. Geräumte Felder sind leere Strings:
	 * JSON-null hieße dort „nicht mitgeschickt", und Löschen wäre nicht möglich. Nur das
	 * Geburtsdatum geht leer als null hinaus, der Server lässt es sich nicht leeren.
	 * @param {any} daten
	 * @param {any} student
	 * @param {boolean} hatKonto
	 * @returns {Record<string, unknown>}
	 */
	function nutzlast(daten, student, hatKonto) {
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
			...schulfelder(daten, student, hatKonto)
		};
	}

	/**
	 * Speichert die Felder, die seit dem Laden der Akte geändert wurden. Der Server schreibt
	 * jedes Feld, das der Rumpf nennt: Mit allen Feldern schriebe die Maske zurück, was ein
	 * anderer Platz, der LUSD-Import oder die Versetzung inzwischen geändert hat.
	 */
	async function save() {
		const student = getStudent();
		if (!student?.id) {
			showSnackbar('Kein Schüler ausgewählt.', 'error');
			return;
		}
		const payload = nurGeaendertes(geladen, nutzlast(formData, student, kontoVorhanden));
		// Ohne Änderung gibt es nichts zu schicken; der Server wiese den leeren Rumpf ab.
		if (Object.keys(payload).length === 0) {
			showSnackbar('Änderungen gespeichert.', 'success');
			onSave();
			return;
		}
		saving = true;
		try {
			const res = await apiClient.patch(`/api/schueler/${student.id}`, payload);
			if (!res.ok) {
				const data = await res.json().catch(() => ({}));
				throw new Error(data.error || 'Speichern fehlgeschlagen');
			}
			showSnackbar('Änderungen gespeichert.', 'success');
			onSave();
		} catch (e) {
			showSnackbar(fehlertext(e), 'error');
		} finally {
			saving = false;
		}
	}

	return {
		get formData() {
			return formData;
		},
		get kontoVorhanden() {
			return kontoVorhanden;
		},
		get saving() {
			return saving;
		},
		syncData,
		save
	};
}
