import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import BestandImport from './BestandImport.svelte';
import { serverAntwort } from '../../serverAntwort.js';

// Startet der Server gerade neu, antwortet der Proxy mit 502 ohne Rumpf. Der Import las die
// Antwort als JSON, bevor er nach res.ok fragte, und zeigte „Unexpected end of JSON input".
describe('BestandImport: abgelehnter Import', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	/** @param {any} antwort */
	async function importiere(antwort) {
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort);
		const screen = render(BestandImport);
		const feld = /** @type {HTMLInputElement} */ (screen.getByTestId('bestandimport-datei'));
		const datei = new File(['Titel;Autor;Verlag'], 'bestand.csv', { type: 'text/csv' });
		// bind:files schreibt zurück (nach dem Import leer), das Feld muss es annehmen.
		Object.defineProperty(feld, 'files', { value: [datei], writable: true, configurable: true });
		await fireEvent.change(feld);
		await fireEvent.click(screen.getByRole('button', { name: 'Bestand importieren' }));
		return screen;
	}

	it('nennt ohne Rumpf, etwa bei 502 vom Proxy, die Vorgabe', async () => {
		const screen = await importiere(serverAntwort(502));
		const ergebnis = await screen.findByTestId('bestandimport-ergebnis');
		expect(ergebnis.textContent?.trim()).toBe('Bestands-Import fehlgeschlagen');
	});

	it('nennt die Meldung des Servers', async () => {
		const meldung = 'Zeile 3: Die Nummer ist die eines Leserausweises.';
		const screen = await importiere(serverAntwort(400, { error: meldung }));
		const ergebnis = await screen.findByTestId('bestandimport-ergebnis');
		expect(ergebnis.textContent?.trim()).toBe(meldung);
	});

	// Das Ergebnis erscheint nach dem Klick, ohne dass der Fokus zu ihm wandert: Ohne Rolle liest
	// ein Bildschirmleser es nicht vor. Fehler als alert, Erfolg als status.
	it('lässt Fehler und Erfolg vorlesen', async () => {
		let screen = await importiere(serverAntwort(400, { error: 'Zeile 3 ist leer.' }));
		expect((await screen.findByRole('alert')).textContent?.trim()).toBe('Zeile 3 ist leer.');
		screen.unmount();

		screen = await importiere(
			serverAntwort(200, { new_titles_count: 2, imported_copies_count: 5 })
		);
		expect((await screen.findByRole('status')).textContent).toContain('2 neue Titel');
	});
});
