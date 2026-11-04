import { describe, it, expect, vi, beforeEach } from 'vitest';

// leseApiFehler bleibt echt (ARCHITEKTUR.md 8.14).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('../../stores/bestaetigung.svelte.js', () => ({ loeschenMitGrund: vi.fn() }));

import { apiFetch } from '../../apiFetch.js';
import { loeschenMitGrund } from '../../stores/bestaetigung.svelte.js';
import { frageGrundFuerExemplare, loescheExemplarMitGrund } from './exemplareLoeschen.js';

/** @param {number} status @param {any} koerper */
const antwort = (status, koerper) =>
	/** @type {any} */ ({
		ok: status < 400,
		status,
		json: async () => koerper,
		text: async () => JSON.stringify(koerper)
	});

describe('frageGrundFuerExemplare', () => {
	beforeEach(() => {
		vi.mocked(loeschenMitGrund).mockReset();
	});

	it('nennt das eine Exemplar und liefert den Grund', async () => {
		vi.mocked(loeschenMitGrund).mockResolvedValue({ grund: 'BESCHAEDIGUNG', zusatz: false });
		expect(await frageGrundFuerExemplare(['B-7'])).toBe('BESCHAEDIGUNG');
		expect(loeschenMitGrund).toHaveBeenCalledWith({
			titel: 'Exemplar B-7 löschen?',
			text: 'Es kommt mit diesem Grund ins Abgangsbuch.'
		});
	});

	it('zählt mehrere und liefert bei Abbrechen null', async () => {
		vi.mocked(loeschenMitGrund).mockResolvedValue(null);
		expect(await frageGrundFuerExemplare(['B-1', 'B-2'])).toBeNull();
		expect(loeschenMitGrund).toHaveBeenCalledWith({
			titel: '2 Exemplare löschen?',
			text: 'Sie kommen mit diesem Grund ins Abgangsbuch.'
		});
	});
});

describe('loescheExemplarMitGrund', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	it('schickt den Grund mit', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(200, { message: 'ok' }));
		await loescheExemplarMitGrund('e-1', 'VERLUST');
		expect(apiFetch).toHaveBeenCalledWith(
			'/api/buecher/exemplare/e-1',
			expect.objectContaining({ method: 'DELETE', body: JSON.stringify({ grund: 'VERLUST' }) })
		);
	});

	it('nennt die Meldung des Servers bei einem verliehenen Exemplar', async () => {
		const meldung =
			'Das Exemplar ist noch verliehen. Erst die Rückgabe buchen oder in der Leserakte den Verlust melden.';
		vi.mocked(apiFetch).mockResolvedValue(antwort(400, { error: meldung }));
		await expect(loescheExemplarMitGrund('e-1', 'AUSSORTIERT')).rejects.toThrow(meldung);
	});
});
