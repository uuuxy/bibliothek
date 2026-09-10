import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', () => ({ apiFetch: vi.fn() }));
import { apiFetch } from '../../apiFetch.js';
import LusdImportView from './LusdImportView.svelte';

// Die Massenabgang-Bremse (409 vom Server) ist eine Bestätigung für GENAU EINE Datei.
//
// Bestands-Durchgang 10.09.2026: needsGraduateConfirm wurde nur in resetFlow() und nach
// einem Erfolg zurückgesetzt. Wer nach der Bremse eine ANDERE Datei direkt über die
// Ablagefläche wählte (sie bleibt im Vorschau-Schritt sichtbar), bekam nach deren
// Vorschau sofort den roten Knopf „Massenabgang bestätigen" — der erste Klick schickte
// confirm_graduates=true für eine Datei, die der Server nie gebremst hatte. Ein
// Teilexport machte so Hunderte zu Abgängern.

/** @type {any} */
const vorschau = {
	modus: 'lusd_id',
	total_csv_records: 10,
	active_db_students: 10,
	skipped_no_id: 0,
	dubletten_in_datei: 0,
	graduates: [],
	umbenennungen: []
};
/** @param {boolean} ok @param {number} status @param {any} body */
const antwort = (ok, status, body) => ({ ok, status, json: async () => body });

/** @param {HTMLElement} container @param {string} name */
async function waehleDatei(container, name) {
	const input = /** @type {HTMLInputElement} */ (container.querySelector('input[type="file"]'));
	const datei = new File(['vorname;nachname;klasse'], name, { type: 'text/csv' });
	Object.defineProperty(input, 'files', { value: [datei], configurable: true });
	await fireEvent.change(input);
}

describe('LusdImportView: Massenabgang-Bestätigung', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	it('gilt nicht für eine danach gewählte andere Datei', async () => {
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(/** @type {any} */ (antwort(true, 200, vorschau)))
			.mockResolvedValueOnce(
				/** @type {any} */ (antwort(false, 409, { error: 'Auffällig viele Abgänger' }))
			)
			.mockResolvedValueOnce(/** @type {any} */ (antwort(true, 200, vorschau)));
		const { container, getByRole, findByRole, queryByRole } = render(LusdImportView);

		await waehleDatei(container, 'teilexport.csv');
		await fireEvent.click(getByRole('button', { name: /Vorschau laden/ }));
		await fireEvent.click(await findByRole('button', { name: /Import finalisieren/ }));
		await findByRole('button', { name: /Massenabgang/ });

		await waehleDatei(container, 'andere.csv');
		await fireEvent.click(getByRole('button', { name: /Vorschau laden/ }));
		await findByRole('button', { name: /Import finalisieren/ });
		expect(queryByRole('button', { name: /Massenabgang/ })).toBeNull();
	});
});
