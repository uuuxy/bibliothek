import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import { authStore } from '../../stores/authStore.svelte.js';
import Anmeldung from './Anmeldung.svelte';

/** @param {HTMLElement} feld */
const beschreibung = (feld) =>
	document.getElementById(feld.getAttribute('aria-describedby') ?? '')?.textContent;

/** Ein echter Klick legt den Fokus auf den Knopf; fireEvent allein tut das nicht. @param {HTMLElement} knopf */
async function klicke(knopf) {
	knopf.focus();
	await fireEvent.click(knopf);
}

// „Anmelden" bleibt bei leerem Feld bedienbar: Der Klick nennt am Feld, was fehlt, führt dorthin
// und schickt nichts an den Server.
describe('Anmeldung: leeres Feld', () => {
	const fetchVorher = globalThis.fetch;

	beforeEach(() => {
		authStore.loginEmail = '';
		authStore.loginPassword = '';
		globalThis.fetch = vi.fn();
	});

	afterEach(() => {
		globalThis.fetch = fetchVorher;
	});

	it('nennt erst die fehlende E-Mail, dann das fehlende Passwort', async () => {
		const k = render(Anmeldung);
		const email = k.getByLabelText('E-Mail');
		const passwort = k.getByLabelText('Passwort');
		const anmelden = k.getByRole('button', { name: 'Anmelden' });
		// Die Seite legt den Fokus nach dem Aufbau selbst ins E-Mail-Feld; erst danach klicken.
		await waitFor(() => expect(document.activeElement).toBe(email));

		await klicke(anmelden);
		expect(email.getAttribute('aria-invalid')).toBe('true');
		expect(beschreibung(email)).toBe('Bitte deine Schul-E-Mail eintragen.');
		expect(document.activeElement).toBe(email);

		await fireEvent.input(email, { target: { value: 'lehrkraft@example.org' } });
		await klicke(anmelden);
		expect(email.getAttribute('aria-invalid')).toBeNull();
		expect(passwort.getAttribute('aria-invalid')).toBe('true');
		expect(beschreibung(passwort)).toBe('Bitte dein Passwort eintragen.');
		expect(document.activeElement).toBe(passwort);
		// Hinaus geht nur die Frage nach der Domain der Schule, keine Anmeldung.
		expect(vi.mocked(globalThis.fetch).mock.calls.map(([adresse]) => adresse)).toEqual([
			'/api/auth/domain'
		]);
	});
});

// Mit der Domain der Schule genügt der Name: Das Feld nimmt ihn an und sagt darunter, was der
// Server ergänzt. Der Fehler bei leerem Feld ersetzt diesen Hinweis.
describe('Anmeldung: Domain der Schule', () => {
	const fetchVorher = globalThis.fetch;

	/** @param {unknown} domain */
	function serverNennt(domain) {
		globalThis.fetch = vi.fn(async () => new Response(JSON.stringify(domain), { status: 200 }));
	}

	beforeEach(() => {
		authStore.loginEmail = '';
		authStore.loginPassword = '';
	});

	afterEach(() => {
		globalThis.fetch = fetchVorher;
	});

	it('nimmt den Namen an und nennt die Domain unter dem Feld', async () => {
		serverNennt({ domain: 'schule.example', ldap: false });
		const k = render(Anmeldung);
		const feld = await k.findByLabelText('Name oder E-Mail');
		expect(feld.getAttribute('type')).toBe('text');
		expect(feld.getAttribute('autocomplete')).toBe('username');
		expect(beschreibung(feld)).toBe(
			'Der Name vor dem @ genügt – @schule.example ergänzt das Programm.'
		);

		await klicke(k.getByRole('button', { name: 'Anmelden' }));
		expect(feld.getAttribute('aria-invalid')).toBe('true');
		expect(beschreibung(feld)).toBe('Bitte deinen Namen oder deine Schul-E-Mail eintragen.');
	});

	it('bleibt ohne Domain bei der vollen Adresse', async () => {
		serverNennt({ domain: '', ldap: false });
		const k = render(Anmeldung);
		await waitFor(() => expect(globalThis.fetch).toHaveBeenCalledWith('/api/auth/domain'));
		const feld = k.getByLabelText('E-Mail');
		expect(feld.getAttribute('type')).toBe('email');
		expect(beschreibung(feld) ?? '').toBe('');
	});

	// Mit LDAP tippt manche Person ihren Benutzernamen: Ein Feld vom Typ email ließe den Browser
	// ihn abweisen, bevor er den Server erreicht.
	it('nimmt mit LDAP auch einen Benutzernamen an', async () => {
		serverNennt({ domain: '', ldap: true });
		const k = render(Anmeldung);
		const feld = await k.findByLabelText('E-Mail oder Benutzername');
		expect(feld.getAttribute('type')).toBe('text');
		expect(feld.getAttribute('autocomplete')).toBe('username');
		expect(k.getByText('Mit den Zugangsdaten deiner Schule.')).toBeTruthy();

		await klicke(k.getByRole('button', { name: 'Anmelden' }));
		expect(beschreibung(feld)).toBe(
			'Bitte deine Schul-E-Mail oder deinen Benutzernamen eintragen.'
		);
	});
});

// Der Grund einer gescheiterten Anmeldung erscheint nach dem Klick, ohne dass der Fokus zu ihm
// wandert: Ohne Rolle liest ein Bildschirmleser nichts vor.
describe('Anmeldung: der Grund einer Ablehnung', () => {
	afterEach(() => {
		authStore.loginError = null;
	});

	it('wird als Meldung vorgelesen', async () => {
		const k = render(Anmeldung);
		authStore.loginError = 'E-Mail oder Passwort falsch.';
		expect((await k.findByRole('alert')).textContent).toContain('E-Mail oder Passwort falsch.');
	});
});

// Der Fokus ins E-Mail-Feld kommt 50 ms nach dem Aufbau. Ein Zeitgeber, der den Abbau überlebt,
// griff in der CI nach dem Ende der Testdatei auf `document` zu, das es dann nicht mehr gab.
describe('Anmeldung: Abbau', () => {
	const fetchVorher = globalThis.fetch;

	beforeEach(() => {
		globalThis.fetch = vi.fn(() => new Promise(() => {}));
		vi.useFakeTimers();
	});

	afterEach(() => {
		vi.useRealTimers();
		globalThis.fetch = fetchVorher;
	});

	it('räumt den Zeitgeber für den Fokus ab', () => {
		const k = render(Anmeldung);
		const suche = vi.spyOn(document, 'getElementById');
		k.unmount();
		vi.advanceTimersByTime(100);
		expect(suche).not.toHaveBeenCalledWith('login-email');
		suche.mockRestore();
	});
});
