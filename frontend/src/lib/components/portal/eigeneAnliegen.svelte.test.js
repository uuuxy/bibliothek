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
		expect(anliegen.fehler, 'wirklich leer ist kein Fehler').toBe(false);
	});

	// Die andere Hälfte derselben Frage (17.09.2026): Beim ERSTEN Laden gibt es keinen
	// alten Stand, auf den man zurückfallen kann. Ein 503 liess die Liste leer — und leer
	// liest sich wie „du hast keine Anliegen". Wer gestern einen Wunsch geschickt hat,
	// hält ihn für verloren und schickt ihn noch einmal.
	it('meldet den gescheiterten ERSTEN Abruf, statt „nichts da" zu behaupten', async () => {
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: false, status: 503 }));
		const anliegen = erzeugeEigeneAnliegen();

		await anliegen.lade();

		expect(anliegen.fehler, 'ohne je geladene Daten ist die leere Liste keine Auskunft').toBe(true);
		expect(anliegen.liste).toHaveLength(0);
	});

	it('meldet auch einen Netzfehler beim ersten Abruf', async () => {
		vi.mocked(apiFetch).mockRejectedValueOnce(new Error('Netz weg'));
		const anliegen = erzeugeEigeneAnliegen();
		await anliegen.lade();
		expect(anliegen.fehler).toBe(true);
	});

	it('nimmt die Fehlanzeige zurück, sobald ein Abruf durchkommt', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(/** @type {any} */ ({ ok: false, status: 503 }));
		const anliegen = erzeugeEigeneAnliegen();
		await anliegen.lade();
		expect(anliegen.fehler).toBe(true);

		vi.mocked(apiFetch).mockResolvedValueOnce(
			/** @type {any} */ ({ ok: true, json: async () => ANLIEGEN })
		);
		await anliegen.lade();
		expect(anliegen.fehler).toBe(false);
		expect(anliegen.liste).toHaveLength(1);
	});
});
