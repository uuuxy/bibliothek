import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import StatistikDetailPage from './StatistikDetailPage.svelte';
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
});

describe('Statistik-Detailseite', () => {
	it('nennt je Titel Zahl und Autor', async () => {
		const screen = render(StatistikDetailPage);
		expect(await screen.findByText('Brücke nach Terabithia')).toBeTruthy();
		expect(screen.getByText('12×')).toBeTruthy();
		expect(screen.getByText('2 von 2 Einträgen')).toBeTruthy();
	});

	// Ohne Cover steht die Initiale wie in der Übersicht und in den übrigen Listen
	// (ui/BuchCover), nicht ein Buch-Emoji.
	it('zeigt ohne Cover die Initiale statt eines Emojis', async () => {
		const screen = render(StatistikDetailPage);
		const zeile = (await screen.findByText('Brücke nach Terabithia')).closest('li');
		expect(zeile?.textContent).not.toContain('📖');
		expect(zeile?.firstElementChild?.textContent?.trim()).toBe('B');
		expect(screen.container.querySelector('li img[src*="momo"]')).toBeTruthy();
	});
});
