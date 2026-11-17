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
import BestandImport from './BestandImport.svelte';
import { serverAntwort } from '../../serverAntwort.js';

// Startet der Server gerade neu, antwortet der Proxy mit 502 ohne Rumpf. Der Import las die
// Antwort als JSON, bevor er nach res.ok fragte, und zeigte „Unexpected end of JSON input".
// Das Ergebnis steht in der Meldung am unteren Rand; vorgelesen wird es dort (SnackbarAnzeige).
describe('BestandImport: Ergebnis in der Meldung', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(snackbars.addToast).mockReset();
	});

	/** @param {any} antwort */
	async function importiere(antwort) {
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort);
		const screen = render(BestandImport);
		const feld = /** @type {HTMLInputElement} */ (screen.getByTestId('bestandimport-datei'));
		const datei = new File(['Titel;Autor;Verlag'], 'bestand.csv', { type: 'text/csv' });
		Object.defineProperty(feld, 'files', { value: [datei], configurable: true });
		await fireEvent.change(feld);
		await fireEvent.click(screen.getByRole('button', { name: 'Bestand importieren' }));
		return screen;
	}

	it('nennt ohne Rumpf, etwa bei 502 vom Proxy, die Vorgabe', async () => {
		await importiere(serverAntwort(502));
		await vi.waitFor(() =>
			expect(snackbars.addToast).toHaveBeenCalledWith('Bestands-Import fehlgeschlagen', 'error')
		);
	});

	it('nennt die Meldung des Servers', async () => {
		const meldung = 'Zeile 3: Die Nummer ist die eines Leserausweises.';
		await importiere(serverAntwort(400, { error: meldung }));
		await vi.waitFor(() => expect(snackbars.addToast).toHaveBeenCalledWith(meldung, 'error'));
	});

	it('nennt nach dem Erfolg die Zahlen, in der Einzahl bei einem Titel', async () => {
		await importiere(serverAntwort(200, { new_titles_count: 1, imported_copies_count: 5 }));
		await vi.waitFor(() =>
			expect(snackbars.addToast).toHaveBeenCalledWith(
				'Bestand übernommen: 1 neuer Titel, 5 Exemplare.',
				'success'
			)
		);
	});
});
