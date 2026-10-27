import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../stores/snackbars.svelte.js', () => ({
	snackbars: { addToast: vi.fn() }
}));

// leseApiFehler bleibt echt (ARCHITEKTUR.md 8.14).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import { serverAntwort } from '../../serverAntwort.js';
import SchuelerDokumente from './SchuelerDokumente.svelte';

const apiFetchMock = vi.mocked(apiFetch);

afterEach(() => {
	vi.unstubAllGlobals();
});

// Ein Doppelklick auf „DSGVO-Auskunft" erteilte die Auskunft zweimal: zwei Tabs und zwei
// Einträge im Protokoll, in Chromium, Firefox und WebKit (OFFEN.md 5.106).
describe('DSGVO-Auskunft: eine Auskunft je Klick', () => {
	it('fragt bei einem zweiten Klick vor der Antwort nicht noch einmal an', async () => {
		/** @type {(antwort: any) => void} */
		let liefereAuskunft = () => {};
		apiFetchMock.mockImplementation(() => new Promise((fertig) => (liefereAuskunft = fertig)));
		const fenster = { close: vi.fn(), closed: false, location: { replace: vi.fn() } };
		const oeffne = vi.fn(() => fenster);
		vi.stubGlobal('open', oeffne);
		const k = render(SchuelerDokumente, {
			leser: { id: 'l1', vorname: 'Mia', nachname: 'Probe', entliehene_buecher: [] },
			darfAuskunft: true,
			kontoauszugLaedt: false,
			rechnungLaedt: false,
			zeigeKontoauszug: vi.fn(),
			zeigeRechnung: vi.fn()
		});
		const knopf = /** @type {HTMLButtonElement} */ (
			k.getByRole('button', { name: /DSGVO-Auskunft/ })
		);

		await fireEvent.click(knopf);
		await fireEvent.click(knopf);

		expect(apiFetchMock).toHaveBeenCalledTimes(1);
		expect(oeffne).toHaveBeenCalledTimes(1);
		expect(knopf.disabled).toBe(true);

		liefereAuskunft(serverAntwort(500, { error: 'Auskunft nicht erzeugt' }));
		await waitFor(() => expect(knopf.disabled).toBe(false));
	});
});
