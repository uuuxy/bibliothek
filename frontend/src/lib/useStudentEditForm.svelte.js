import { apiClient } from './apiFetch.js';

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
			const payload = {
				vorname: formData.vorname || null,
				nachname: formData.nachname || null,
				geburtsdatum: formData.geburtsdatum || null,
				lusd_id: formData.lusd_id || null,
				klasse: formData.klasse || null,
				barcode_id: formData.barcode_id || null,
				abgaenger_jahr: formData.abgaenger_jahr
					? Number.parseInt(formData.abgaenger_jahr, 10)
					: null,
				strasse: formData.strasse || null,
				hausnummer: formData.hausnummer || null,
				plz: formData.plz || null,
				ort: formData.ort || null,
				eltern_email: formData.eltern_email || null
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
