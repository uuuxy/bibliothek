import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('./apiFetch.js', () => ({ apiFetch: vi.fn() }));
import { apiFetch } from './apiFetch.js';
import { sendeKontoauszuege, kontoauszugAdresse, ladeKontoauszuege } from './abgaengerDienst.js';

// Der Versand der Kontoauszüge an die Klassenleitungen hat im Browser nur ein Saisonfenster:
// Die beiden e2e-Specs (abgaenger-versand, schueler-profil-klick) überspringen sich vom
// 01.08. bis 30.04., weil der Server das Fenster aus seiner Uhr rechnet. Ein Bruch in der
// Nutzlast fiele sonst erst im Mai auf — mitten in der Saison (Bestands-Durchgang
// 10.09.2026, „Kalender-Gate"). Dieser Test hält die Nutzlast das ganze Jahr.
describe('abgaengerDienst.sendeKontoauszuege', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	it('schickt genau die gewählten Klassen und die Ausnahme-Adresse', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			/** @type {any} */ ({ ok: true, json: async () => ({ message: '2 Mails versendet.' }) })
		);
		const erg = await sendeKontoauszuege({
			klassen: ['09H1', '10R2'],
			overrideEmail: 'test@schule.example'
		});

		const [url, opts] = vi.mocked(apiFetch).mock.calls[0];
		expect(url).toBe('/api/abgaenger/mail');
		expect(opts?.method).toBe('POST');
		expect(JSON.parse(String(opts?.body))).toEqual({
			klassen: ['09H1', '10R2'],
			override_email: 'test@schule.example'
		});
		expect(erg).toEqual({ ok: true, meldung: '2 Mails versendet.' });
	});

	it('ohne Ausnahme-Adresse ein leerer String, nie ein fehlendes Feld', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			/** @type {any} */ ({ ok: true, json: async () => ({}) })
		);
		await sendeKontoauszuege({ klassen: ['09H1'] });
		const body = JSON.parse(String(vi.mocked(apiFetch).mock.calls[0][1]?.body));
		expect(body).toEqual({ klassen: ['09H1'], override_email: '' });
	});

	it('meldet den Fehler des Servers, nicht eine eigene Formulierung', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: false,
				json: async () => ({ error: 'Mailserver nicht erreichbar' })
			})
		);
		const erg = await sendeKontoauszuege({ klassen: ['09H1'] });
		expect(erg).toEqual({ ok: false, meldung: 'Mailserver nicht erreichbar' });
	});
});

// Der Druck folgt der Suche (seit 21.09.2026). Auch das sieht im Browser nur die Saison:
// Außerhalb von Mai bis Juli ist die Liste leer und der Druck 404. Die Adresse wird deshalb
// hier festgehalten — was der Server aus ihr macht, hält api/abgaenger_druck_auswahl_pg_test.go.
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
		await expect(ladeKontoauszuege('', 'zzz', [])).rejects.toThrow(/niemanden/);
		expect(apiFetch).not.toHaveBeenCalled();
	});
});
