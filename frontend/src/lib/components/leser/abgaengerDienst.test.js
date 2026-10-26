import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiFetch: vi.fn()
}));
vi.mock('../../../inventur/lib/store.svelte.js', () => ({ melde: vi.fn() }));
import { apiFetch, FRIST_SAMMELVERSAND_MS } from '../../apiFetch.js';
import { melde } from '../../../inventur/lib/store.svelte.js';
import { sendeKontoauszuege, kontoauszugAdresse, zeigeKontoauszuege } from './abgaengerDienst.js';
import { erzeugeSammelversand } from '../../stores/sammelversand.svelte.js';
import { serverAntwort } from '../../serverAntwort.js';

// Der Versand der Kontoauszüge an die Klassenleitungen hat im Browser nur ein Saisonfenster:
// Die beiden e2e-Specs (abgaenger-versand, schueler-profil-klick) überspringen sich vom
// 01.08. bis 30.04., weil der Server das Fenster aus seiner Uhr rechnet. Ein Bruch in der
// Nutzlast fiele sonst erst im Mai auf — mitten in der Saison (Bestands-Durchgang
// 10.09.2026, „Kalender-Gate"). Dieser Test hält die Nutzlast das ganze Jahr, über denselben
// Sammelversand, den Abgaenger.svelte baut.
describe('abgaengerDienst.sendeKontoauszuege', () => {
	const versand = erzeugeSammelversand(sendeKontoauszuege, () => 'Kontoauszüge versendet');

	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(melde).mockReset();
	});

	it('schickt genau die gewählten Klassen und die Ausnahme-Adresse', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			/** @type {any} */ ({ ok: true, json: async () => ({ message: '2 Mails versendet.' }) })
		);
		await versand.sende({ klassen: ['09H1', '10R2'], overrideEmail: 'test@schule.example' });

		const [url, opts] = vi.mocked(apiFetch).mock.calls[0];
		expect(url).toBe('/api/abgaenger/mail');
		expect(opts?.method).toBe('POST');
		// Je Klasse eine Mail: länger als die Vorgabe für Anfragen.
		expect(opts?.timeoutMs).toBe(FRIST_SAMMELVERSAND_MS);
		expect(JSON.parse(String(opts?.body))).toEqual({
			klassen: ['09H1', '10R2'],
			override_email: 'test@schule.example'
		});
		expect(melde).toHaveBeenCalledWith('2 Mails versendet.', 'success');
	});

	it('ohne Ausnahme-Adresse ein leerer String, nie ein fehlendes Feld', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			/** @type {any} */ ({ ok: true, json: async () => ({}) })
		);
		await versand.sende({ klassen: ['09H1'] });
		const body = JSON.parse(String(vi.mocked(apiFetch).mock.calls[0][1]?.body));
		expect(body).toEqual({ klassen: ['09H1'], override_email: '' });
		expect(melde).toHaveBeenCalledWith('Kontoauszüge versendet', 'success');
	});

	it('meldet den Fehler des Servers, nicht eine eigene Formulierung', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			serverAntwort(502, { error: 'Mailserver nicht erreichbar' })
		);
		await versand.sende({ klassen: ['09H1'] });
		expect(melde).toHaveBeenCalledWith('Mailserver nicht erreichbar', 'error');
	});

	// Ohne Rumpf, etwa bei 502 vom Proxy während eines Neustarts, las der Versand die Antwort als
	// JSON, bevor er nach res.ok fragte, und meldete einen Satz des Browsers.
	it('nennt ohne Rumpf die Vorgabe, keinen Satz des Browsers', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(serverAntwort(502));
		await versand.sende({ klassen: ['09H1'] });
		expect(melde).toHaveBeenCalledWith('Versand fehlgeschlagen.', 'error');
	});
});

// Der Druck folgt der Suche (seit 21.09.2026). Auch das sieht im Browser nur die Saison:
// Außerhalb von Mai bis Juli ist die Liste leer und der Druck 404. Die Adresse wird deshalb
// hier festgehalten — was der Server aus ihr macht, hält api/mahnwesen/abgaenger_druck_auswahl_pg_test.go.
describe('abgaengerDienst.kontoauszugAdresse', () => {
	const SICHTBAR = [{ id: 'a-1' }, { id: 'b-2' }];

	it('ohne Suche gilt der Klassenfilter allein — wie bisher', () => {
		expect(kontoauszugAdresse('', '', SICHTBAR)).toBe('/api/abgaenger/pdf');
		expect(kontoauszugAdresse('10R1', '', SICHTBAR)).toBe('/api/abgaenger/pdf?klasse=10R1');
		// Nur Leerzeichen ist keine Suche: Die Liste filtert dann auch nicht.
		expect(kontoauszugAdresse('', '   ', SICHTBAR)).toBe('/api/abgaenger/pdf');
	});

	it('bei aktiver Suche gehen die Kennungen der sichtbaren Zeilen mit', () => {
		const adresse = new URL(kontoauszugAdresse('', 'müller', SICHTBAR), 'http://x');
		expect(adresse.pathname).toBe('/api/abgaenger/pdf');
		expect(adresse.searchParams.get('ids')).toBe('a-1,b-2');
		expect(adresse.searchParams.has('klasse')).toBe(false);
	});

	it('Klasse und Suche gelten zusammen', () => {
		const adresse = new URL(kontoauszugAdresse('09H1', 'an', [{ id: 'a-1' }]), 'http://x');
		expect(adresse.searchParams.get('klasse')).toBe('09H1');
		expect(adresse.searchParams.get('ids')).toBe('a-1');
	});

	it('eine Suche ohne Treffer ist ein Fehler — nie `ids=`, das hieße „alle"', async () => {
		expect(() => kontoauszugAdresse('', 'zzz', [])).toThrow(/niemanden/);

		vi.mocked(apiFetch).mockReset();
		vi.stubGlobal('open', vi.fn());
		await expect(zeigeKontoauszuege('', 'zzz', [])).rejects.toThrow(/niemanden/);
		expect(apiFetch).not.toHaveBeenCalled();
		expect(window.open, 'kein leerer Tab für eine Suche ohne Treffer').not.toHaveBeenCalled();
		vi.unstubAllGlobals();
	});
});

// Die Kontoauszüge tragen die Namen der Abgänger; als Download blieben sie im Download-Ordner
// des Arbeitsplatzes liegen. Sie gehen im Tab auf (utils/pdfImFenster.js); den Dateinamen fürs
// Speichern nennt der Server (api/mahnwesen/abgaenger_druck_auswahl_pg_test.go).
describe('abgaengerDienst.zeigeKontoauszuege: im Tab', () => {
	const erzeuge = URL.createObjectURL;
	/** @type {{ closed: boolean, close: any, location: { replace: any } } | null} */
	let fenster;

	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, blob: async () => new Blob(['%PDF']) })
		);
		fenster = { closed: false, close: vi.fn(), location: { replace: vi.fn() } };
		vi.stubGlobal(
			'open',
			vi.fn(() => fenster)
		);
		URL.createObjectURL = vi.fn(() => 'blob:kontoauszug');
	});
	afterEach(() => {
		vi.restoreAllMocks();
		vi.unstubAllGlobals();
		URL.createObjectURL = erzeuge;
	});

	it('zeigt das PDF im Tab und lädt keine Datei herunter', async () => {
		const anker = vi.spyOn(HTMLAnchorElement.prototype, 'click');

		expect(await zeigeKontoauszuege('10R1')).toBe(true);

		expect(apiFetch).toHaveBeenCalledWith('/api/abgaenger/pdf?klasse=10R1');
		expect(fenster?.location.replace).toHaveBeenCalledWith('blob:kontoauszug');
		expect(anker).not.toHaveBeenCalled();
	});

	it('meldet einen blockierten Tab und fragt nicht beim Server an', async () => {
		fenster = null;

		expect(await zeigeKontoauszuege('10R1')).toBe(false);
		expect(apiFetch).not.toHaveBeenCalled();
	});
});
