import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({
	snackbars: { addToast: vi.fn() }
}));

import { apiFetch } from '../../apiFetch.js';
import { snackbars } from '../../stores/snackbars.svelte.js';
import LitteraImport from './LitteraImport.svelte';
import { serverAntwort } from '../../serverAntwort.js';

// Der Katalog-Import aus Littera ist die Übernahme des Altbestands: Ein Erfolg nennt seine Zahlen,
// ein Fehlschlag den Grund. Vertauscht, hieße ein gelungener Import gescheitert und lüde zum
// zweiten Lauf ein, ein gescheiterter hieße gelungen.
describe('LitteraImport: Ergebnis in der Meldung', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(snackbars.addToast).mockReset();
	});

	/** @param {any} antwort */
	async function importiere(antwort) {
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort);
		const screen = render(LitteraImport);
		const feld = /** @type {HTMLInputElement} */ (
			screen.container.querySelector('input[type="file"]')
		);
		const datei = new File(['Titel;Autor'], 'bestand.csv', { type: 'text/csv' });
		Object.defineProperty(feld, 'files', { value: [datei], configurable: true });
		await fireEvent.change(feld);
		await fireEvent.click(screen.getByRole('button', { name: 'Import starten' }));
	}

	it('nennt nach dem Erfolg die Zahlen', async () => {
		await importiere(
			serverAntwort(200, { type: 'csv', new_titles_count: 2, imported_copies_count: 5 })
		);
		await vi.waitFor(() =>
			expect(snackbars.addToast).toHaveBeenCalledWith(
				'2 neue Titel und 5 Exemplare angelegt',
				'success'
			)
		);
	});

	it('nennt die Meldung des Servers', async () => {
		const meldung = 'Die Datei ist kein Katalog aus Littera.';
		await importiere(serverAntwort(400, { error: meldung }));
		await vi.waitFor(() => expect(snackbars.addToast).toHaveBeenCalledWith(meldung, 'error'));
		expect(snackbars.addToast).not.toHaveBeenCalledWith(expect.anything(), 'success');
	});
});
