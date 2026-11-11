import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import StatistikDetail from './StatistikDetail.svelte';
import { apiFetch } from '../../apiFetch.js';
import { uiStore } from '../../stores/uiStore.svelte.js';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

const RENNER = [
	{ id: 't1', titel: 'Brücke nach Terabithia', autor: 'Paterson', count: 12 },
	{ id: 't2', titel: 'Momo', autor: 'Ende', count: 9, cover_url: '/covers/momo.webp' }
];

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
	vi.mocked(apiFetch).mockResolvedValue(
		/** @type {any} */ ({ ok: true, json: async () => ({ popular_titles: RENNER }) })
	);
	uiStore.statsDetailKind = 'renner';
	uiStore.statsDetailFilter = { zeitraum: 'all', medientyp: '' };
});

// Die Detailseite lud immer ohne Zeitraum und Bestand und sagte trotzdem „im gewählten
// Zeitraum". Jetzt lädt sie die Auswahl der Übersicht und nennt sie.
describe('Statistik-Detailseite: Auswahl der Übersicht', () => {
	it('lädt Zeitraum und Bestand und nennt sie unter der Überschrift', async () => {
		uiStore.statsDetailFilter = { zeitraum: 'monat', medientyp: 'lmf' };
		const screen = render(StatistikDetail);
		await screen.findByText('Brücke nach Terabithia');
		expect(String(vi.mocked(apiFetch).mock.calls.at(-1)?.[0])).toBe(
			'/api/statistiken?limit=100&zeitraum=monat&type=lmf'
		);
		expect(
			screen.getByText('Nach Anzahl der Ausleihen in den letzten 30 Tagen, nur LMF.')
		).toBeTruthy();
	});

	it('nennt ohne Auswahl alle Zeiträume und den ganzen Bestand (direkter Aufruf)', async () => {
		const screen = render(StatistikDetail);
		await screen.findByText('Brücke nach Terabithia');
		expect(String(vi.mocked(apiFetch).mock.calls.at(-1)?.[0])).toBe('/api/statistiken?limit=100');
		expect(screen.getByText('Nach Anzahl der Ausleihen, alle Zeiträume.')).toBeTruthy();
	});

	it('nennt bei den Ladenhütern nur den Bestand', async () => {
		uiStore.statsDetailKind = 'ladenhueter';
		uiStore.statsDetailFilter = { zeitraum: 'monat', medientyp: 'freihand' };
		const screen = render(StatistikDetail);
		expect(
			await screen.findByText('Seit über 2 Jahren nicht ausgeliehen — oder noch nie, nur Freihand.')
		).toBeTruthy();
	});
});

describe('Statistik-Detailseite', () => {
	it('nennt je Titel Zahl und Autor', async () => {
		const screen = render(StatistikDetail);
		expect(await screen.findByText('Brücke nach Terabithia')).toBeTruthy();
		expect(screen.getByText('12×')).toBeTruthy();
		expect(screen.getByText('2 von 2 Einträgen')).toBeTruthy();
	});

	// Ohne Cover steht die Initiale wie in der Übersicht und in den übrigen Listen
	// (ui/BuchCover), nicht ein Buch-Emoji.
	it('zeigt ohne Cover die Initiale statt eines Emojis', async () => {
		const screen = render(StatistikDetail);
		const zeile = (await screen.findByText('Brücke nach Terabithia')).closest('li');
		expect(zeile?.textContent).not.toContain('📖');
		expect(zeile?.firstElementChild?.textContent?.trim()).toBe('B');
		expect(screen.container.querySelector('li img[src*="momo"]')).toBeTruthy();
	});

	// Eine Liste, die der Server nicht lesen konnte, kommt leer an und ist kein leerer Bestand.
	it('zeigt den Ladefehler, wenn der Server die Liste nicht lesen konnte', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({ popular_titles: [], nicht_gelesen: ['renner'] })
			})
		);
		const screen = render(StatistikDetail);
		expect(await screen.findByText('Statistik nicht geladen')).toBeTruthy();
		expect(screen.queryByText('Noch keine Daten vorhanden.')).toBeNull();
		expect(screen.queryByText('0 von 0 Einträgen')).toBeNull();
	});
});

// Der Satz unter der Überschrift erklärt, wonach die Liste geht; beim Ladenhüter ist es das Gegenteil.
describe('Statistik-Detailseite: Hinweis je Liste', () => {
	it('nennt je Liste ihren eigenen Maßstab', async () => {
		const renner = render(StatistikDetail);
		expect(await renner.findByText('Nach Anzahl der Ausleihen, alle Zeiträume.')).toBeTruthy();
		renner.unmount();

		uiStore.statsDetailKind = 'ladenhueter';
		const ladenhueter = render(StatistikDetail);
		expect(
			await ladenhueter.findByText('Seit über 2 Jahren nicht ausgeliehen — oder noch nie.')
		).toBeTruthy();
	});
});

// Die Suche auf der Seite filtert die geladene Liste nach Titel oder Autor.
describe('Statistik-Detailseite: Suche', () => {
	it('zeigt nur die Titel, die zum Suchwort passen', async () => {
		const screen = render(StatistikDetail);
		await screen.findByText('Momo');

		await fireEvent.input(screen.getByLabelText('Titel oder Autor suchen'), {
			target: { value: 'pater' }
		});

		expect(screen.getByText('Brücke nach Terabithia')).toBeTruthy();
		expect(screen.queryByText('Momo')).toBeNull();
	});
});
