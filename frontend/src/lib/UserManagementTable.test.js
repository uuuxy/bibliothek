import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, within } from '@testing-library/svelte';
import UserManagementTable from './UserManagementTable.svelte';

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

/** @param {any[]} filteredUsers @param {Record<string, any>} [rest] */
const tabelle = (filteredUsers, rest = {}) =>
	render(UserManagementTable, {
		loadingUsers: false,
		filteredUsers,
		openEditUserModal: vi.fn(),
		openDeleteConfirm: vi.fn(),
		...rest
	});

/** Die Zellen der Zeile, in der `name` steht. */
const zellen = (/** @type {ReturnType<typeof tabelle>} */ screen, /** @type {string} */ name) =>
	within(/** @type {HTMLElement} */ (screen.getByText(name).closest('tr')))
		.getAllByRole('cell')
		.map((z) => z.textContent?.trim());

describe('UserManagementTable', () => {
	it('nennt je Konto Name, E-Mail, Ausweisnummer, Rolle und Zustand', () => {
		const screen = tabelle([
			konto(),
			konto({ id: 'u2', vorname: 'Ole', nachname: 'Ohne', barcode_id: '', aktiv: false }),
			konto({
				id: 'u3',
				vorname: 'Anna',
				nachname: 'Antrag',
				aktiv: false,
				zugang_beantragt_am: '2026-08-26T10:00:00Z'
			})
		]);

		expect(zellen(screen, 'Kora Muster').slice(0, 5)).toEqual([
			'Kora Muster',
			'kora@test.local',
			'A-11266',
			'kollegium',
			'Aktiv'
		]);
		expect(zellen(screen, 'Ole Ohne').slice(2, 5)).toEqual(['Keine', 'kollegium', 'Inaktiv']);
		expect(zellen(screen, 'Anna Antrag')[4]).toBe('Zugang beantragt');
	});

	it('reicht das Konto der Zeile an Bearbeiten und Löschen', async () => {
		const openEditUserModal = vi.fn();
		const openDeleteConfirm = vi.fn();
		const eins = konto();
		const screen = tabelle([eins], { openEditUserModal, openDeleteConfirm });

		await fireEvent.click(screen.getByRole('button', { name: 'Bearbeiten' }));
		await fireEvent.click(screen.getByRole('button', { name: 'Löschen' }));

		expect(openEditUserModal).toHaveBeenCalledWith(eins);
		expect(openDeleteConfirm).toHaveBeenCalledWith(eins);
	});

	it('sagt ohne Sinnbild, dass niemand gefunden wurde', () => {
		const screen = tabelle([]);
		expect(screen.container.textContent?.trim()).toBe('Keine Systembenutzer gefunden');
	});
});
