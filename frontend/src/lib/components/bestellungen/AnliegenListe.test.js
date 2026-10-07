import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import AnliegenListe from './AnliegenListe.svelte';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	extractApiError: vi.fn(async (/** @type {any} */ res) => JSON.parse(await res.text()).error)
}));
vi.mock('../../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

// „Keine offenen Anliegen" ist eine Aussage über die Arbeit der Bibliothek: nichts zu tun.
// Bis zum 12.09.2026 stand derselbe Satz da, wenn der Abruf gescheitert war
// (`anliegen = res.ok ? await res.json() : []`) — die Wünsche und Meldungen des
// Kollegiums blieben dann liegen, ohne dass jemand davon wusste (Register,
// Bestands-Durchgang 10.09.2026).
describe('AnliegenListe', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	it('unterscheidet „nichts zu tun" von „nicht geladen"', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: false,
				status: 503,
				text: async () => JSON.stringify({ error: 'Datenbank nicht erreichbar' })
			})
		);
		const { findByText, queryByText } = render(AnliegenListe);

		expect(await findByText('Datenbank nicht erreichbar')).toBeTruthy();
		expect(queryByText('Keine offenen Anliegen.')).toBeNull();
	});

	it('leer bleibt leer', async () => {
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: true, json: async () => [] }));
		const { findByText } = render(AnliegenListe);
		expect(await findByText('Keine offenen Anliegen.')).toBeTruthy();
	});

	// Ein Wunsch wartet auf die nächste Bestellung, ein Problem soll am selben Tag erledigt
	// werden. In einer Liste nach Alter stünde die Meldung von heute unter den Wünschen.
	it('stellt die Meldungen über die Wünsche, auch wenn der Wunsch älter ist', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => [
					{
						id: 'w',
						art: 'wunsch',
						titel_text: 'Alter Wunsch',
						klasse: '7A',
						erstellt_am: '2026-09-01T08:00:00Z'
					},
					{
						id: 'm',
						art: 'meldung',
						titel_text: 'Neue Meldung',
						klasse: '8G3',
						erstellt_am: '2026-10-05T08:00:00Z'
					}
				]
			})
		);
		const { findByRole, getAllByRole, container } = render(AnliegenListe);

		// Die Meldungen stehen ohne Zwischenüberschrift unter der Überschrift der Seite: Eine
		// zweite Zeile „Meldungen" stünde direkt unter der ersten.
		await findByRole('heading', { name: 'Wünsche' });
		expect(getAllByRole('heading', { level: 2 }).map((h) => h.textContent?.trim())).toEqual([
			'Meldungen'
		]);
		expect(getAllByRole('heading', { level: 3 }).map((h) => h.textContent?.trim())).toEqual([
			'Wünsche'
		]);
		const text = container.textContent ?? '';
		expect(text.indexOf('Neue Meldung')).toBeLessThan(text.indexOf('Alter Wunsch'));
	});

	it('zeigt einen Abschnitt nur, wenn er Einträge hat', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => [
					{
						id: 'w',
						art: 'wunsch',
						titel_text: 'Ein Wunsch',
						klasse: '7A',
						erstellt_am: '2026-09-01T08:00:00Z'
					}
				]
			})
		);
		const { findByRole, container } = render(AnliegenListe);

		await findByRole('heading', { name: 'Wünsche' });
		expect(container.querySelectorAll('section')).toHaveLength(1);
	});
});
