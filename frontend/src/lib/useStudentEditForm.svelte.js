import { apiClient } from './apiFetch.js';

/**
 * Custom hook to manage the state and submission of the student edit form.
 * @param {Object} props
 * @param {any} props.student - The student object to initialize form data
 * @param {() => void} props.onSave - Callback when the save is successful
 * @param {(msg: string, type: 'success' | 'error') => void} props.showSnackbar - Callback to show notifications
 * @returns {{ formData: any, saving: boolean, syncData: () => void, save: () => Promise<void> }}
 */
export function useStudentEditForm({ student, onSave, showSnackbar }) {
	let saving = $state(false);

	let formData = $state({
		vorname: '',
		nachname: '',
		geburtsdatum: '',
		lusd_id: '',
		klasse: '',
		barcode_id: '',
		abgaenger_jahr: '',
		status: '',
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
		if (!student) return;
		formData.vorname = student.vorname || '';
		formData.nachname = student.nachname || '';
		formData.geburtsdatum = student.geburtsdatum ? student.geburtsdatum.slice(0, 10) : '';
		formData.lusd_id = student.lusd_id || '';
		formData.klasse = student.klasse || '';
		formData.barcode_id = student.barcode_id || '';
		formData.abgaenger_jahr = student.abgaenger_jahr?.toString() || '';
		formData.status = student.status || '';
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
		saving = true;
		try {
			const payload = {
				vorname: formData.vorname || null,
				nachname: formData.nachname || null,
				geburtsdatum: formData.geburtsdatum || null,
				lusd_id: formData.lusd_id || null,
				klasse: formData.klasse || null,
				barcode_id: formData.barcode_id || null,
				abgaenger_jahr: formData.abgaenger_jahr ? parseInt(formData.abgaenger_jahr, 10) : null,
				status: formData.status || null,
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
			showSnackbar(e?.message || String(e), 'error');
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
