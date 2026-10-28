import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor, within } from '@testing-library/svelte';
import Konten from './Konten.svelte';
import { apiFetch } from '../../apiFetch.js';
import { authStore } from '../../stores/authStore.svelte.js';
import { snackbars } from '../../stores/snackbars.svelte.js';

// importOriginal statt eines vollständigen Ersatzes: Am Modul hängt auch der Sitzungs-Haken
// des authStore (siehe Zugangsanfragen.test.js).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn(),
	leseApiFehler: vi.fn(async () => 'Der Server meldet einen Fehler.')
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
	const screen = render(Konten);
	await screen.findByRole('table', { name: 'Benutzerkonten' });
	return screen;
}

const meldungen = () => snackbars.toasts.map((t) => [t.message, t.type]);

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
	server();
	authStore.currentUser = null;
	for (const t of [...snackbars.toasts]) snackbars.removeToast(t.id);
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

	// M3, Dialogs: „Dismissive actions are never disabled." Die Anfrage lässt sich nicht
	// zurückholen; „Abbrechen" schließt den Dialog wie Escape, das Ergebnis steht auf der Seite.
	it('lässt „Abbrechen" bedienbar, solange gelöscht wird, und meldet das Ergebnis auf der Seite', async () => {
		/** @type {(antwort: any) => void} */
		let beantworte = () => {};
		vi.mocked(apiFetch).mockImplementation(async (adresse, optionen) =>
			adresse === '/api/benutzer' && !optionen
				? antwort([KONTO])
				: new Promise((fertig) => (beantworte = fertig))
		);
		const screen = await benutzerliste();
		await fireEvent.click(screen.getByRole('button', { name: 'Löschen' }));
		const dialog = screen.getByRole('dialog');
		await fireEvent.click(within(dialog).getByRole('button', { name: 'Löschen' }));

		const knopf = (/** @type {string} */ name) =>
			/** @type {HTMLButtonElement} */ (within(dialog).getByRole('button', { name }));
		expect(knopf('Löschen').disabled).toBe(true);
		expect(knopf('Abbrechen').disabled).toBe(false);
		await fireEvent.click(knopf('Abbrechen'));
		expect(screen.queryByRole('dialog')).toBeNull();

		beantworte(antwort({}, false, 409));
		const meldung = await screen.findByRole('alert');
		expect(meldung.textContent).toContain('Der Server meldet einen Fehler.');
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

	it('lässt den Grund eines gescheiterten Speicherns nach dem Abbrechen über einen benannten Knopf schließen', async () => {
		server(antwort({}, false, 500));
		const screen = await benutzerliste();
		await fireEvent.click(screen.getByRole('button', { name: /Benutzer anlegen/ }));
		const dialog = screen.getByRole('dialog');
		await fireEvent.submit(/** @type {HTMLFormElement} */ (dialog.querySelector('form')));
		await within(dialog).findByText('Der Server meldet einen Fehler.');
		await fireEvent.click(within(dialog).getByRole('button', { name: 'Abbrechen' }));

		const meldung = await screen.findByRole('alert');
		expect(meldung.textContent).toContain('Der Server meldet einen Fehler.');

		await fireEvent.click(screen.getByRole('button', { name: 'Meldung schließen' }));
		expect(screen.queryByRole('alert')).toBeNull();
	});

	it('filtert die Liste nach dem Suchtext und sagt, wenn nichts passt', async () => {
		const screen = await benutzerliste();
		const suche = screen.getByRole('searchbox', { name: 'Benutzer suchen' });

		await fireEvent.input(suche, { target: { value: 'A-112' } });
		expect(screen.getByText('Kora Muster')).toBeTruthy();

		await fireEvent.input(suche, { target: { value: 'niemand' } });
		expect(screen.queryByText('Kora Muster')).toBeNull();
		expect(screen.getByText('Keine Systembenutzer gefunden')).toBeTruthy();
	});

	// Wie überall: Scheitert nur das Neuladen, ist die Liste von vorhin noch brauchbar. Der Grund
	// steht dann oben im Band, nicht als Kasten an ihrer Stelle.
	it('lässt die Liste stehen, wenn nur das Neuladen scheitert, und nennt den Grund oben', async () => {
		const screen = await benutzerliste();
		vi.mocked(apiFetch).mockImplementation(async (adresse, optionen) =>
			adresse === '/api/benutzer' && !optionen ? antwort({}, false, 500) : antwort({})
		);
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
			expect(screen.getByRole('alert').textContent).toContain('Der Server meldet einen Fehler.')
		);
		expect(screen.getByText('Kora Muster')).toBeTruthy();
		expect(screen.queryByText('Benutzerkonten nicht geladen')).toBeNull();
	});

	// Darunter stand „Keine Systembenutzer gefunden": Es gäbe keine Konten, während nur der
	// Abruf scheiterte.
	it('nennt den Grund statt der leeren Liste, wenn die Konten nicht kommen, und lädt neu', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort({}, false, 500));
		const screen = render(Konten);

		await screen.findByText('Benutzerkonten nicht geladen');
		expect(screen.getAllByText('Der Server meldet einen Fehler.')).toHaveLength(1);
		expect(screen.queryByText('Keine Systembenutzer gefunden')).toBeNull();

		server();
		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));
		await screen.findByRole('table', { name: 'Benutzerkonten' });
		expect(screen.queryByText('Benutzerkonten nicht geladen')).toBeNull();
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

// Ging die Antwort verloren, hatte der Server schon gewirkt: Die Liste zeigte das gelöschte Konto
// weiter und das angelegte nicht, und der zweite Versuch meldete „Konto nicht gefunden" oder „gibt
// es bereits". In Chromium, Firefox und WebKit (OFFEN.md 5.106).
describe('Benutzerliste: nach einem Fehler neu laden', () => {
	const NEU = { ...KONTO, id: 'u2', vorname: 'Nele', nachname: 'Neu', email: 'nele@test.local' };

	/** Die Liste antwortet mit `listen[n]` beim n-ten Abruf; jeder Auftrag scheitert ohne Antwort. */
	function ohneAntwort(/** @type {any[][]} */ ...listen) {
		const abrufe = { n: 0 };
		vi.mocked(apiFetch).mockImplementation(async (adresse, optionen) => {
			if (adresse === '/api/benutzer' && !optionen) return antwort(listen[abrufe.n++] ?? []);
			throw new Error('Keine Verbindung zum Server.');
		});
		return abrufe;
	}

	it('lädt die Liste nach einem gescheiterten Anlegen neu und lässt den Grund im Dialog', async () => {
		const abrufe = ohneAntwort([KONTO], [KONTO, NEU]);
		const screen = await benutzerliste();
		await fireEvent.click(screen.getByRole('button', { name: /Benutzer anlegen/ }));
		const dialog = screen.getByRole('dialog');
		for (const [feld, wert] of [
			['Vorname', NEU.vorname],
			['Nachname', NEU.nachname],
			['E-Mail Adresse', NEU.email]
		])
			await fireEvent.input(within(dialog).getByLabelText(feld), { target: { value: wert } });
		await fireEvent.submit(/** @type {HTMLFormElement} */ (dialog.querySelector('form')));

		await waitFor(() => expect(abrufe.n).toBe(2));
		expect(await screen.findByText(NEU.email)).toBeTruthy();
		expect(within(dialog).getByRole('alert').textContent).toContain('Keine Verbindung zum Server.');
	});

	it('lädt die Liste nach einem gescheiterten Löschen neu und lässt den Grund im Dialog', async () => {
		const abrufe = ohneAntwort([KONTO], []);
		const screen = await benutzerliste();
		await fireEvent.click(screen.getByRole('button', { name: 'Löschen' }));
		const dialog = screen.getByRole('dialog');
		await fireEvent.click(within(dialog).getByRole('button', { name: 'Löschen' }));

		await waitFor(() => expect(abrufe.n).toBe(2));
		await waitFor(() => expect(screen.queryByText(KONTO.email)).toBeNull());
		expect(within(dialog).getByRole('alert').textContent).toContain('Keine Verbindung zum Server.');
	});
});
