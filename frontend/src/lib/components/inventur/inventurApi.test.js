import { describe, it, expect, vi } from 'vitest';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import { serverAntwort } from '../../serverAntwort.js';
import {
	ladeOffeneSessions,
	ladeAbgeschlosseneInventuren,
	starteLauf,
	schliesseAb,
	brichAb,
	meldeVerlustGefunden,
	loescheVerlustEndgueltig,
	ladeFehlbestand,
	loescheAbgeschlosseneInventur
} from './inventurApi.js';

// Eine leere Liste hieße „keine Inventur läuft" und „keine frühere"; lehnt der Server ab,
// werfen die Abrufe deshalb mit seinem Grund.
describe('inventurApi: Listen der Seite', () => {
	it.each([
		['laufende', ladeOffeneSessions],
		['frühere', ladeAbgeschlosseneInventuren]
	])('wirft bei abgelehntem Abruf der %s mit dem Grund des Servers', async (_name, lade) => {
		vi.mocked(apiFetch).mockResolvedValue(
			serverAntwort(503, { error: 'Die Datenbank antwortet nicht' })
		);
		await expect(lade()).rejects.toThrow('Die Datenbank antwortet nicht');
	});
});

// Ohne Antwort des Servers (kein Netz, Frist abgelaufen) warfen die Anfragen, und keiner der
// Aufrufer fing das: Start, Abschluss, Verwerfen, „Gefunden" und „endgültig löschen" taten
// sichtbar nichts. Gleich in Chromium, Firefox und WebKit (OFFEN.md 5.106).
describe('inventurApi: ohne Antwort des Servers', () => {
	it.each([
		['Start', () => starteLauf({ type: 'global' })],
		['Abschluss', () => schliesseAb('s1')],
		['Verwerfen', () => brichAb('s1')],
		['Gefunden', () => meldeVerlustGefunden('e1')],
		['endgültig Löschen', () => loescheVerlustEndgueltig(['e1'])],
		['Fehlbestand', () => ladeFehlbestand('s1')],
		['Löschen einer früheren Inventur', () => loescheAbgeschlosseneInventur('s1')]
	])('liefert beim %s den Fehler als Ergebnis, statt zu werfen', async (_name, anfrage) => {
		vi.mocked(apiFetch).mockRejectedValue(new Error('Keine Verbindung zum Server.'));
		await expect(anfrage()).resolves.toEqual({
			ok: false,
			status: 0,
			error: 'Keine Verbindung zum Server.',
			data: {}
		});
	});
});

// Die nächste Seite schließt an die vorige an: Ihr Zeichen geht unverändert an den Server zurück.
describe('inventurApi: Seiten der früheren Inventuren', () => {
	it('holt die erste Seite ohne Zeichen und die nächste mit dem Zeichen der vorigen', async () => {
		const seite = { laeufe: [{ session_id: 's1' }], weitere: 3, naechste_seite: 'z+1' };
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => seite })
		);

		expect(await ladeAbgeschlosseneInventuren()).toEqual(seite);
		await ladeAbgeschlosseneInventuren('2026-10-29T20:10:00.5Z_s1');

		expect(vi.mocked(apiFetch).mock.calls.map(([adresse]) => adresse)).toEqual([
			'/api/inventur/abgeschlossen',
			'/api/inventur/abgeschlossen?seite=2026-10-29T20%3A10%3A00.5Z_s1'
		]);
	});
});
