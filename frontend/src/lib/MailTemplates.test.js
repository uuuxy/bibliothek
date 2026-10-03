import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import MailTemplates from './MailTemplates.svelte';
import { apiClient } from './apiFetch.js';
import { toastStore } from './stores/toastStore.svelte.js';

// apiClient ruft apiFetch innerhalb desselben Moduls; ersetzt wird deshalb apiClient selbst.
vi.mock('./apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiClient: { get: vi.fn(), put: vi.fn() }
}));

const VORLAGEN = [
	{ id: 1, typ: 'BESTELLUNG_HAENDLER', betreff: 'Buchbestellung', text_body: 'Sehr geehrte …' },
	{ id: 2, typ: 'MAHNUNG_ELTERN', betreff: 'Erinnerung', text_body: 'Liebe Eltern …' }
];

const antwort = (/** @type {any} */ daten, ok = true) =>
	/** @type {any} */ ({ ok, json: async () => daten });

async function vorlagen() {
	const screen = render(MailTemplates);
	await screen.findByLabelText('Text-Inhalt');
	return screen;
}

beforeEach(() => {
	vi.mocked(apiClient.get).mockReset().mockResolvedValue(antwort(VORLAGEN));
	vi.mocked(apiClient.put).mockReset().mockResolvedValue(antwort({}));
	for (const t of [...toastStore.toasts]) toastStore.removeToast(t.id);
});

describe('Mail-Vorlagen', () => {
	it('meldet das Speichern über die Meldungen der Anwendung', async () => {
		const screen = await vorlagen();
		await fireEvent.click(screen.getByRole('button', { name: /Speichern/ }));

		await waitFor(() =>
			expect(toastStore.toasts.map((t) => [t.message, t.type])).toEqual([
				['Vorlage gespeichert.', 'success']
			])
		);
		expect(apiClient.put).toHaveBeenCalledWith('/api/mail-templates/1', {
			betreff: 'Buchbestellung',
			text_body: 'Sehr geehrte …'
		});
	});

	// Die Liste ist eine Auswahl wie die Kategorien links daneben: Die gewählte Vorlage
	// trägt aria-current, damit ein Vorleseprogramm sie nennt.
	it('kennzeichnet die gewählte Vorlage und wechselt mit einem Klick', async () => {
		const screen = await vorlagen();
		const [bestellung, mahnung] = screen
			.getAllByRole('button')
			.filter((k) => /Bestellung|Mahnbrief/.test(k.textContent ?? ''));
		expect(bestellung.getAttribute('aria-current')).toBe('true');
		expect(mahnung.getAttribute('aria-current')).toBeNull();

		await fireEvent.click(mahnung);
		expect(mahnung.getAttribute('aria-current')).toBe('true');
		expect(/** @type {HTMLTextAreaElement} */ (screen.getByLabelText('Text-Inhalt')).value).toBe(
			'Liebe Eltern …'
		);
	});

	// Der Schlüssel aus der Datenbank („BESTELLUNG_HAENDLER") ist kein Wort für Menschen. Die
	// Liste nennt die Vorlage so, wie der Hinweis unter dem Editor sie beschreibt.
	it('nennt die Vorlagen mit ihrem Namen statt mit dem Schlüssel', async () => {
		vi.mocked(apiClient.get).mockResolvedValue(
			antwort([...VORLAGEN, { id: 3, typ: 'NOCH_OHNE_NAMEN', betreff: 'Neu', text_body: '' }])
		);
		const screen = await vorlagen();
		const namen = screen
			.getAllByRole('button')
			.filter((k) => k.hasAttribute('aria-current') || /Mahnbrief|OHNE/.test(k.textContent ?? ''))
			.map((k) => k.firstElementChild?.textContent?.trim());

		expect(namen).toEqual([
			'Bestellung an den Händler',
			'Mahnbrief an die Eltern',
			'NOCH OHNE NAMEN'
		]);
		expect(screen.container.textContent).not.toMatch(/BESTELLUNG|MAHNUNG/);
	});

	it('nennt einen Ladefehler als Meldung', async () => {
		vi.mocked(apiClient.get).mockResolvedValue(antwort({}, false));
		const screen = render(MailTemplates);
		const meldung = await screen.findByRole('alert');
		expect(meldung.textContent).toContain('Fehler beim Laden der Vorlagen.');
		// Neben dem Fehler behauptet die Liste nicht, sie lade noch.
		expect(screen.queryByText(/Lade Vorlagen/)).toBeNull();
	});

	// Die Kategorie trägt die Überschrift „Mail-Vorlagen" schon; eine zweite, größere
	// darunter kehrte die Ordnung der Seite um.
	it('trägt keine zweite Überschrift neben der der Kategorie', async () => {
		const screen = await vorlagen();
		expect(screen.queryAllByRole('heading', { level: 3 })).toEqual([]);
	});
});
