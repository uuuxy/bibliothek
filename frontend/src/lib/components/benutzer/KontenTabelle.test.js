import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, within } from '@testing-library/svelte';
import KontenTabelle from './KontenTabelle.svelte';
import { authStore } from '../../stores/authStore.svelte.js';

const konto = (/** @type {Record<string, any>} */ felder = {}) => ({
	id: 'u1',
	vorname: 'Kora',
	nachname: 'Muster',
	email: 'kora@test.local',
	barcode_id: 'A-11266',
	rolle: 'kollegium',
	aktiv: true,
	zugang_beantragt_am: null,
	...felder
});

/** @param {any[]} konten @param {Record<string, any>} [rest] */
const tabelle = (konten, rest = {}) =>
	render(KontenTabelle, {
		laedt: false,
		konten,
		oeffneKonto: vi.fn(),
		frageLoeschen: vi.fn(),
		...rest
	});

/** Die Zellen der Zeile, in der `name` steht. */
const zellen = (/** @type {ReturnType<typeof tabelle>} */ screen, /** @type {string} */ name) =>
	within(/** @type {HTMLElement} */ (screen.getByText(name).closest('tr')))
		.getAllByRole('cell')
		.map((z) => z.textContent?.trim());

describe('KontenTabelle', () => {
	it('nennt je Konto Name, E-Mail, Anmeldung, Ausweisnummer, Rolle und Zustand', () => {
		const screen = tabelle([
			konto({ anmeldeverfahren: 'imap', ldap_kennung: '' }),
			konto({
				id: 'u2',
				vorname: 'Ole',
				nachname: 'Ohne',
				barcode_id: '',
				aktiv: false,
				anmeldeverfahren: 'ldap',
				ldap_kennung: 'OOH'
			}),
			konto({
				id: 'u3',
				vorname: 'Anna',
				nachname: 'Antrag',
				aktiv: false,
				zugang_beantragt_am: '2026-08-26T10:00:00Z'
			})
		]);

		expect(zellen(screen, 'Kora Muster').slice(0, 6)).toEqual([
			'Kora Muster',
			'kora@test.local',
			'IMAP',
			'A-11266',
			'Kollegium',
			'Aktiv'
		]);
		expect(zellen(screen, 'Ole Ohne').slice(2, 6)).toEqual([
			'LDAP',
			'Keine',
			'Kollegium',
			'Inaktiv'
		]);
		// Ein Konto aus einem Stand ohne Verfahren ist IMAP, die Vorgabe der Datenbank.
		expect(zellen(screen, 'Anna Antrag').slice(2, 6)).toEqual([
			'IMAP',
			'A-11266',
			'Kollegium',
			'Zugang beantragt'
		]);
	});

	// Grün und Gelb heißen im Programm „in Ordnung" und „Achtung". Die Rolle steht deshalb in
	// einem Abzeichen für alle, mit dem Wort aus dem Bearbeiten-Dialog statt dem Schlüssel.
	it('zeigt jede Rolle mit ihrem Namen im selben Abzeichen', () => {
		const rollen = ['kollegium', 'helfer', 'mitarbeiter', 'leitung', 'admin', 'unbekannt'];
		const screen = tabelle(
			rollen.map((rolle, i) => konto({ id: `u${i}`, vorname: 'Konto', nachname: rolle, rolle }))
		);
		const abzeichen = rollen.map((rolle) => {
			const zeile = /** @type {HTMLElement} */ (screen.getByText(`Konto ${rolle}`).closest('tr'));
			return /** @type {HTMLElement} */ (zeile.querySelector('[data-chip]'));
		});

		expect(abzeichen.map((a) => a.textContent?.trim())).toEqual([
			'Kollegium',
			'Helfer',
			'Mitarbeiter',
			'Leitung',
			'Administrator',
			'unbekannt'
		]);
		expect(new Set(abzeichen.map((a) => a.className)).size, 'eine Form für alle Rollen').toBe(1);
	});

	it('reicht das Konto der Zeile an Bearbeiten und Löschen', async () => {
		const oeffneKonto = vi.fn();
		const frageLoeschen = vi.fn();
		const eins = konto();
		const screen = tabelle([eins], { oeffneKonto, frageLoeschen });

		await fireEvent.click(screen.getByRole('button', { name: 'Kora Muster bearbeiten' }));
		await fireEvent.click(screen.getByRole('button', { name: 'Kora Muster löschen' }));

		expect(oeffneKonto).toHaveBeenCalledWith(eins);
		expect(frageLoeschen).toHaveBeenCalledWith(eins);
	});

	it('sagt ohne Sinnbild, dass niemand gefunden wurde', () => {
		const screen = tabelle([]);
		expect(screen.container.textContent?.trim()).toBe('Keine Systembenutzer gefunden');
	});

	// Das eigene Konto löscht niemand: Der Server lehnt es ab, der Knopf ist deshalb schon in der
	// Liste abgeschaltet und sagt, warum.
	it('schaltet das Löschen des eigenen Kontos ab', () => {
		authStore.currentUser = { user_id: 'u1' };
		try {
			const screen = tabelle([konto(), konto({ id: 'u2', vorname: 'Ole', nachname: 'Ohne' })]);
			const eigen = screen.getByRole('button', { name: 'Kora Muster löschen' });
			const fremd = screen.getByRole('button', { name: 'Ole Ohne löschen' });
			expect(eigen.hasAttribute('disabled')).toBe(true);
			expect(eigen.getAttribute('data-tip')).toBe('Das eigene Konto lässt sich nicht löschen');
			expect(fremd.hasAttribute('disabled')).toBe(false);
		} finally {
			authStore.currentUser = null;
		}
	});
});
