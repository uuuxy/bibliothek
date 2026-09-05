import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
	MonitorTakt,
	FOLIE_MS,
	COVER_MS,
	NACHLADEN_MS,
	NEUVERSUCH_MS,
	NEUSTART_STUNDE,
	msBisNeustart
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
		// Feste Uhr für JEDEN Fall. Ohne sie erbt der Neustart-Wecker (setTimeout auf die
		// nächste volle NEUSTART_STUNDE) die Wanduhr der Maschine — der Test hinge davon ab,
		// wann am Tag er läuft, und zwischen 2 und 3 Uhr nachts feuerte er in fremde Fälle
		// hinein. 10:00 liegt 17 Stunden vor dem nächsten Neustart, weiter als jeder Fall
		// hier die Uhr vorstellt.
		vi.setSystemTime(new Date(2026, 8, 1, 10, 0, 0));
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

	// In sechs Wochen Sommerferien hat „Beliebt diese Woche" keine einzige Ausleihe. Ein
	// Flurbildschirm, der ein Drittel der Zeit „Keine Daten verfügbar" zeigt, wird
	// abgeschaltet — und dann fehlt er auch im September.
	it('überspringt leere Folien im Kreis', async () => {
		const ohneRenner = { ...stand('A'), beliebt: [] };
		takt = new MonitorTakt(vi.fn().mockResolvedValue(ohneRenner));
		await takt.start();
		expect(takt.folie).toBe(0);

		await vi.advanceTimersByTimeAsync(FOLIE_MS);
		expect(takt.folie).toBe(1);
		await vi.advanceTimersByTimeAsync(FOLIE_MS);
		expect(takt.folie, 'Folie 2 ist leer und wird übersprungen').toBe(0);
	});

	it('startet auf der ersten Folie mit Inhalt, wenn „Buch des Monats" leer ist', async () => {
		const nurRenner = { ...stand('A'), buch_des_monats: null, neu_eingetroffen: [] };
		takt = new MonitorTakt(vi.fn().mockResolvedValue(nurRenner));
		await takt.start();
		expect(takt.folie).toBe(2);

		await vi.advanceTimersByTimeAsync(FOLIE_MS);
		expect(takt.folie, 'die einzige gefüllte Folie bleibt stehen').toBe(2);
	});

	it('springt beim Übernehmen eines neuen Stands nicht auf eine Folie, die darin leer ist', async () => {
		const lader = vi
			.fn()
			.mockResolvedValueOnce(stand('A'))
			.mockResolvedValue({ ...stand('B'), buch_des_monats: null });
		takt = new MonitorTakt(lader);
		await takt.start();

		// Nachladen bei 5:00 — nach 20 Wechseln steht der Takt auf Folie 2 (20 mod 3). Der
		// Wechsel bei 5:15 übernimmt B, und darin ist die nächste Folie (0) leer.
		await vi.advanceTimersByTimeAsync(NACHLADEN_MS);
		expect(takt.folie).toBe(2);
		await vi.advanceTimersByTimeAsync(FOLIE_MS);
		expect(takt.slides?.beliebt?.[0]?.titel).toBe('B Renner');
		expect(takt.folie, 'Folie 0 ist im neuen Stand leer').toBe(1);
	});

	it('bleibt stehen, wenn alle Folien leer sind — der Balken läuft trotzdem weiter', async () => {
		const leer = { buch_des_monats: null, neu_eingetroffen: [], beliebt: [] };
		takt = new MonitorTakt(vi.fn().mockResolvedValue(leer));
		await takt.start();
		const lauf = takt.lauf;
		await vi.advanceTimersByTimeAsync(FOLIE_MS);
		expect(takt.folie).toBe(0);
		expect(takt.lauf).toBe(lauf + 1);
	});

	// Der Bildschirm hat keine Tastatur und navigiert nie. Der Service Worker prüft nur beim
	// Laden auf eine neue Version (main.js: registerSW) — ohne eigenen Neustart bekäme der
	// Monitor ein Deploy erst mit, wenn jemand den Stecker zieht. Nachts um drei stört es
	// niemanden; und /events (SSE) steht dem Monitor ohne Anmeldung nicht zur Verfügung.
	describe('nächtlicher Neustart', () => {
		it('msBisNeustart() rechnet bis zur nächsten vollen NEUSTART_STUNDE in Ortszeit', () => {
			expect(NEUSTART_STUNDE).toBe(3);
			const stunde = 60 * 60 * 1000;
			expect(msBisNeustart(new Date(2026, 8, 1, 10, 0, 0))).toBe(17 * stunde);
			expect(msBisNeustart(new Date(2026, 8, 1, 2, 30, 0))).toBe(stunde / 2);
			// Genau um drei ist der nächste Neustart morgen, nicht jetzt.
			expect(msBisNeustart(new Date(2026, 8, 1, 3, 0, 0))).toBe(24 * stunde);
		});

		// Kurz VOR drei stellen, nicht siebzehn Stunden davor: Die Uhr ist zwar gestellt,
		// die Takte laufen beim Vorstellen aber wirklich — 17 Stunden sind 4 080 Folien-,
		// 24 480 Cover- und 204 Nachlade-Durchläufe, jeder mit einem await. Im vollen
		// Vitest-Lauf riss das den Fall gelegentlich in die 5-Sekunden-Grenze (04.09.2026:
		// einmal rot, dreimal grün, allein immer grün). Dass die Rechnung auch über große
		// Abstände stimmt, prüft der Fall darüber an msBisNeustart() direkt — ohne Takte.
		it('startet die Seite um drei Uhr nachts neu — und nicht vorher', async () => {
			vi.setSystemTime(new Date(2026, 8, 1, 2, 59, 0));
			const neustart = vi.fn();
			takt = new MonitorTakt(vi.fn().mockResolvedValue(stand('A')), { neustart });
			await takt.start();

			await vi.advanceTimersByTimeAsync(60 * 1000 - 1000);
			expect(neustart).not.toHaveBeenCalled();
			await vi.advanceTimersByTimeAsync(1000);
			expect(neustart).toHaveBeenCalledTimes(1);
		});

		it('stop() nimmt auch den Neustart-Wecker mit', async () => {
			vi.setSystemTime(new Date(2026, 8, 1, 2, 59, 0));
			const neustart = vi.fn();
			takt = new MonitorTakt(vi.fn().mockResolvedValue(stand('A')), { neustart });
			await takt.start();
			takt.stop();
			await vi.advanceTimersByTimeAsync(2 * 60 * 1000);
			expect(neustart).not.toHaveBeenCalled();
		});
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
