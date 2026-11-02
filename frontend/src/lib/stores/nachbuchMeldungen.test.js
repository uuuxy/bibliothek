import { describe, it, expect, vi, beforeEach } from 'vitest';
import { serverAntwort } from '../serverAntwort.js';

// Der Zähler gehört jeder Theken-Rolle, die Liste nur `view_students` — beides prüft der
// Server. Hier steht, was der Browser daraus macht: dass er die WIRKUNG liest (die Zeile
// verschwindet erst, wenn der Server zugestimmt hat), dass ein 403 in einem Satz steht,
// den ein Mensch versteht, und dass die Zahl auf das Live-Signal hin neu geholt wird.
//
// Bis zum 16.09.2026 gab es diese Tür im Browser gar nicht: Der Server schrieb Meldungen,
// die niemand zu sehen bekam (Bugklasse „Nie verdrahtet", OFFEN.md Abschnitt 2).

/** @type {{ handler: null | (() => void) }} */
const sse = { handler: null };
vi.mock('../liveEvents.js', () => ({
	abonniereNeuladen: (_name, cb) => {
		sse.handler = cb;
		return () => (sse.handler = null);
	}
}));
const fetchMock = vi.fn();
const postMock = vi.fn();
vi.mock('../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: (...args) => fetchMock(...args),
	apiClient: { post: (...args) => postMock(...args) }
}));

const { nachbuchMeldungen: m } = await import('./nachbuchMeldungen.svelte.js');
const { snackbars } = await import('./snackbars.svelte.js');

/** @param {any} daten @param {number} status */
const antwort = (daten, status = 200) => ({ ok: status < 400, status, json: async () => daten });

const zeile = (id, ergebnis = 'umgebucht') => ({
	id,
	barcode: 'B-1',
	titel: 'Momo',
	ergebnis,
	gescannt_am: '2026-09-16T10:00:00Z'
});

describe('Meldungen aus dem Nachbuchen', () => {
	beforeEach(() => {
		m._zuruecksetzen();
		vi.clearAllMocks();
		fetchMock.mockReset();
	});

	it('holt die Zahl fürs Band', async () => {
		fetchMock.mockResolvedValue(antwort({ offen: 3 }));
		await m.zaehle();
		expect(m.offen).toBe(3);
	});

	it('lässt die Zahl stehen, wenn der Zähler nicht erreichbar ist', async () => {
		fetchMock.mockResolvedValue(antwort({ offen: 2 }));
		await m.zaehle();
		fetchMock.mockRejectedValue(new TypeError('Failed to fetch'));
		await m.zaehle();
		expect(m.offen, 'eine alte Zahl ist besser als eine falsche 0').toBe(2);
	});

	it('sagt bei fehlendem Recht, woran es liegt', async () => {
		fetchMock.mockResolvedValue(antwort({}, 403));
		await m.lade();
		expect(m.ladefehler).toMatch(/Recht/);
		expect(m.liste).toHaveLength(0);
	});

	it('nimmt die Zahl aus der geladenen Liste — Band und Liste zeigen dasselbe', async () => {
		fetchMock.mockResolvedValue(antwort([zeile('a'), zeile('b')]));
		await m.lade();
		expect(m.liste).toHaveLength(2);
		expect(m.offen).toBe(2);
	});

	// Erledigt ist, was der Server entschieden hat: Erst danach wird neu geladen.
	it('quittiert und holt die Liste danach neu', async () => {
		postMock.mockResolvedValue(antwort({ status: 'ok' }));
		fetchMock.mockResolvedValue(antwort([zeile('b')]));

		await m.quittiere('a');

		expect(postMock).toHaveBeenCalledWith('/api/action/nachbuch-meldungen/a/quittieren', {});
		expect(m.liste.map((z) => z.id)).toEqual(['b']);
		expect(m.fehler).toBe('');
	});

	// Ein anderer Arbeitsplatz war schneller: Das ist kein Fehler dieses Bedieners.
	it('behandelt eine schon quittierte Meldung (404) nicht als Fehler', async () => {
		postMock.mockResolvedValue(antwort({}, 404));
		fetchMock.mockResolvedValue(antwort([]));

		await m.quittiere('a');

		expect(m.fehler).toBe('');
		expect(m.liste).toHaveLength(0);
	});

	it('meldet einen echten Fehlschlag beim Quittieren', async () => {
		postMock.mockResolvedValue(antwort({}, 500));
		await m.quittiere('a');
		expect(m.fehler).toMatch(/500/);
	});

	// Lehnt der Server ab, steht sein Grund da; der eigene Satz mit dem Status bleibt für eine
	// Antwort ohne Rumpf.
	it('nennt beim Quittieren den Grund, den der Server angibt', async () => {
		postMock.mockResolvedValue(
			serverAntwort(500, { error: 'Nachbuch-Meldung konnte nicht quittiert werden' })
		);
		await m.quittiere('a');
		expect(m.fehler).toBe('Nachbuch-Meldung konnte nicht quittiert werden');
	});

	it('nennt beim Laden den Grund, den der Server angibt', async () => {
		fetchMock.mockResolvedValue(
			serverAntwort(500, { error: 'Ein interner Serverfehler ist aufgetreten' })
		);
		await m.lade();
		expect(m.ladefehler).toBe('Ein interner Serverfehler ist aufgetreten');
	});

	// Signale kommen kurz nacheinander, und Antworten überholen sich.
	describe('die jüngste Anfrage gilt', () => {
		/** @returns {{ antworte: (wert: any) => void, scheitere: (grund: Error) => void, zusage: Promise<any> }} */
		function offeneAntwort() {
			/** @type {any} */
			let antworte;
			/** @type {any} */
			let scheitere;
			const zusage = new Promise((fertig, nein) => {
				antworte = fertig;
				scheitere = nein;
			});
			return { antworte, scheitere, zusage };
		}

		it('das Band springt nicht auf die Zahl der älteren Antwort zurück', async () => {
			const alt = offeneAntwort();
			fetchMock.mockReturnValueOnce(alt.zusage);
			const aelter = m.zaehle();
			fetchMock.mockResolvedValueOnce(antwort({ offen: 0 }));
			await m.zaehle();
			expect(m.offen).toBe(0);

			alt.antworte(antwort({ offen: 5 }));
			await aelter;

			expect(m.offen).toBe(0);
		});

		it('die Zahl aus der Liste schlägt einen älteren Zähler, auch in der anderen Richtung', async () => {
			const alt = offeneAntwort();
			fetchMock.mockReturnValueOnce(alt.zusage);
			const aelter = m.zaehle();
			fetchMock.mockResolvedValueOnce(antwort([zeile('a')]));
			await m.lade();
			expect(m.offen).toBe(1);
			alt.antworte(antwort({ offen: 5 }));
			await aelter;
			expect(m.offen).toBe(1);

			const liste = offeneAntwort();
			fetchMock.mockReturnValueOnce(liste.zusage);
			const ladend = m.lade();
			fetchMock.mockResolvedValueOnce(antwort({ offen: 7 }));
			await m.zaehle();
			expect(m.offen).toBe(7);
			liste.antworte(antwort([zeile('a')]));
			await ladend;
			expect(m.offen).toBe(7);
		});

		// Scheitert die jüngere, bleibt die Zahl der älteren, die geantwortet hat.
		it('eine gescheiterte jüngere Anfrage nimmt der älteren nichts', async () => {
			fetchMock.mockResolvedValueOnce(antwort({ offen: 0 }));
			await m.zaehle();
			const alt = offeneAntwort();
			fetchMock.mockReturnValueOnce(alt.zusage);
			const aelter = m.zaehle();
			fetchMock.mockRejectedValueOnce(new TypeError('Failed to fetch'));
			await m.zaehle();

			alt.antworte(antwort({ offen: 3 }));
			await aelter;

			expect(m.offen).toBe(3);
		});

		it('die Liste der vorigen Ansicht ersetzt die neue nicht und beendet das Laden nicht', async () => {
			const alt = offeneAntwort();
			fetchMock.mockReturnValueOnce(alt.zusage);
			const aelter = m.lade();
			const neu = offeneAntwort();
			fetchMock.mockReturnValueOnce(neu.zusage);
			m.auchQuittierte = true;
			const juenger = m.lade();

			alt.antworte(antwort([zeile('alt')]));
			await aelter;
			expect(m.liste).toHaveLength(0);
			expect(m.laeuft).toBe(true);

			neu.antworte(antwort([zeile('neu')]));
			await juenger;
			expect(m.liste.map((z) => z.id)).toEqual(['neu']);
			expect(m.laeuft).toBe(false);
		});

		it('eine späte Absage nennt keinen Fehler, wenn die jüngere Anfrage kam', async () => {
			for (const t of [...snackbars.toasts]) snackbars.removeToast(t.id);
			const alt = offeneAntwort();
			fetchMock.mockReturnValueOnce(alt.zusage);
			const aelter = m.lade();
			fetchMock.mockResolvedValueOnce(antwort([zeile('neu')]));
			await m.lade();

			alt.antworte(serverAntwort(500, { error: 'Zu spät' }));
			await aelter;

			expect(m.ladefehler).toBe('');
			expect(snackbars.toasts).toHaveLength(0);
			expect(m.laeuft).toBe(false);
		});

		it('auch eine späte Absage ohne Antwort nennt keinen Fehler', async () => {
			for (const t of [...snackbars.toasts]) snackbars.removeToast(t.id);
			const alt = offeneAntwort();
			fetchMock.mockReturnValueOnce(alt.zusage);
			const aelter = m.lade();
			fetchMock.mockResolvedValueOnce(antwort([zeile('neu')]));
			await m.lade();

			alt.scheitere(new TypeError('Failed to fetch'));
			await aelter;

			expect(m.ladefehler).toBe('');
			expect(snackbars.toasts).toHaveLength(0);
		});
	});

	// Die Liste steht offen, während an einem anderen Platz quittiert wird. Scheitert das Auffrischen,
	// zeigt der Dialog weiter, was er weiß; „nicht geladen" nähme die Arbeit weg.
	describe('das Auffrischen scheitert', () => {
		beforeEach(async () => {
			for (const t of [...snackbars.toasts]) snackbars.removeToast(t.id);
			fetchMock.mockResolvedValueOnce(antwort([zeile('a')]));
			await m.lade();
		});

		it('die Liste bleibt stehen, und eine Meldung nennt den Grund', async () => {
			fetchMock.mockResolvedValueOnce(serverAntwort(503, { error: 'Datenbank kurz weg' }));
			await m.lade();

			expect(m.liste.map((z) => z.id)).toEqual(['a']);
			expect(m.ladefehler).toBe('');
			expect(snackbars.toasts.map((t) => t.message)).toEqual([
				'Die Liste wurde nicht aktualisiert: Datenbank kurz weg'
			]);

			fetchMock.mockRejectedValueOnce(new TypeError('Failed to fetch'));
			await m.lade();
			expect(m.liste.map((z) => z.id)).toEqual(['a']);
			expect(m.ladefehler).toBe('');
		});

		// Unter „Auch erledigte zeigen" stünde sonst die Liste der offenen.
		it('scheitert der Wechsel der Ansicht, steht die alte Liste nicht unter der neuen', async () => {
			fetchMock.mockResolvedValueOnce(serverAntwort(503, { error: 'Datenbank kurz weg' }));
			m.auchQuittierte = true;
			await m.lade();

			expect(m.ladefehler).toBe('Datenbank kurz weg');
			expect(snackbars.toasts).toHaveLength(0);
		});
	});

	// Der Store überlebt das Abmelden. Die Liste nennt Namen und Klassen, und das nächste Konto hat das
	// Recht vielleicht nicht.
	describe('die Liste bleibt nicht liegen', () => {
		beforeEach(async () => {
			for (const t of [...snackbars.toasts]) snackbars.removeToast(t.id);
			fetchMock.mockResolvedValueOnce(antwort([zeile('a')]));
			await m.lade();
		});

		it('fehlt das Recht, geht auch die Liste von vorhin, ohne Meldung', async () => {
			fetchMock.mockResolvedValueOnce(antwort({}, 403));
			await m.lade();

			expect(m.liste).toHaveLength(0);
			expect(m.ladefehler).toMatch(/Recht/);
			expect(snackbars.toasts).toHaveLength(0);
		});

		it('das Schließen leert sie, und ein späteres Scheitern zeigt sie nicht wieder', async () => {
			m.schliesse();
			expect(m.liste).toHaveLength(0);

			fetchMock.mockResolvedValueOnce(serverAntwort(503, { error: 'Datenbank kurz weg' }));
			await m.lade();

			expect(m.ladefehler).toBe('Datenbank kurz weg');
			expect(snackbars.toasts).toHaveLength(0);
		});

		it('eine Antwort, die nach dem Schließen kommt, schreibt nichts und beendet nichts', async () => {
			/** @type {(wert: any) => void} */
			let antworte = () => {};
			fetchMock.mockReturnValueOnce(new Promise((fertig) => (antworte = fertig)));
			const ladend = m.lade();
			m.schliesse();
			expect(m.laeuft).toBe(false);

			antworte(antwort([zeile('spaet')]));
			await ladend;

			expect(m.liste).toHaveLength(0);
		});
	});

	// Der Store überlebt das Abmelden, das Abo nicht: Sonst stünde die Zahl der vorigen Sitzung im
	// Band, auch auf der Anmeldeseite, und ein Konto ohne Theken-Recht fragte sie weiter ab.
	describe('die Zahl gehört zur Anmeldung', () => {
		it('das Abmelden beendet das Abo und leert die Zahl', async () => {
			fetchMock.mockResolvedValue(antwort({ offen: 3 }));
			const beende = m.init();
			await vi.waitFor(() => expect(m.offen).toBe(3));

			beende();
			expect(sse.handler, 'nach dem Abmelden hört der Store nicht mehr zu').toBeNull();
			expect(m.offen, 'die Zahl der vorigen Sitzung steht nicht im Band').toBe(0);

			m.init();
			expect(sse.handler, 'die nächste Anmeldung hört wieder zu').not.toBeNull();
		});

		it('eine Zahl, die nach dem Abmelden kommt, steht nicht im Band', async () => {
			/** @type {(wert: any) => void} */
			let antworte = () => {};
			fetchMock.mockReturnValueOnce(new Promise((fertig) => (antworte = fertig)));
			const beende = m.init();
			beende();

			antworte(antwort({ offen: 4 }));
			await new Promise((fertig) => setTimeout(fertig, 0));

			expect(m.offen).toBe(0);
		});
	});

	// Die Leitung holt nach, wenn eine Antwort nicht kam (liveEvents.js). Ein fehlendes Recht kommt
	// auch beim nächsten Versuch nicht.
	describe('was der Leitung gemeldet wird', () => {
		it('der Zähler meldet, ob die Antwort kam', async () => {
			fetchMock.mockResolvedValueOnce(antwort({ offen: 1 }));
			expect(await m.zaehle()).toBe(true);
			fetchMock.mockResolvedValueOnce(antwort({}, 500));
			expect(await m.zaehle()).toBe(false);
			fetchMock.mockRejectedValueOnce(new TypeError('Failed to fetch'));
			expect(await m.zaehle()).toBe(false);
		});

		it('ein fehlendes Recht beim Zähler zählt nicht als nachzuholen', async () => {
			fetchMock.mockResolvedValueOnce(antwort({}, 403));
			expect(await m.zaehle()).toBe(true);
		});

		it('die Liste meldet es ebenso, ein fehlendes Recht zählt nicht als nachzuholen', async () => {
			fetchMock.mockResolvedValueOnce(antwort([]));
			expect(await m.lade()).toBe(true);
			fetchMock.mockResolvedValueOnce(antwort({}, 500));
			expect(await m.lade()).toBe(false);
			fetchMock.mockRejectedValueOnce(new TypeError('Failed to fetch'));
			expect(await m.lade()).toBe(false);
			fetchMock.mockResolvedValueOnce(antwort({}, 403));
			expect(await m.lade()).toBe(true);
		});

		it('das Signal meldet false, wenn Zahl oder offene Liste nicht kamen', async () => {
			fetchMock.mockResolvedValue(antwort({ offen: 0 }));
			m.init();
			await vi.waitFor(() => expect(fetchMock).toHaveBeenCalled());

			fetchMock.mockResolvedValue(antwort({}, 500));
			expect(await sse.handler?.()).toBe(false);

			fetchMock.mockResolvedValue(antwort({ offen: 2 }));
			expect(await sse.handler?.()).toBe(true);

			m.geoeffnet = true;
			fetchMock.mockImplementation(async (/** @type {string} */ url) =>
				url.endsWith('/anzahl') ? antwort({ offen: 2 }) : antwort({}, 500)
			);
			expect(await sse.handler?.()).toBe(false);
		});
	});

	// Der zweite Arbeitsplatz zeigte sonst die alte Zahl bis zur nächsten Anmeldung.
	it('holt die Zahl neu, wenn der Server eine Änderung meldet', async () => {
		fetchMock.mockResolvedValue(antwort({ offen: 0 }));
		m.init();
		expect(sse.handler, 'ohne Abo bleibt das Band stehen').not.toBeNull();

		fetchMock.mockResolvedValue(antwort({ offen: 5 }));
		sse.handler?.();
		await vi.waitFor(() => expect(m.offen).toBe(5));
	});
});
