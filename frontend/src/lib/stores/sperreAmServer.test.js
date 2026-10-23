import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn() }));

import { apiFetch } from '../apiFetch.js';
import { leseSperrzustand } from './sperreAmServer.js';

const antwort = (/** @type {number} */ status, /** @type {any} */ rumpf) =>
	/** @type {any} */ ({ ok: status < 300, status, json: async () => rumpf });

// Der Sperrbildschirm nennt den Grund, den der Server für die Sperre angibt (auth/anmelden.go,
// AngemeldetesKonto): Wer zehn Stunden nach der Passworteingabe gerade gearbeitet hat, liest
// nicht „wegen Inaktivität".
describe('leseSperrzustand', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	it('nennt die Höchstdauer, wenn der Server sie als Grund angibt', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			antwort(423, { error: 'Seit …', email: 'a@schule.example', grund: 'hoechstdauer' })
		);
		expect(await leseSperrzustand()).toEqual({ zustand: 'gesperrt', grund: 'hoechstdauer' });
	});

	it('nennt die Inaktivität, wenn der Server sie angibt', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(423, { grund: 'inaktivitaet' }));
		expect(await leseSperrzustand()).toEqual({ zustand: 'gesperrt', grund: 'inaktivitaet' });
	});

	it('nimmt ohne lesbaren Grund die Inaktivität an und bleibt gesperrt', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: false,
				status: 423,
				json: async () => {
					throw new SyntaxError('kein JSON');
				}
			})
		);
		expect(await leseSperrzustand()).toEqual({ zustand: 'gesperrt', grund: 'inaktivitaet' });
	});
});
