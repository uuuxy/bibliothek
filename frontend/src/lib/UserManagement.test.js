import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor, within } from '@testing-library/svelte';
import UserManagement from './UserManagement.svelte';
import { apiFetch } from './apiFetch.js';
import { authStore } from './stores/authStore.svelte.js';
import { toastStore } from './stores/toastStore.svelte.js';

// importOriginal statt eines vollständigen Ersatzes: Am Modul hängt auch der Sitzungs-Haken
// des authStore (siehe UserManagementZugangsanfragen.test.js).
vi.mock('./apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn(),
	extractApiError: vi.fn(async () => 'Der Server meldet einen Fehler.')
}));

const KONTO = {
	id: 'u1',
	vorname: 'Kora',
	nachname: 'Muster',
	email: 'kora@test.local',
	barcode_id: 'A-11266',
	rolle: 'kollegium',
	aktiv: true,
	zugang_beantragt_am: null
};

const antwort = (/** @type {any} */ daten, ok = true, status = 200) =>
	/** @type {any} */ ({ ok, status, json: async () => daten });

/** Die Liste antwortet mit einem Konto; jede andere Anfrage mit `sonst`. */
function server(/** @type {any} */ sonst = antwort({})) {
	vi.mocked(apiFetch).mockImplementation(async (adresse, optionen) =>
		adresse === '/api/benutzer' && !optionen ? antwort([KONTO]) : sonst
	);
}

async function benutzerliste() {
	const screen = render(UserManagement);
	await screen.findByRole('table', { name: 'Benutzerkonten' });
	return screen;
}

const meldungen = () => toastStore.toasts.map((t) => [t.message, t.type]);

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
	server();
	authStore.currentUser = null;
	for (const t of [...toastStore.toasts]) toastStore.removeToast(t.id);
});

describe('Benutzerliste: Meldungen', () => {
	// Die Seite trug eine eigene Erfolgsmeldung unten rechts. Die Anwendung hat eine Stelle
	// für Meldungen; zwei Stellen sehen verschieden aus und stehen verschieden lang.
	it('meldet das Anlegen über die Meldungen der Anwendung', async () => {
		const screen = await benutzerliste();
		await fireEvent.click(screen.getByRole('button', { name: /Benutzer anlegen/ }));

		const dialog = screen.getByRole('dialog');
		for (const [feld, wert] of [
			['Vorname', 'Nele'],
			['Nachname', 'Neu'],
			['E-Mail Adresse', 'nele@test.local']
		])
			await fireEvent.input(within(dialog).getByLabelText(feld), { target: { value: wert } });
		await fireEvent.submit(/** @type {HTMLFormElement} */ (dialog.querySelector('form')));

		await waitFor(() =>
			expect(meldungen()).toEqual([['Benutzer erfolgreich angelegt.', 'success']])
		);
		expect(screen.queryByRole('dialog')).toBeNull();
	});

	it('meldet das Löschen über die Meldungen der Anwendung', async () => {
		const screen = await benutzerliste();
		await fireEvent.click(screen.getByRole('button', { name: 'Löschen' }));
		await fireEvent.click(
			within(screen.getByRole('dialog')).getByRole('button', { name: 'Löschen' })
		);

		await waitFor(() =>
			expect(meldungen()).toEqual([['Benutzer erfolgreich gelöscht.', 'success']])
		);
		expect(screen.queryByRole('dialog')).toBeNull();
	});

	// Ein Konto mit offenen Ausleihen lässt sich nicht löschen. Der Satz des Servers sagt, was
	// zu tun ist, und steht deshalb im Dialog, an dem die Frage gestellt wurde.
	it('lässt den Dialog offen, wenn das Löschen scheitert, und nennt dort den Grund', async () => {
		server(antwort({}, false, 409));
		const screen = await benutzerliste();
		await fireEvent.click(screen.getByRole('button', { name: 'Löschen' }));
		const dialog = screen.getByRole('dialog');
		await fireEvent.click(within(dialog).getByRole('button', { name: 'Löschen' }));

		const grund = await within(dialog).findByRole('alert');
		expect(grund.textContent).toContain('Der Server meldet einen Fehler.');
		expect(meldungen()).toEqual([]);
	});

	// M3, Dialogs: „Buttons are aligned to the trailing edge of the dialog … The confirmation
	// button is always closest to the edge." So steht es auch in der Rückfrage des Hauses.
	it('stellt die Knöpfe der Lösch-Rückfrage an den rechten Rand, Löschen außen', async () => {
		const screen = await benutzerliste();
		await fireEvent.click(screen.getByRole('button', { name: 'Löschen' }));
		const dialog = screen.getByRole('dialog');

		const knoepfe = within(dialog).getAllByRole('button');
		expect(knoepfe.map((k) => k.textContent?.trim())).toEqual(['Abbrechen', 'Löschen']);
		const zeile = /** @type {HTMLElement} */ (knoepfe[0].parentElement);
		expect(knoepfe[1].parentElement).toBe(zeile);
		expect(zeile.className.split(/\s+/)).toContain('justify-end');
		expect(zeile.className).not.toMatch(/justify-center/);
	});

	it('zeigt einen Ladefehler als Meldung, die sich über einen benannten Knopf schließen lässt', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort({}, false, 500));
		const screen = render(UserManagement);

		const meldung = await screen.findByRole('alert');
		expect(meldung.textContent).toContain('Der Server meldet einen Fehler.');

		await fireEvent.click(screen.getByRole('button', { name: 'Meldung schließen' }));
		expect(screen.queryByRole('alert')).toBeNull();
	});
});

describe('Benutzerliste: Namen für Vorleseprogramme', () => {
	it('nennt den Knopf zum Anlegen ohne Sinnbild', async () => {
		const screen = await benutzerliste();
		expect(screen.getByRole('button', { name: 'Benutzer anlegen' })).toBeTruthy();
	});

	it('gibt jedem der drei Dialoge seine Überschrift als Namen', async () => {
		const screen = await benutzerliste();

		await fireEvent.click(screen.getByRole('button', { name: /Benutzer anlegen/ }));
		const anlegen = screen.getByRole('dialog', { name: 'Neuen Benutzer anlegen' });
		await fireEvent.click(within(anlegen).getByRole('button', { name: 'Abbrechen' }));

		await fireEvent.click(screen.getByRole('button', { name: 'Bearbeiten' }));
		const bearbeiten = screen.getByRole('dialog', { name: 'Benutzer bearbeiten' });
		await fireEvent.click(within(bearbeiten).getByRole('button', { name: 'Abbrechen' }));

		await fireEvent.click(screen.getByRole('button', { name: 'Löschen' }));
		expect(screen.getByRole('dialog', { name: 'Benutzer unwiderruflich löschen?' })).toBeTruthy();
	});
});
