import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('./apiFetch.js', () => ({ apiFetch: vi.fn() }));
import { apiFetch } from './apiFetch.js';
import { sendeKontoauszuege } from './abgaengerDienst.js';

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
