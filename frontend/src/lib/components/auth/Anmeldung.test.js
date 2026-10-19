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
		expect(globalThis.fetch).not.toHaveBeenCalled();
	});
});
