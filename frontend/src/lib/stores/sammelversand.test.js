import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../../inventur/lib/store.svelte.js', () => ({ melde: vi.fn() }));
import { melde } from '../../inventur/lib/store.svelte.js';
import { erzeugeSammelversand } from './sammelversand.svelte.js';

// Der Sammelversand an die Klassenleitungen (Mahnlauf, Kontoauszüge der Abgänger): Ein Lauf
// dauert je Klasse eine Mail lang, und ein zweiter Start in der Zeit verschickte jede Mail noch
// einmal. Die Sperre greift vor der Anfrage; der Knopf der Seite liest `laeuft`. Rumpf und
// Meldungen prüfen mahnwesenMail.test.js und abgaengerDienst.test.js an beiden Aufrufern.

/** @param {any} json */
const antwort = (json, ok = true) => /** @type {any} */ ({ ok, json: async () => json });

/** Ein Aufruf, dessen Antwort erst kommt, wenn der Test sie freigibt. */
function halteAntwort() {
	/** @type {(wert: any) => void} */
	let gib = () => {};
	const schicke = vi.fn(() => new Promise((fertig) => (gib = fertig)));
	return { schicke, gib: (/** @type {any} */ json) => gib(antwort(json)) };
}

describe('erzeugeSammelversand', () => {
	beforeEach(() => {
		vi.mocked(melde).mockReset();
	});

	it('schickt einen zweiten Start nicht, solange der erste läuft, und zeigt den Lauf', async () => {
		const { schicke, gib } = halteAntwort();
		const versand = erzeugeSammelversand(schicke, () => 'versendet');
		expect(versand.laeuft).toBe(false);

		const erster = versand.sende({ klassen: ['5a'] });
		expect(versand.laeuft, 'läuft, bevor die Antwort da ist').toBe(true);
		// Nicht abwarten: Ein zweiter Aufruf, der durchkäme, wartete auf eine Antwort, die nie
		// kommt, und der Test liefe in seine Frist statt an dieser Zeile rot zu werden.
		const zweiter = versand.sende({ klassen: ['5a'] });
		expect(schicke).toHaveBeenCalledTimes(1);

		gib({ sent_count: 1 });
		await Promise.all([erster, zweiter]);
		expect(versand.laeuft).toBe(false);
		expect(melde).toHaveBeenCalledTimes(1);
		expect(melde).toHaveBeenCalledWith('versendet', 'success');
	});

	it('gibt den Lauf nach einem Netzfehler frei', async () => {
		const schicke = vi
			.fn()
			.mockRejectedValueOnce(new Error('weg'))
			.mockResolvedValueOnce(antwort({ sent_count: 1 }));
		const versand = erzeugeSammelversand(schicke, () => 'versendet');

		await versand.sende({ klassen: ['5a'] });
		expect(versand.laeuft).toBe(false);
		expect(vi.mocked(melde).mock.calls[0][1]).toBe('error');

		await versand.sende({ klassen: ['5a'] });
		expect(schicke).toHaveBeenCalledTimes(2);
	});

	it('meldet einen teilweise gescheiterten Lauf als Warnung, nicht als Erfolg', async () => {
		const schicke = vi
			.fn()
			.mockResolvedValueOnce(
				antwort({ sent_count: 2, failed_count: 1, message: 'Für 1 Klasse nicht zugestellt' })
			);
		await erzeugeSammelversand(schicke, () => 'versendet').sende({ klassen: ['5a', '6b'] });
		expect(melde).toHaveBeenCalledWith('Für 1 Klasse nicht zugestellt', 'warning');
	});

	it('beginnt ohne Auswahl keinen Lauf', async () => {
		const schicke = vi.fn();
		const versand = erzeugeSammelversand(schicke, () => 'versendet');
		await versand.sende({ klassen: [] });
		expect(schicke).not.toHaveBeenCalled();
		expect(versand.laeuft).toBe(false);
		expect(vi.mocked(melde).mock.calls[0][1]).toBe('error');
	});
});
