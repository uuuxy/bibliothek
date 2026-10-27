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
import { etikettenStore } from '../../stores/etiketten.svelte.js';
import { serverAntwort } from '../../serverAntwort.js';
import Etikettendruck from './Etikettendruck.svelte';

const apiFetchMock = vi.mocked(apiFetch);

afterEach(() => {
	vi.unstubAllGlobals();
});

// Ein Doppelklick auf „A4-Bogen drucken" schickte den Bogen zweimal und öffnete ihn in zwei Tabs,
// in Chromium, Firefox und WebKit (OFFEN.md 5.106). Gedruckt worden wären dieselben Etiketten
// zweimal.
describe('Etikettendruck: ein Bogen je Klick', () => {
	it('schickt den Bogen bei einem zweiten Klick vor der Antwort nicht noch einmal', async () => {
		/** @type {(antwort: any) => void} */
		let liefereBogen = () => {};
		apiFetchMock.mockImplementation(async (url) => {
			if (url === '/api/print/labels') return new Promise((fertig) => (liefereBogen = fertig));
			if (String(url).endsWith('/exemplare'))
				return serverAntwort(200, [{ barcode_id: 'B-1' }, { barcode_id: 'B-2' }]);
			return serverAntwort(200, []);
		});
		const fenster = { close: vi.fn(), closed: false, location: { replace: vi.fn() } };
		const oeffne = vi.fn(() => fenster);
		vi.stubGlobal('open', oeffne);
		await etikettenStore.waehleTitel({ id: 't1', titel: 'Titel', autor: '' });
		const k = render(Etikettendruck);
		const knopf = /** @type {HTMLButtonElement} */ (
			k.getByRole('button', { name: /A4-Bogen drucken/ })
		);

		await fireEvent.click(knopf);
		await fireEvent.click(knopf);

		const bogenAnfragen = () =>
			apiFetchMock.mock.calls.filter(([url]) => url === '/api/print/labels').length;
		expect(bogenAnfragen()).toBe(1);
		expect(oeffne).toHaveBeenCalledTimes(1);
		expect(knopf.disabled).toBe(true);

		liefereBogen(serverAntwort(500, { error: 'Bogen nicht erzeugt' }));
		await waitFor(() => expect(knopf.disabled).toBe(false));
	});
});
