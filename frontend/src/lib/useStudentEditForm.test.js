import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('./apiFetch.js', () => ({
	apiClient: { patch: vi.fn() }
}));

import { apiClient } from './apiFetch.js';
import { useStudentEditForm } from './useStudentEditForm.svelte.js';

const patchMock = vi.mocked(apiClient.patch);

/**
 * Ein geräumtes Feld muss den Server als LEERER STRING erreichen, nicht als null.
 *
 * Der Hintergrund steht im Backend (api/student_update.go): Die Stammdatenfelder sind
 * dort *string, und JSON-null bedeutet nil — "nicht mitgeschickt, Spalte in Ruhe
 * lassen". Bis zum 23.08.2026 baute dieses Formular seine Nutzlast als
 * `strasse: formData.strasse || null`; wer eine Adresse oder die Eltern-Mail löschte,
 * bekam "Änderungen gespeichert" zu sehen, während in der Datenbank der alte Wert
 * stehen blieb. Betroffen war genau das, dessen Entfernung jemand verlangen kann.
 *
 * Der Test steht hier und nicht nur als PG-Test im Backend, weil der Rückfall auf
 * dieser Seite passiert: Ein `|| null` ist eine Zeile, die beim nächsten Refactoring
 * harmlos aussieht. Das Gegenstück (Pflichtfelder lassen sich nicht leeren) sichert
 * api/schueler_feld_leeren_pg_test.go ab.
 */
describe('useStudentEditForm.save', () => {
	beforeEach(() => vi.clearAllMocks());

	const schueler = {
		id: 'abc',
		vorname: 'Mia',
		nachname: 'Muster',
		geburtsdatum: '2012-04-05',
		klasse: '7a',
		barcode_id: 'S-1',
		strasse: 'Hauptstr',
		hausnummer: '12',
		plz: '60311',
		ort: 'Frankfurt',
		eltern_email: 'eltern@example.org'
	};

	function baueFormular() {
		const hook = useStudentEditForm({
			getStudent: () => schueler,
			onSave: () => {},
			showSnackbar: () => {}
		});
		hook.syncData();
		return hook;
	}

	it('schickt geräumte Stammdatenfelder als leeren String, nicht als null', async () => {
		patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
		const hook = baueFormular();

		for (const feld of ['strasse', 'hausnummer', 'plz', 'ort', 'eltern_email']) {
			hook.formData[feld] = '';
		}
		await hook.save();

		const [, payload] = patchMock.mock.calls[0];
		for (const feld of ['strasse', 'hausnummer', 'plz', 'ort', 'eltern_email']) {
			expect(payload[feld], `${feld} muss als '' rausgehen — null hiesse "nicht anfassen"`).toBe(
				''
			);
		}
	});

	it('schickt geleerte Pflichtfelder ebenfalls als leeren String — der Server lehnt sie ab', async () => {
		patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
		const hook = baueFormular();

		hook.formData.vorname = '';
		hook.formData.klasse = '';
		await hook.save();

		const [, payload] = patchMock.mock.calls[0];
		expect(payload.vorname).toBe('');
		expect(payload.klasse).toBe('');
	});

	it('lässt ein nie gesetztes Geburtsdatum als null durch (Altdaten bleiben speicherbar)', async () => {
		patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
		const hook = baueFormular();

		hook.formData.geburtsdatum = '';
		await hook.save();

		const [, payload] = patchMock.mock.calls[0];
		expect(payload.geburtsdatum).toBeNull();
	});
});
