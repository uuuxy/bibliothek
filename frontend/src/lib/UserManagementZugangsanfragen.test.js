import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import UserManagementZugangsanfragen from './UserManagementZugangsanfragen.svelte';

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
					aktiv: true,
					zugang_beantragt_am: null
				}
			]
		});
		const text = screen.getByRole('status').textContent ?? '';
		expect(text).toContain('Ausweis L-4908');
		expect(text).toContain('Anfrage löschen');
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
		expect(screen.getByRole('status').textContent).not.toContain('Anfrage löschen');
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
