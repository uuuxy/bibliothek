import { describe, it, expect, vi } from 'vitest';
import 'fake-indexeddb/auto';

// Die Reihe der Theke liegt nur im Speicher (omniboxReihe.svelte.js). Antwortet der Server
// nicht, wartet ein Scan bis zu 10 s, bevor er ohne Netz abgelegt wird; ein Neuladen in dieser
// Zeit verwürfe ihn und die Scans dahinter ohne Meldung (docs/OFFEN.md 5.83). Der Browser fragt
// deshalb vorher. Eigene Datei, damit kein Store eines anderen Tests mitfragt.

vi.mock('./snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));
vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn(), apiClient: { post: vi.fn() } }));
vi.mock('../audio.js', () => ({
	spieleErfolgston: vi.fn(),
	spieleFehlerton: vi.fn()
}));
vi.mock('./buchBarcodes.svelte.js', () => ({
	buchBarcodes: {
		istBuch: () => false,
		bereitstellen: vi.fn(),
		laden: vi.fn(),
		auffrischen: vi.fn(),
		anzahl: 0,
		geholtAm: 0
	}
}));

import { apiClient } from '../apiFetch.js';
import { erzeugeOmniboxStore } from './omnibox.svelte.js';

/** Fragt der Browser vor dem Verlassen? */
function fragtVorDemVerlassen() {
	const ereignis = new Event('beforeunload', { cancelable: true });
	window.dispatchEvent(ereignis);
	return ereignis.defaultPrevented;
}

describe('Theke: Neuladen, solange Scans auf die Antwort warten', () => {
	it('fragt, solange ein Scan gebucht wird oder wartet, und danach nicht mehr', async () => {
		const theke = erzeugeOmniboxStore();
		/** @type {(antwort: any) => void} */
		let beantworte = () => {};
		vi.mocked(apiClient.post).mockImplementationOnce(() => new Promise((ja) => (beantworte = ja)));
		const buchung = /** @type {any} */ ({
			ok: true,
			status: 200,
			json: async () => ({ type: 'ausleihe', book: { titel: 'Natura 2' } })
		});
		vi.mocked(apiClient.post).mockResolvedValue(buchung);
		expect(fragtVorDemVerlassen(), 'ohne Scan').toBe(false);

		theke.aktiverLeser = { id: 'leser-1', art: 'schueler', vorname: 'Mia', nachname: 'Muster' };
		theke.eingabe = 'B-1';
		const laeuft = theke.fuehreAktionAus(null, null);
		theke.eingabe = 'B-2';
		theke.fuehreAktionAus(null, null);
		expect(fragtVorDemVerlassen(), 'ein Scan wird gebucht, einer wartet').toBe(true);

		beantworte(buchung);
		await laeuft;
		expect(vi.mocked(apiClient.post), 'beide Scans gehen an die Buchungstür').toHaveBeenCalledTimes(
			2
		);
		expect(fragtVorDemVerlassen(), 'beide gebucht').toBe(false);
		theke.stoppeZeitgeber();
	});
});
