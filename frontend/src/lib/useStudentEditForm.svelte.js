import { apiClient } from './apiFetch.js';
import { istKollegium } from './leserArt.js';

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
 * @returns {{ formData: any, saving: boolean, syncData: () => void, save: () => Promise<void> }}
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
		eltern_email: ''
	});

	/**
	 * Syncs the form data with the provided student object.
	 * Call this in an $effect when the student prop changes.
	 */
	function syncData() {
		const student = getStudent();
		if (!student) return;
		formData.vorname = student.vorname || '';
		formData.nachname = student.nachname || '';
		// Eine Zeile ohne Art stammt aus der Zeit vor Migration 123 und ist ein Schüler —
		// dieselbe Vorgabe wie in der Spalte und in leserArtText().
		formData.art = student.art || 'schueler';
		formData.geburtsdatum = student.geburtsdatum ? student.geburtsdatum.slice(0, 10) : '';
		formData.lusd_id = student.lusd_id || '';
		formData.klasse = student.klasse || '';
		formData.barcode_id = student.barcode_id || '';
		formData.abgaenger_jahr = student.abgaenger_jahr?.toString() || '';
		formData.strasse = student.strasse || '';
		formData.hausnummer = student.hausnummer || '';
		formData.plz = student.plz || '';
		formData.ort = student.ort || '';
		formData.eltern_email = student.eltern_email || '';
	}

	/**
	 * Die drei Felder, die einem Kollegen in der Maske VERSCHLOSSEN sind: Klasse,
	 * Abgangsjahr und LUSD-ID. Sie gehen bei ihm gar nicht erst mit.
	 *
	 * Sie dürfen deshalb auch nicht als leerer String mitgehen. Der Server liest den
	 * leeren String bei den Pflichtfeldern als „räum das weg" und antwortet „Klasse darf
	 * nicht leer sein." — genau daran scheiterte das Speichern eines Kollegen, bevor
	 * dieses Formular ihn überhaupt anbot. Weggelassen heißt `undefined`, wird von
	 * JSON.stringify fallengelassen und kommt im Backend als nil an: Spalte in Ruhe
	 * lassen.
	 *
	 * Die Ausweisnummer steht in der Mitte: Sie gehört jedem Leser, aber ein Kollege darf
	 * ohne dastehen (Migration 123). Leer lassen heißt bei ihm „hat noch keine", nicht
	 * „nimm ihm seine" — der Server weist ein leeres Pflichtfeld sonst mit 400 ab.
	 * @returns {Record<string, unknown>}
	 */
	function schulfelder() {
		const kollege = istKollegium({ art: formData.art });
		// Die Ausweisnummer steht in der Mitte: Sie gehört jedem Leser, aber ein Kollege
		// darf ohne dastehen (Migration 123). Leer lassen heißt bei ihm „hat noch keine",
		// nicht „nimm ihm seine" — der Server weist ein leeres Pflichtfeld sonst mit 400 ab.
		const ausweisLeer = !formData.barcode_id?.trim();
		const ausweis = kollege && ausweisLeer ? {} : { barcode_id: formData.barcode_id };
		if (kollege) return ausweis;
		return {
			...ausweis,
			lusd_id: formData.lusd_id,
			klasse: formData.klasse,
			abgaenger_jahr: formData.abgaenger_jahr ? Number.parseInt(formData.abgaenger_jahr, 10) : null
		};
	}

	/**
	 * Submits the form data to the server.
	 */
	async function save() {
		const student = getStudent();
		if (!student?.id) {
			showSnackbar('Kein Schüler ausgewählt.', 'error');
			return;
		}
		saving = true;
		try {
			// Geräumte Felder gehen als LEERER STRING raus, nicht als null.
			//
			// Der Unterschied ist nicht kosmetisch: Im Backend sind diese Felder *string,
			// und JSON-null landet dort als nil — die Bedeutung von nil ist "nicht
			// mitgeschickt, Spalte in Ruhe lassen". Bis zum 23.08.2026 stand hier überall
			// `|| null`; wer eine Adresse oder die Eltern-Mail löschte und speicherte,
			// bekam "Änderungen gespeichert" zu sehen, und beim nächsten Öffnen der Akte
			// stand der alte Wert wieder da. Löschen war über dieses Formular schlicht
			// nicht möglich.
			//
			// Für die Pflichtfelder (Vor-/Nachname, Klasse, Ausweisnummer) ist der leere
			// String ebenfalls die richtige Nachricht: Der Server lehnt ihn jetzt mit einer
			// Begründung ab, statt still nichts zu tun.
			//
			// EINE Ausnahme: geburtsdatum bleibt bei `|| null`. Der Server verweigert das
			// Leeren dieses Feldes (es ist der LUSD-Schlüssel) — schickte das Formular hier
			// den leeren String, bekäme jeder Altdatensatz OHNE Geburtsdatum bei jedem
			// Speichern "Geburtsdatum kann nicht geleert werden" zu sehen und liesse sich
			// gar nicht mehr bearbeiten. Es war nie gesetzt, also wird auch nichts geleert.
			// Preis dieser Entscheidung, offen benannt: Wer ein GESETZTES Geburtsdatum im
			// Feld räumt und speichert, bekommt weiterhin ein stilles No-op — der alte Wert
			// steht beim nächsten Öffnen wieder da. Löschbar ist es ohnehin nicht.
			const payload = {
				vorname: formData.vorname,
				nachname: formData.nachname,
				art: formData.art,
				geburtsdatum: formData.geburtsdatum || null,
				strasse: formData.strasse,
				hausnummer: formData.hausnummer,
				plz: formData.plz,
				ort: formData.ort,
				eltern_email: formData.eltern_email,
				...schulfelder()
			};
			const res = await apiClient.patch(`/api/schueler/${student.id}`, payload);
			if (!res.ok) {
				const data = await res.json().catch(() => ({}));
				throw new Error(data.error || 'Speichern fehlgeschlagen');
			}
			showSnackbar('Änderungen gespeichert.', 'success');
			onSave();
		} catch (e) {
			showSnackbar(e instanceof Error ? e.message : String(e), 'error');
		} finally {
			saving = false;
		}
	}

	return {
		get formData() {
			return formData;
		},
		get saving() {
			return saving;
		},
		syncData,
		save
	};
}
