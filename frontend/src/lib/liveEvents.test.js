import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { verbinde, trenne, abonniere, _zuruecksetzenFuerTests } from './liveEvents.js';

// Die Live-Verbindung ist der einzige Weg, auf dem ein Arbeitsplatz erfährt, dass an einem
// ANDEREN etwas gebucht wurde. Fällt sie aus, ohne sich zu melden, arbeitet der Bildschirm
// mit altem Stand weiter — und sieht dabei aus wie immer.
//
// Genau dieser Zustand war real: Zwei der drei Verbindungen hatten keine Fehlerbehandlung
// und verliessen sich auf den Wiederaufbau des Browsers. Der greift bei Netzfehlern, aber
// NICHT nach einer Fehlerantwort des Servers: Dann schliesst ein EventSource endgültig.
// Nach einer abgelaufenen Sitzung (12 h, /events antwortet 401) blieben sie tot bis F5.

/** Ein EventSource-Ersatz, der sich befragen und von Hand auslösen lässt. */
class FakeEventSource {
	static instanzen = [];

	constructor(url) {
		this.url = url;
		this.geschlossen = false;
		this.onerror = null;
		this.listener = new Map();
		FakeEventSource.instanzen.push(this);
	}

	addEventListener(name, handler) {
		if (!this.listener.has(name)) this.listener.set(name, []);
		this.listener.get(name).push(handler);
	}

	close() {
		this.geschlossen = true;
	}

	/** Server schickt ein Ereignis. */
	sende(name, data = '{}') {
		for (const handler of this.listener.get(name) ?? []) handler({ data });
	}

	/** Verbindung scheitert (Netzfehler ODER Fehlerantwort — der Browser meldet beides so). */
	scheitert() {
		this.onerror?.(new Event('error'));
	}

	static get letzte() {
		return FakeEventSource.instanzen[FakeEventSource.instanzen.length - 1];
	}
}

describe('liveEvents', () => {
	beforeEach(() => {
		FakeEventSource.instanzen = [];
		vi.stubGlobal('EventSource', FakeEventSource);
		vi.useFakeTimers();
	});

	afterEach(() => {
		_zuruecksetzenFuerTests();
		vi.useRealTimers();
		vi.unstubAllGlobals();
	});

	it('öffnet für viele Zuhörer nur EINE Verbindung', () => {
		abonniere('action', () => {});
		abonniere('action', () => {});
		abonniere('ping', () => {});
		verbinde();
		verbinde(); // Login + Session-Restore rufen beide auf.

		expect(FakeEventSource.instanzen).toHaveLength(1);
	});

	it('liefert jedes Ereignis an alle Zuhörer desselben Namens', () => {
		const a = vi.fn();
		const b = vi.fn();
		abonniere('action', a);
		abonniere('action', b);
		verbinde();

		FakeEventSource.letzte.sende('action', '{"event":"rueckgabe"}');

		expect(a).toHaveBeenCalledTimes(1);
		expect(b).toHaveBeenCalledTimes(1);
		expect(JSON.parse(a.mock.calls[0][0].data).event).toBe('rueckgabe');
	});

	it('ein gescheiterter Zuhörer reisst die anderen nicht mit', () => {
		const kaputt = vi.fn(() => {
			throw new Error('Absicht');
		});
		const heil = vi.fn();
		abonniere('action', kaputt);
		abonniere('action', heil);
		verbinde();

		vi.spyOn(console, 'error').mockImplementation(() => {});
		FakeEventSource.letzte.sende('action');

		expect(heil).toHaveBeenCalledTimes(1);
	});

	it('abgemeldete Zuhörer bekommen nichts mehr', () => {
		const handler = vi.fn();
		const abmelden = abonniere('action', handler);
		verbinde();

		abmelden();
		FakeEventSource.letzte.sende('action');

		expect(handler).not.toHaveBeenCalled();
	});

	// Der Kern: Auch nach einer Fehlerantwort muss die Verbindung zurückkommen. Ein
	// EventSource täte das von sich aus NICHT.
	it('baut nach einem Fehler neu auf', () => {
		abonniere('action', () => {});
		verbinde();
		const erste = FakeEventSource.letzte;

		erste.scheitert();
		expect(erste.geschlossen).toBe(true); // wir steuern den Wiederaufbau, nicht der Browser
		expect(FakeEventSource.instanzen).toHaveLength(1);

		vi.advanceTimersByTime(2000);
		expect(FakeEventSource.instanzen).toHaveLength(2);
	});

	it('zieht die Wartezeit hoch und setzt sie nach einem Server-Signal zurück', () => {
		verbinde();

		FakeEventSource.letzte.scheitert();
		vi.advanceTimersByTime(2000);
		expect(FakeEventSource.instanzen).toHaveLength(2);

		// Zweiter Fehlschlag: erst nach 4 s, nicht schon nach 2 s.
		FakeEventSource.letzte.scheitert();
		vi.advanceTimersByTime(2000);
		expect(FakeEventSource.instanzen).toHaveLength(2);
		vi.advanceTimersByTime(2000);
		expect(FakeEventSource.instanzen).toHaveLength(3);

		// Ein Ping beweist eine funktionierende Leitung — der nächste Aussetzer darf
		// nicht eine halbe Minute lang unbemerkt bleiben.
		FakeEventSource.letzte.sende('ping');
		FakeEventSource.letzte.scheitert();
		vi.advanceTimersByTime(2000);
		expect(FakeEventSource.instanzen).toHaveLength(4);
	});

	it('deckelt die Wartezeit bei 30 Sekunden', () => {
		verbinde();
		// Sieben Fehlschläge: 2, 4, 8, 16, 30, 30, 30 …
		let erwartet = 2000;
		for (let i = 0; i < 7; i++) {
			const vorher = FakeEventSource.instanzen.length;
			FakeEventSource.letzte.scheitert();
			vi.advanceTimersByTime(Math.min(erwartet, 30000));
			expect(FakeEventSource.instanzen.length).toBe(vorher + 1);
			erwartet = Math.min(erwartet * 2, 30000);
		}
	});

	// Nach der Abmeldung antwortet /events nur noch mit 401. Ein weiterlaufender
	// Wiederaufbau klopfte dann bis zum Feierabend dagegen — mal zehn Arbeitsplätze.
	it('trenne() beendet die Verbindung und jeden geplanten Wiederaufbau', () => {
		verbinde();
		const erste = FakeEventSource.letzte;
		erste.scheitert();

		trenne();
		vi.advanceTimersByTime(60000);

		expect(erste.geschlossen).toBe(true);
		expect(FakeEventSource.instanzen).toHaveLength(1);
	});

	it('nimmt nach trenne() bei erneutem verbinde() wieder auf', () => {
		verbinde();
		trenne();
		verbinde();

		expect(FakeEventSource.instanzen).toHaveLength(2);
		expect(FakeEventSource.letzte.geschlossen).toBe(false);
	});

	// Eine Ansicht, die sich anmeldet, während die Leitung schon steht, muss Ereignisse
	// bekommen — sonst hinge die Aktualisierung davon ab, wer zuerst geladen wurde.
	it('zieht Zuhörer nach, die sich erst nach dem Verbinden anmelden', () => {
		verbinde();
		const handler = vi.fn();
		abonniere('action', handler);

		FakeEventSource.letzte.sende('action');

		expect(handler).toHaveBeenCalledTimes(1);
	});

	it('behält Zuhörer über einen Wiederaufbau hinweg', () => {
		const handler = vi.fn();
		abonniere('action', handler);
		verbinde();

		FakeEventSource.letzte.scheitert();
		vi.advanceTimersByTime(2000);
		FakeEventSource.letzte.sende('action');

		expect(handler).toHaveBeenCalledTimes(1);
	});
});
