import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
	MonitorTakt,
	FOLIE_MS,
	COVER_MS,
	NACHLADEN_MS,
	NEUVERSUCH_MS
} from './monitorTakt.svelte.js';

// Der Takt des Flur-Monitors, mit gestellter Uhr. Der Bildschirm hat keine Tastatur und
// läuft wochenlang — was hier schiefgeht, sieht wochenlang niemand.

/** @param {string} marke */
function stand(marke) {
	const titel = (/** @type {string} */ zusatz) => ({
		id: zusatz,
		titel: `${marke} ${zusatz}`,
		autor: '',
		cover_url: '',
		isbn: ''
	});
	return {
		buch_des_monats: titel('Monatsbuch'),
		neu_eingetroffen: [titel('Neu 1'), titel('Neu 2'), titel('Neu 3')],
		beliebt: [titel('Renner')]
	};
}

describe('MonitorTakt', () => {
	/** @type {MonitorTakt} */
	let takt;

	beforeEach(() => {
		vi.useFakeTimers();
	});
	afterEach(() => {
		takt?.stop();
		vi.useRealTimers();
	});

	it('versucht es alle 30 s erneut, solange noch nichts da ist — und hört damit auf, sobald etwas da ist', async () => {
		const lader = vi
			.fn()
			.mockResolvedValueOnce(null)
			.mockResolvedValueOnce(null)
			.mockResolvedValue(stand('A'));
		takt = new MonitorTakt(lader);
		await takt.start();
		expect(takt.slides).toBeNull();
		expect(lader).toHaveBeenCalledTimes(1);

		await vi.advanceTimersByTimeAsync(NEUVERSUCH_MS);
		expect(lader).toHaveBeenCalledTimes(2);
		expect(takt.slides).toBeNull();

		await vi.advanceTimersByTimeAsync(NEUVERSUCH_MS);
		expect(lader).toHaveBeenCalledTimes(3);
		expect(takt.slides?.buch_des_monats?.titel).toBe('A Monatsbuch');

		// Danach nur noch der Fünf-Minuten-Takt — kein Neuversuch mehr.
		await vi.advanceTimersByTimeAsync(NEUVERSUCH_MS * 3);
		expect(lader).toHaveBeenCalledTimes(3);
	});

	it('lädt alle fünf Minuten nach und blendet den neuen Stand erst am Folienwechsel ein', async () => {
		const lader = vi.fn().mockResolvedValueOnce(stand('A')).mockResolvedValue(stand('B'));
		takt = new MonitorTakt(lader);
		await takt.start();
		expect(takt.slides?.buch_des_monats?.titel).toBe('A Monatsbuch');

		await vi.advanceTimersByTimeAsync(NACHLADEN_MS);
		expect(lader).toHaveBeenCalledTimes(2);
		// Der neue Stand liegt bereit, ist aber noch nicht zu sehen.
		expect(takt.slides?.buch_des_monats?.titel).toBe('A Monatsbuch');

		await vi.advanceTimersByTimeAsync(FOLIE_MS);
		expect(takt.slides?.buch_des_monats?.titel).toBe('B Monatsbuch');
	});

	it('lässt bei einem gescheiterten Abruf den alten Stand stehen — ohne Neuversuch-Dauerfeuer', async () => {
		const lader = vi
			.fn()
			.mockResolvedValueOnce(stand('A'))
			.mockRejectedValue(new Error('Netz weg'));
		takt = new MonitorTakt(lader);
		await takt.start();

		await vi.advanceTimersByTimeAsync(NACHLADEN_MS + FOLIE_MS);
		expect(lader).toHaveBeenCalledTimes(2);
		expect(takt.slides?.buch_des_monats?.titel).toBe('A Monatsbuch');

		// Ein Stand ist da: Der 30-Sekunden-Neuversuch bleibt aus, es gilt der Fünf-Minuten-Takt.
		await vi.advanceTimersByTimeAsync(NEUVERSUCH_MS * 2);
		expect(lader).toHaveBeenCalledTimes(2);
	});

	it('läuft im Kreis durch die drei Folien; der Cover-Lauf bewegt sich nur auf „Neu eingetroffen"', async () => {
		takt = new MonitorTakt(vi.fn().mockResolvedValue(stand('A')));
		await takt.start();
		expect(takt.folie).toBe(0);

		await vi.advanceTimersByTimeAsync(COVER_MS);
		expect(takt.coverIndex, 'auf Folie 0 kein Cover-Lauf').toBe(0);

		await vi.advanceTimersByTimeAsync(FOLIE_MS - COVER_MS);
		expect(takt.folie).toBe(1);
		// 15 000 ist ein Vielfaches von 2 500: Im Tick des Folienwechsels feuert auch der
		// Cover-Takt — deshalb relativ messen, nicht absolut.
		const vorher = takt.coverIndex;
		await vi.advanceTimersByTimeAsync(COVER_MS);
		expect(takt.coverIndex).toBe(vorher + 1);

		await vi.advanceTimersByTimeAsync(FOLIE_MS - COVER_MS);
		expect(takt.folie).toBe(2);
		expect(takt.coverIndex, 'Folienwechsel setzt den Cover-Lauf zurück').toBe(0);

		await vi.advanceTimersByTimeAsync(FOLIE_MS);
		expect(takt.folie).toBe(0);
	});

	it('springeZu() wechselt die Folie von Hand und startet den Fortschrittsbalken neu', async () => {
		takt = new MonitorTakt(vi.fn().mockResolvedValue(stand('A')));
		await takt.start();
		const lauf = takt.lauf;
		takt.springeZu(2);
		expect(takt.folie).toBe(2);
		expect(takt.lauf).toBe(lauf + 1);
	});

	it('stop() hinterlässt keine Uhr — auch nicht mitten im Ladezustand', async () => {
		const lader = vi.fn().mockResolvedValue(null);
		takt = new MonitorTakt(lader);
		await takt.start();
		takt.stop();

		await vi.advanceTimersByTimeAsync(NEUVERSUCH_MS * 2 + NACHLADEN_MS);
		expect(lader).toHaveBeenCalledTimes(1);
	});
});
