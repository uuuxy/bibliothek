import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import KollegiumPortal from './KollegiumPortal.svelte';
import { apiFetch } from './apiFetch.js';

vi.mock('./apiFetch.js', () => ({ apiFetch: vi.fn() }));

/**
 * Eine Lehrkraft bestellt einen Klassensatz für 8a — und danach denselben Titel für 8b.
 *
 * Genau das ging nicht: Nach dem Absenden ersetzte das Badge „✓ Gesendet" den Knopf
 * dauerhaft. Aufgeräumt wird der Formularzustand aber ausgerechnet in toggleForm, und
 * das hing an diesem Knopf — der einzige Rückweg war ein Seitenreload. Kein Fehler, kein
 * Statuscode, nichts zu sehen: die Anfrage war ja erfolgreich.
 *
 * Der Test hält deshalb nicht das Badge fest, sondern die Handlungsfähigkeit danach.
 */
const TITEL = 'Seydlitz Geographie';

/**
 * Antwort des Katalogs des Kollegiums (api/katalog_kollegium.go) — ein nacktes Array, kein
 * `{books: …}`-Umschlag, mit den Bestandszahlen und dem Kopf X-Treffer-Gesamt.
 */
function suchtreffer(verfuegbar = 12, gesamt = 30, im_zulauf = 0) {
	return {
		ok: true,
		headers: new Headers({ 'X-Treffer-Gesamt': '1' }),
		json: async () => [
			{ id: 'titel-1', titel: TITEL, autor: 'Klaus Berger', verfuegbar, gesamt, im_zulauf }
		]
	};
}

/** Sucht wie das Portal: debounced, deshalb über findBy* abwarten. */
async function sucheUndOeffneFormular(screen) {
	await fireEvent.input(
		screen.getByRole('searchbox', { name: 'Bücher für einen Klassensatz suchen' }),
		{
			target: { value: 'Seydlitz' }
		}
	);
	const knopf = await screen.findByRole('button', { name: 'Klassensatz reservieren' });
	await fireEvent.click(knopf);
	await fireEvent.input(await screen.findByLabelText('Klasse / Kurs *'), {
		target: { value: '08a' }
	});
	await fireEvent.click(screen.getByRole('button', { name: /Anfrage senden/ }));
}

describe('KollegiumPortal', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockImplementation(
			/** @type {any} */ (
				async (/** @type {string} */ url) =>
					url.startsWith('/api/reservierungen/klassensatz/katalog')
						? suchtreffer()
						: { ok: true, text: async () => '', json: async () => ({}) }
			)
		);
	});

	it('lässt nach einer gesendeten Anfrage sofort die nächste Klasse zu', async () => {
		const screen = render(KollegiumPortal, { user: { klasse: '' } });

		await sucheUndOeffneFormular(screen);

		expect(await screen.findByText('✓ Gesendet')).toBeTruthy();

		// Der Kern: Der Weg zur nächsten Reservierung ist offen — ohne Reload.
		const erneut = await screen.findByRole('button', { name: 'Weitere Klasse reservieren' });
		expect(erneut.hasAttribute('disabled')).toBe(false);

		await fireEvent.click(erneut);
		expect(await screen.findByLabelText('Klasse / Kurs *')).toBeTruthy();
	});

	/**
	 * Der Bestand MUSS am Treffer stehen — sonst kann eine Lehrkraft nicht entscheiden,
	 * ob ein Klassensatz für ihre Gruppe überhaupt reicht.
	 *
	 * Das war lange kaputt und völlig unsichtbar: Das Portal fragte `/api/search`, das
	 * `BookTitle` ohne Bestandsfeld liefert. Das Abzeichen hängt an `{#if book.verfuegbar
	 * != null}` — ein Wächter, der still übersprang. Kein Fehler, keine Lücke im Layout,
	 * die Zahl fehlte einfach. Deshalb prüft dieser Test den TEXT, nicht das Vorhandensein
	 * eines Elements.
	 */
	it('zeigt am Treffer, wie viele Exemplare frei sind und wie viele es gibt', async () => {
		const screen = render(KollegiumPortal, { user: { klasse: '' } });

		await fireEvent.input(
			screen.getByRole('searchbox', { name: 'Bücher für einen Klassensatz suchen' }),
			{
				target: { value: 'Seydlitz' }
			}
		);

		expect(await screen.findByText('12 von 30 verfügbar')).toBeTruthy();
	});

	it('nennt bei vergriffenem Titel trotzdem den Gesamtbestand', async () => {
		vi.mocked(apiFetch).mockImplementation(
			/** @type {any} */ (
				async (/** @type {string} */ url) =>
					url.startsWith('/api/reservierungen/klassensatz/katalog')
						? suchtreffer(0, 30)
						: { ok: true, text: async () => '', json: async () => ({}) }
			)
		);

		const screen = render(KollegiumPortal, { user: { klasse: '' } });
		await fireEvent.input(
			screen.getByRole('searchbox', { name: 'Bücher für einen Klassensatz suchen' }),
			{
				target: { value: 'Seydlitz' }
			}
		);

		// „nicht verfügbar" allein hieße für die Lehrkraft: gibt es hier gar nicht.
		expect(await screen.findByText('nicht verfügbar (30 im Bestand)')).toBeTruthy();
	});

	// Ein Titel, dessen Exemplare bestellt und noch nicht eingetroffen sind, steht in der
	// Trefferliste und lässt sich reservieren. „nicht verfügbar (0 im Bestand)" hieße für
	// die Lehrkraft: gibt es nicht.
	it('nennt einen Titel, der nur bestellt ist, „bestellt" und lässt ihn reservieren', async () => {
		vi.mocked(apiFetch).mockImplementation(
			/** @type {any} */ (
				async (/** @type {string} */ url) =>
					url.startsWith('/api/reservierungen/klassensatz/katalog')
						? suchtreffer(0, 0, 30)
						: { ok: true, text: async () => '', json: async () => ({}) }
			)
		);

		const screen = render(KollegiumPortal, { user: { klasse: '' } });
		await fireEvent.input(
			screen.getByRole('searchbox', { name: 'Bücher für einen Klassensatz suchen' }),
			{ target: { value: 'Seydlitz' } }
		);

		expect(await screen.findByText('30 bestellt')).toBeTruthy();
		expect(screen.queryByText(/nicht verfügbar/)).toBeNull();
		expect(screen.getByRole('button', { name: 'Klassensatz reservieren' })).toBeTruthy();
	});

	it('meldet einen abgelehnten Versuch und blockiert den Knopf nicht', async () => {
		vi.mocked(apiFetch).mockImplementation(
			/** @type {any} */ (
				async (/** @type {string} */ url) =>
					url.startsWith('/api/reservierungen/klassensatz/katalog')
						? suchtreffer()
						: { ok: false, text: async () => 'Titel ist gesperrt.' }
			)
		);

		const screen = render(KollegiumPortal, { user: { klasse: '' } });
		await sucheUndOeffneFormular(screen);

		expect(await screen.findByText('Titel ist gesperrt.')).toBeTruthy();
		expect(screen.queryByText('✓ Gesendet')).toBeNull();
		expect(screen.getByRole('button', { name: /Anfrage senden/ }).hasAttribute('disabled')).toBe(
			false
		);
	});
});

/** Der Rumpf der letzten Anfrage an die Tür für Anliegen. */
function gemeldet() {
	const ruf = vi.mocked(apiFetch).mock.calls.findLast(([url]) => url === '/api/anliegen');
	return ruf ? JSON.parse(/** @type {any} */ (ruf[1]).body) : null;
}

/** Katalog und Annahme der Meldung antworten, alles andere ist leer. */
function portalMitTreffer() {
	vi.mocked(apiFetch).mockReset();
	vi.mocked(apiFetch).mockImplementation(
		/** @type {any} */ (
			async (/** @type {string} */ url) =>
				url.startsWith('/api/reservierungen/klassensatz/katalog')
					? suchtreffer()
					: { ok: true, text: async () => '', json: async () => ({}) }
		)
	);
	return render(KollegiumPortal, { user: { klasse: '' } });
}

/** @param {any} screen */
async function suche(screen) {
	await fireEvent.input(
		screen.getByRole('searchbox', { name: 'Bücher für einen Klassensatz suchen' }),
		{ target: { value: 'Seydlitz' } }
	);
	await screen.findByRole('button', { name: 'Klassensatz reservieren' });
}

/**
 * „Problem melden" steht am Treffer: Das Buch ist gewählt, sein Titel geht in die Meldung.
 * Ein zweites Feld, in das die Lehrkraft das Buch noch einmal tippt, gibt es nicht.
 */
describe('Problem melden im Portal', () => {
	it('meldet am Treffer mit dem Titel des Buchs und verlangt die Beschreibung', async () => {
		const screen = portalMitTreffer();
		await suche(screen);

		await fireEvent.click(screen.getByRole('button', { name: `Problem melden zu ${TITEL}` }));
		expect(screen.queryByLabelText('Worum geht es? *'), 'das Buch ist schon gewählt').toBeNull();
		const absenden = /** @type {HTMLButtonElement} */ (
			await screen.findByRole('button', { name: 'Absenden' })
		);
		expect(absenden.disabled, 'ohne Beschreibung').toBe(true);

		await fireEvent.input(screen.getByLabelText('Klasse / Kurs'), { target: { value: '8G3' } });
		await fireEvent.input(screen.getByLabelText('Was stimmt nicht? *'), {
			target: { value: 'falsche Auflage' }
		});
		await fireEvent.click(absenden);

		await vi.waitFor(() => expect(gemeldet()).toBeTruthy());
		expect(gemeldet()).toEqual({
			art: 'meldung',
			titel_text: TITEL,
			klasse: '8G3',
			kommentar: 'falsche Auflage'
		});
		// Danach ist das Formular zu, und der Treffer steht wie vorher da.
		await vi.waitFor(() => expect(screen.queryByRole('button', { name: 'Absenden' })).toBeNull());
		expect(screen.getByRole('button', { name: 'Klassensatz reservieren' })).toBeTruthy();
	});

	it('hält je Treffer höchstens ein Formular offen', async () => {
		const screen = portalMitTreffer();
		await suche(screen);

		await fireEvent.click(screen.getByRole('button', { name: 'Klassensatz reservieren' }));
		expect(await screen.findByRole('button', { name: 'Anfrage senden' })).toBeTruthy();

		await fireEvent.click(screen.getByRole('button', { name: `Problem melden zu ${TITEL}` }));
		expect(await screen.findByRole('button', { name: 'Absenden' })).toBeTruthy();
		expect(screen.queryByRole('button', { name: 'Anfrage senden' })).toBeNull();

		await fireEvent.click(screen.getByRole('button', { name: 'Klassensatz reservieren' }));
		expect(await screen.findByRole('button', { name: 'Anfrage senden' })).toBeTruthy();
		expect(screen.queryByRole('button', { name: 'Absenden' })).toBeNull();
	});

	// Ein Reiter für alles, was an die Bibliothek geht: „Meine Anliegen" gibt es nicht mehr,
	// einen Buchwunsch auch nicht. Ohne Buch steht „Problem melden" unter der Suche.
	it('meldet ohne Buch unter der Suche, ohne eigenen Reiter', async () => {
		const screen = portalMitTreffer();

		expect(screen.getAllByRole('tab').map((r) => r.textContent?.trim())).toEqual([
			'Reservieren & Melden',
			'Klassensätze',
			'Schulbücher',
			'LMF-Plan'
		]);
		expect(screen.queryByRole('button', { name: 'Buchwunsch' })).toBeNull();

		await fireEvent.click(screen.getByRole('button', { name: 'Problem melden' }));
		const worum = await screen.findByLabelText('Worum geht es? *');
		// Der Knopf steht direkt über dem Formular: Eine Überschrift mit denselben Worten
		// stünde dort doppelt.
		expect(screen.getAllByText('Problem melden')).toHaveLength(1);
		await fireEvent.input(worum, {
			target: { value: 'die Bücher der 8G3' }
		});
		await fireEvent.input(screen.getByLabelText('Was stimmt nicht? *'), {
			target: { value: 'drei fehlen' }
		});
		await fireEvent.click(screen.getByRole('button', { name: 'Absenden' }));

		await vi.waitFor(() => expect(gemeldet()).toBeTruthy());
		expect(gemeldet()).toEqual({
			art: 'meldung',
			titel_text: 'die Bücher der 8G3',
			klasse: '',
			kommentar: 'drei fehlen'
		});
		await vi.waitFor(() =>
			expect(screen.queryByRole('button', { name: 'Problem melden' })).toBeTruthy()
		);
	});

	// Der Knopf ohne Buch bleibt beim Tippen stehen, und der Platzhalter nennt beide Zwecke:
	// Wer ein Problem ins Suchfeld schreibt, schickt es von dort ab.
	it('lässt „Problem melden" beim Tippen stehen und übernimmt das Getippte', async () => {
		const screen = portalMitTreffer();
		expect(screen.getByPlaceholderText('Buch suchen für Reservierung oder Meldung …')).toBeTruthy();
		await suche(screen);

		await fireEvent.click(screen.getByRole('button', { name: 'Problem melden' }));
		const worum = /** @type {HTMLInputElement} */ (
			await screen.findByLabelText('Worum geht es? *')
		);
		expect(worum.value).toBe('Seydlitz');
	});

	// Solange nichts gesucht wird, steht unter der Suche, was die Lehrkraft geschickt hat:
	// Reservierungen und Meldungen an einer Stelle. Beim Suchen weichen sie den Treffern.
	it('zeigt unter der Suche die eigenen Reservierungen und Meldungen', async () => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockImplementation(
			/** @type {any} */ (
				async (/** @type {string} */ url) => {
					if (url.startsWith('/api/reservierungen/klassensatz/katalog')) return suchtreffer();
					if (url === '/api/reservierungen/klassensatz/eigene') {
						return {
							ok: true,
							json: async () => [
								{
									id: 'r1',
									titel: 'Tschick',
									klasse: '8G3',
									anzahl: 28,
									erledigt: false,
									erstellt_am: '01.10.2026'
								}
							]
						};
					}
					if (url === '/api/anliegen/eigene') {
						return {
							ok: true,
							json: async () => [
								{
									id: 'a1',
									art: 'meldung',
									titel_text: 'Markl Biologie 2',
									klasse: '8G4',
									erstellt_am: 'x'
								}
							]
						};
					}
					return { ok: true, text: async () => '', json: async () => ({}) };
				}
			)
		);
		const screen = render(KollegiumPortal, { user: { klasse: '' } });

		expect(await screen.findByRole('heading', { name: 'Deine Reservierungen' })).toBeTruthy();
		expect(await screen.findByRole('heading', { name: 'Deine Meldungen' })).toBeTruthy();
		expect(screen.getByText('Markl Biologie 2')).toBeTruthy();

		await suche(screen);
		expect(screen.queryByRole('heading', { name: 'Deine Meldungen' })).toBeNull();
		expect(screen.queryByRole('heading', { name: 'Deine Reservierungen' })).toBeNull();
	});
});

/**
 * Das Warteschlangen-Modell (16.08.2026): Reservieren sperrt nichts — wer denselben
 * Titel reserviert, stellt sich an. Das Portal muss beides leisten: die bestehende
 * Reservierung VOR dem Klick zeigen und nach dem Absenden sagen, hinter wem man steht.
 */
it('zeigt die Warteschlange am Treffer und nennt nach dem Absenden den Vordermann', async () => {
	vi.mocked(apiFetch).mockImplementation(
		/** @type {any} */ (
			async (/** @type {string} */ url, /** @type {any} */ opts) => {
				if (url.startsWith('/api/reservierungen/klassensatz/katalog')) return suchtreffer();
				if (url.startsWith('/api/reservierungen/klassensatz/offen')) {
					return {
						ok: true,
						json: async () => [
							{ titel_id: 'titel-1', klasse: '8a', anzahl: 28, erstellt_am: '10.08.2026' }
						]
					};
				}
				if (url === '/api/reservierungen/klassensatz' && opts?.method === 'POST') {
					return { ok: true, text: async () => '', json: async () => ({ id: 'neu' }) };
				}
				return { ok: true, text: async () => '', json: async () => ({}) };
			}
		)
	);
	const screen = render(KollegiumPortal, { user: { klasse: '' } });

	await fireEvent.input(
		screen.getByRole('searchbox', { name: 'Bücher für einen Klassensatz suchen' }),
		{ target: { value: 'Seydlitz' } }
	);

	// Die Warteschlange steht am Treffer, BEVOR reserviert wird.
	expect(await screen.findByText('28 reserviert für 8a (seit 10.08.2026)')).toBeTruthy();
	// Und verrechnet: 12 im Regal minus 28 vorgemerkt — die Lehrkraft muss nicht rechnen.
	expect(screen.getByText('28 vorgemerkt · 0 rechnerisch frei')).toBeTruthy();

	await fireEvent.click(screen.getByRole('button', { name: 'Klassensatz reservieren' }));
	// Die Warnung steht im Formular, bevor irgendetwas gesendet wird.
	expect((await screen.findByRole('status')).textContent?.replace(/\s+/g, ' ').trim()).toBe(
		'Reicht aktuell nicht: 0 rechnerisch frei — du stellst dich hinter 8a an.'
	);
	await fireEvent.input(await screen.findByLabelText('Klasse / Kurs *'), {
		target: { value: '9b' }
	});
	await fireEvent.click(screen.getByRole('button', { name: /Anfrage senden/ }));

	// Die Bestätigung sagt, hinter wem der eigene Satz an der Reihe ist.
	expect(await screen.findByTitle(/dein Satz ist nach 8a an der Reihe/)).toBeTruthy();
});
