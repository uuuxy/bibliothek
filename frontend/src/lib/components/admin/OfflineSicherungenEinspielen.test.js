import { describe, it, expect, vi, beforeEach } from 'vitest';
import { fireEvent, render } from '@testing-library/svelte';

const { importiere, toast } = vi.hoisted(() => ({ importiere: vi.fn(), toast: vi.fn() }));
vi.mock('../../stores/offlineSync.svelte.js', () => ({
	offlineSync: { importiereWarteschlange: importiere, wartendeAnzahl: 0 }
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: toast } }));

import OfflineSicherungenEinspielen from './OfflineSicherungenEinspielen.svelte';

/** Wählt Dateien im versteckten Eingabefeld. @param {HTMLElement} container @param {string[]} namen */
async function waehle(container, namen) {
	const feld = /** @type {HTMLInputElement} */ (container.querySelector('input[type="file"]'));
	const dateien = namen.map((n) => new File(['{}'], n, { type: 'application/json' }));
	Object.defineProperty(feld, 'files', { value: dateien, configurable: true });
	await fireEvent.change(feld);
	await vi.waitFor(() => expect(importiere).toHaveBeenCalledTimes(namen.length));
}

// Der grüne Kasten meldet übernommene Vorgänge. Scheitert jede Datei, stand dort „0 … aus 0
// Datei(en) übernommen“ und „Sie werden jetzt an den Server übertragen“, während die
// Fehlermeldung am Rand erschien.
describe('Offline-Sicherungen einspielen: der Bericht', () => {
	beforeEach(() => {
		importiere.mockReset();
		toast.mockReset();
	});

	it('zeigt keinen Erfolg, wenn keine Datei gelesen wurde', async () => {
		importiere.mockRejectedValue(new Error('Die Datei ist keine Sicherung'));
		const { container, queryByText } = render(OfflineSicherungenEinspielen);
		await waehle(container, ['kaputt.json']);
		await vi.waitFor(() =>
			expect(toast).toHaveBeenCalledWith(expect.stringContaining('kaputt.json'), 'error')
		);
		expect(queryByText(/übernommen/)).toBeNull();
	});

	it('nennt die gelesenen Dateien, wenn eine von zweien scheitert', async () => {
		importiere.mockResolvedValueOnce(3).mockRejectedValueOnce(new Error('unlesbar'));
		const { container, findByText } = render(OfflineSicherungenEinspielen);
		await waehle(container, ['kiosk1.json', 'kaputt.json']);
		expect(await findByText(/3 Vorgang\/Vorgänge aus 1 Datei\(en\) übernommen/)).toBeTruthy();
		expect(toast).toHaveBeenCalledWith(expect.stringContaining('kaputt.json'), 'error');
	});
});
