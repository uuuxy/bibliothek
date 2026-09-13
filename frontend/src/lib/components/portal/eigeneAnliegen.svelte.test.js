import { describe, it, expect, vi, beforeEach } from 'vitest';
import { erzeugeEigeneAnliegen } from './eigeneAnliegen.svelte.js';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({ apiFetch: vi.fn() }));

const ANLIEGEN = [
	{ id: 'a1', art: 'wunsch', titel_text: 'Markl Biologie 2', klasse: '08A', erstellt_am: 'x' }
];

describe('Eigene Anliegen (Kollegiums-Portal)', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	// Nachladen passiert direkt nach dem Absenden eines Wunsches (onaktualisiert). Scheitert
	// es, darf der gerade abgeschickte Wunsch nicht wieder vom Bildschirm verschwinden:
	// Die Lehrkraft läse daraus, dass er nicht angekommen ist — und schickte ihn noch einmal.
	it('hält den alten Stand, wenn das Nachladen scheitert', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ANLIEGEN })
		);
		const anliegen = erzeugeEigeneAnliegen();
		await anliegen.lade();
		expect(anliegen.liste).toHaveLength(1);
		expect(anliegen.offene).toBe(1);

		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: false, status: 503 }));
		await anliegen.lade();
		expect(anliegen.liste, 'der abgeschickte Wunsch ist vom Bildschirm verschwunden').toHaveLength(
			1
		);

		vi.mocked(apiFetch).mockRejectedValueOnce(new Error('Netz weg'));
		await anliegen.lade();
		expect(anliegen.liste).toHaveLength(1);
	});

	it('leer bleibt leer, wenn der Server wirklich nichts liefert', async () => {
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: true, json: async () => [] }));
		const anliegen = erzeugeEigeneAnliegen();
		await anliegen.lade();
		expect(anliegen.liste).toHaveLength(0);
	});
});
