import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, waitFor } from '@testing-library/svelte';
import UserManagementZugangsanfragen from './UserManagementZugangsanfragen.svelte';
import { apiFetch } from './apiFetch.js';
import { authStore } from './stores/authStore.svelte.js';

// Die zweite Quelle der Warnung fragt den Server (Kandidatensuche des Zusammenführens).
// Ohne Mock antwortet niemand — die bestehenden Fälle unten haben keine leser_id und
// lösen deshalb gar keine Anfrage aus.
// importOriginal statt eines vollständigen Ersatzes: Seit die Komponente den authStore
// liest, hängt an diesem Modul auch dessen Sitzungs-Haken
// (registriereSitzungAbgelaufenHandler). Ein Mock, der nur apiFetch kennt, liesse ihn
// fehlen — die Datei bräche beim Laden, noch bevor ein Test läuft.
vi.mock('./apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn(async () => ({ ok: true, json: async () => [] })),
	extractApiError: vi.fn(async () => '')
}));

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
	vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: true, json: async () => [] }));
	// Ohne Recht kein Knopf — die Fälle, die nur den Text prüfen, laufen ohne Anmeldung.
	authStore.currentUser = null;
});

// Ein Antrag aus der Selbstanmeldung (aktiv=false + zugang_beantragt_am) muss ÜBER der
// Tabelle stehen — ein bewusst deaktiviertes Konto (ohne Antragszeit) nicht.
describe('UserManagementZugangsanfragen', () => {
	it('nennt offene Anträge mit Zahl und Namen, ignoriert deaktivierte Konten', () => {
		const screen = render(UserManagementZugangsanfragen, {
			users: [
				{
					vorname: 'Erika',
					nachname: 'Musterfrau',
					email: 'e@x',
					aktiv: false,
					zugang_beantragt_am: '2026-08-26T10:00:00Z'
				},
				{
					vorname: 'Alt',
					nachname: 'Konto',
					email: 'a@x',
					aktiv: false,
					zugang_beantragt_am: null
				},
				{
					vorname: 'Aktiv',
					nachname: 'Schon',
					email: 's@x',
					aktiv: true,
					zugang_beantragt_am: '2026-08-01T10:00:00Z'
				}
			]
		});
		const status = screen.getByRole('status');
		expect(status.textContent).toContain('1 Zugangsanfrage aus der Selbstanmeldung wartet');
		expect(status.textContent).toContain('Erika Musterfrau');
		expect(status.textContent).not.toContain('Alt Konto');
	});

	// Eine aus Littera übernommene Lehrkraft hat keine E-Mail, nur eine Platzhalter-Adresse
	// (internal/littera: littera-<id>@littera.invalid). Meldet sie sich selbst an, findet die
	// Anmeldung sie nicht und legt einen zweiten Eintrag an — Ausweis und Ausleihen am ersten,
	// Anmeldung am zweiten. Verbunden wird bewusst nicht automatisch (gleicher Name heißt nicht
	// gleiche Person); die Freischaltung muss den Treffer aber zeigen.
	it('zeigt beim Antrag den gleichnamigen Eintrag aus der Littera-Übernahme', () => {
		const screen = render(UserManagementZugangsanfragen, {
			users: [
				{
					vorname: 'Erika',
					nachname: 'Musterfrau',
					email: 'erika.musterfrau@schule.example',
					aktiv: false,
					zugang_beantragt_am: '2026-09-16T07:00:00Z'
				},
				{
					vorname: 'erika',
					nachname: 'Musterfrau ',
					email: 'littera-4908@littera.invalid',
					barcode_id: 'L-4908',
					leser_id: 'leser-littera',
					aktiv: true,
					zugang_beantragt_am: null
				}
			]
		});
		const text = screen.getByRole('status').textContent ?? '';
		expect(text).toContain('Ausweis L-4908');
		expect(text).toContain('steht schon aus der Littera-Übernahme im Bestand');
	});

	it('zeigt keinen Treffer für eine gleichnamige Person mit eigener Adresse', () => {
		const screen = render(UserManagementZugangsanfragen, {
			users: [
				{
					vorname: 'Erika',
					nachname: 'Musterfrau',
					email: 'erika.musterfrau@schule.example',
					aktiv: false,
					zugang_beantragt_am: '2026-09-16T07:00:00Z'
				},
				{
					vorname: 'Erika',
					nachname: 'Musterfrau',
					email: 'e.musterfrau@schule.example',
					barcode_id: 'L-77',
					aktiv: true,
					zugang_beantragt_am: null
				}
			]
		});
		expect(screen.getByRole('status').textContent).not.toContain('im Bestand');
	});

	// Der Altbestand: ein Kollege, der vor der Schul-E-Mail-Pflicht von Hand eingetragen
	// wurde. Er hat eine Leserzeile, aber KEIN Konto — in `users` steht er also nicht, und
	// die Warnung über die Littera-Platzhalter konnte ihn nicht sehen. Gefunden wird er
	// über die Kandidatensuche, von der Leserzeile des Antrags aus.
	it('zeigt den gleichnamigen Eintrag aus der Leserdatei, der noch kein Konto hat', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => [
					{ id: 'leser-alt', vorname: 'Erika', nachname: 'Musterfrau', barcode_id: 'A-4711' }
				]
			})
		);
		const screen = render(UserManagementZugangsanfragen, {
			users: [
				{
					vorname: 'Erika',
					nachname: 'Musterfrau',
					email: 'erika.musterfrau@schule.example',
					leser_id: 'leser-neu',
					aktiv: false,
					zugang_beantragt_am: '2026-09-16T07:00:00Z'
				}
			]
		});
		await waitFor(() => {
			const text = screen.getByRole('status').textContent ?? '';
			expect(text).toContain('steht schon in der Leserdatei, bisher ohne Zugang');
			expect(text).toContain('Ausweis A-4711');
			expect(text).toContain('Beide Einträge gehören zusammen');
		});
		expect(vi.mocked(apiFetch).mock.calls[0][0]).toContain(
			'/api/schueler/leser-neu/zusammenfuehren-kandidaten?q=Musterfrau'
		);
	});

	it('meldet einen anders heissenden Treffer nicht', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => [{ id: 'x', vorname: 'Erik', nachname: 'Musterfrau', barcode_id: 'A-9' }]
			})
		);
		const screen = render(UserManagementZugangsanfragen, {
			users: [
				{
					vorname: 'Erika',
					nachname: 'Musterfrau',
					email: 'erika.musterfrau@schule.example',
					leser_id: 'leser-neu',
					aktiv: false,
					zugang_beantragt_am: '2026-09-16T07:00:00Z'
				}
			]
		});
		await waitFor(() => expect(vi.mocked(apiFetch)).toHaveBeenCalled());
		expect(screen.getByRole('status').textContent).not.toContain('bisher ohne Zugang');
	});

	// Wer die Kandidatensuche nicht darf (403), soll die Freischaltungs-Zeile trotzdem
	// unverändert bekommen — das Recht merge_students hat mit dem Freischalten nichts zu tun.
	it('bleibt ohne das Recht zur Kandidatensuche brauchbar', async () => {
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: false, status: 403 }));
		const screen = render(UserManagementZugangsanfragen, {
			users: [
				{
					vorname: 'Erika',
					nachname: 'Musterfrau',
					email: 'erika.musterfrau@schule.example',
					leser_id: 'leser-neu',
					aktiv: false,
					zugang_beantragt_am: '2026-09-16T07:00:00Z'
				}
			]
		});
		await waitFor(() => expect(vi.mocked(apiFetch)).toHaveBeenCalled());
		const text = screen.getByRole('status').textContent ?? '';
		expect(text).toContain('1 Zugangsanfrage');
		expect(text).not.toContain('bisher ohne Zugang');
	});

	it('zeigt nichts, wenn kein Antrag offen ist', () => {
		const screen = render(UserManagementZugangsanfragen, {
			users: [
				{ vorname: 'Alt', nachname: 'Konto', email: 'a@x', aktiv: false, zugang_beantragt_am: null }
			]
		});
		expect(screen.queryByRole('status')).toBeNull();
	});
});

// „Das ist dieselbe Person" — der Knopf, der aus zwei Einträgen einen macht.
//
// Bis zum 16.09.2026 riet die Zeile, die Anfrage zu löschen und die Adresse am
// vorhandenen Eintrag nachzutragen. Das Löschen nimmt aber nur die KONTOZEILE mit; die
// Leserzeile, die der Wächter trg_benutzer_hat_leserzeile an jedes Konto ohne leser_id
// hängt, bleibt als Waise stehen — mit dem aus der Adresse geratenen Namen, ohne Ausweis,
// ohne Konto. Also genau der Doppeleintrag, vor dem die Zeile warnt.
describe('Zugangsanfrage zuordnen', () => {
	const anfrage = {
		id: 'konto-neu',
		vorname: 'Erika',
		nachname: 'Musterfrau',
		email: 'erika.musterfrau@schule.example',
		leser_id: 'leser-neu',
		aktiv: false,
		zugang_beantragt_am: '2026-09-16T07:00:00Z'
	};

	/** @param {any[]} kandidaten */
	const mitKandidaten = (kandidaten) =>
		vi.mocked(apiFetch).mockImplementation(
			/** @type {any} */ (
				async (/** @type {string} */ url) => {
					if (url.includes('zusammenfuehren-kandidaten')) {
						return { ok: true, json: async () => kandidaten };
					}
					return { ok: true, json: async () => ({ ziel_id: 'leser-alt' }) };
				}
			)
		);

	it('ordnet in der richtigen Richtung zu: der vorhandene Eintrag bleibt', async () => {
		authStore.currentUser = { rolle: 'mitarbeiter', permissions: ['merge_students'] };
		mitKandidaten([
			{ id: 'leser-alt', vorname: 'Erika', nachname: 'Musterfrau', barcode_id: 'A-4711' }
		]);
		let neuGeladen = 0;
		const screen = render(UserManagementZugangsanfragen, {
			users: [anfrage],
			onZugeordnet: () => {
				neuGeladen += 1;
			}
		});

		const knopf = await screen.findByRole('button', { name: 'Das ist dieselbe Person' });
		knopf.click();

		await waitFor(() => expect(neuGeladen).toBe(1));
		const aufruf = vi
			.mocked(apiFetch)
			.mock.calls.find(
				(c) => String(c[0]).includes('/zusammenfuehren') && !String(c[0]).includes('-kandidaten')
			);
		if (!aufruf) throw new Error('das Zuordnen hat den Server nie gerufen');
		// Die RICHTUNG ist der Kern: Ziel (in der Adresse) ist der vorhandene Eintrag, er
		// behält Ausweis und Bücher. Quelle ist die Zeile des Wächters, sie geht darin auf.
		// Verkehrt herum verschwände der vorhandene Eintrag, und der aus der Adresse
		// geratene Name bliebe übrig — still und nur über das Protokoll zu reparieren.
		expect(aufruf[0]).toBe('/api/schueler/leser-alt/zusammenfuehren');
		expect(JSON.parse(String(/** @type {any} */ (aufruf[1]).body))).toEqual({
			quelle_id: 'leser-neu'
		});
	});

	// Der Knopf hängt am Recht des Servers (merge_students, api/routes_students.go). Ohne
	// das Recht bliebe er stehen und lieferte 403 — die WARNUNG muss trotzdem bleiben,
	// sonst verschwindet die Dublette aus dem Blick, statt aufgelöst zu werden.
	it('zeigt ohne das Recht keinen Knopf, aber den Hinweis', async () => {
		authStore.currentUser = { rolle: 'mitarbeiter', permissions: ['manage_users'] };
		mitKandidaten([
			{ id: 'leser-alt', vorname: 'Erika', nachname: 'Musterfrau', barcode_id: 'A-4711' }
		]);
		const screen = render(UserManagementZugangsanfragen, { users: [anfrage] });

		await waitFor(() =>
			expect(screen.getByRole('status').textContent).toContain('bisher ohne Zugang')
		);
		expect(screen.queryByRole('button', { name: 'Das ist dieselbe Person' })).toBeNull();
	});
});
