import { describe, it, expect, vi, beforeEach } from 'vitest';
import { erzeugeSchuelerSuche } from './schuelerSuche.svelte.js';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	extractApiError: vi.fn(async (/** @type {any} */ res) => JSON.parse(await res.text()).error)
}));

// Die Suche der Schülerdatei läuft auf dem Server, und nur die JÜNGSTE Antwort darf die
// Liste schreiben (`nr === ladeNr`). Bis zum 12.09.2026 hing daran auch das `res.ok`:
// Scheiterte der Lauf, blieb die Trefferliste der VORIGEN Suche stehen — unter dem neuen
// Suchtext. Das ist der Fund, mit dem der Sweep „verschluckte Fehlantwort" angefangen
// hat (die Theke zeigte unter dem neuen Text die Schüler des alten, Klick buchte auf den
// falschen Schüler); der Detektor sah diese Stelle nur nicht, weil die Bedingung ein
// zweites Glied trägt (Register, Bestands-Durchgang 10.09.2026).
const treffer = [{ id: 's1', vorname: 'Mia', nachname: 'Muster', klasse: '07A' }];

/** @param {any} body */
const ok = (body) => /** @type {any} */ ({ ok: true, json: async () => body });
/** @param {string} meldung */
const fehler = (meldung) =>
	/** @type {any} */ ({
		ok: false,
		status: 500,
		text: async () => JSON.stringify({ error: meldung })
	});

describe('Schülerdatei-Suche', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	it('zeigt nach einem gescheiterten Lauf keine Treffer von vorher', async () => {
		vi.mocked(apiFetch).mockResolvedValue(ok(treffer));
		// Der Store hängt einen $effect auf (Klassen-Stapeldruck aus dem Druck-Center) —
		// erzeugt wird er deshalb in einer Effekt-Wurzel, wie in useBookAkte.effekt.svelte.test.js.
		/** @type {any} */
		let suche;
		const stopp = $effect.root(() => {
			suche = erzeugeSchuelerSuche(() => {});
		});
		try {
			await suche.lade();
			expect(suche.students).toHaveLength(1);

			vi.mocked(apiFetch).mockResolvedValue(fehler('Datenbank nicht erreichbar'));
			suche.query = 'Xaver';
			await suche.lade();

			expect(suche.students, 'die Treffer der vorigen Suche stehen noch da').toHaveLength(0);
			expect(suche.ladefehler).toBe('Datenbank nicht erreichbar');

			// Und der nächste gelungene Lauf räumt die Meldung wieder weg.
			vi.mocked(apiFetch).mockResolvedValue(ok(treffer));
			await suche.lade();
			expect(suche.students).toHaveLength(1);
			expect(suche.ladefehler).toBe('');
		} finally {
			stopp();
		}
	});
});
