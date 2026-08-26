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

	it('zeigt nichts, wenn kein Antrag offen ist', () => {
		const screen = render(UserManagementZugangsanfragen, {
			users: [
				{ vorname: 'Alt', nachname: 'Konto', email: 'a@x', aktiv: false, zugang_beantragt_am: null }
			]
		});
		expect(screen.queryByRole('status')).toBeNull();
	});
});
