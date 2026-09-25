import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', () => ({
	apiFetch: vi.fn(),
	apiClient: { post: vi.fn() },
	extractApiError: vi.fn(async (/** @type {any} */ res) => `Fehler ${res.status}`)
}));
vi.mock('$lib/store.svelte.js', () => ({ showToast: vi.fn() }));

import { apiFetch, apiClient } from '../../../../lib/apiFetch.js';
import AuflageZuordnenDialog from './AuflageZuordnenDialog.svelte';

// Der Dialog „Andere Auflage zuordnen" (docs/OFFEN.md 4.18, Stufe 2). Material 3, Dialogs:
// „Disable confirming actions until a choice is made. Dismissive actions are never
// disabled." Dazu: Gehört der gewählte Titel schon zu einem Buch, kommen dessen Auflagen
// mit — das steht vor dem Klick da, und wenn die Nachfrage scheitert, sagt der Dialog das.

const treffer = {
	id: 'neu',
	title: 'Mathe 7',
	auflage: '4. Aufl.',
	erscheinungsjahr: 2023,
	istLernmittel: true,
	gesamt: 30,
	verfuegbar: 30,
	imZulauf: 0
};
const buecher = (liste) => ({ ok: true, status: 200, json: async () => ({ data: liste }) });
const auflagen = (n) => ({
	ok: true,
	status: 200,
	json: async () => ({ auflagen: Array.from({ length: n }, (_, i) => ({ id: `a${i}` })) })
});

async function bisZumTreffer() {
	const onZugeordnet = vi.fn();
	const screen = render(AuflageZuordnenDialog, {
		props: { open: true, titelId: 'alt', bekannte: ['alt'], onZugeordnet }
	});
	await fireEvent.input(screen.getByLabelText('Andere Auflage suchen'), {
		target: { value: 'Mathe' }
	});
	await vi.advanceTimersByTimeAsync(250);
	return { screen, onZugeordnet };
}

beforeEach(() => {
	vi.useFakeTimers();
	vi.mocked(apiFetch).mockReset();
	vi.mocked(apiClient.post).mockReset();
});
afterEach(() => {
	vi.useRealTimers();
});

describe('AuflageZuordnenDialog', () => {
	it('sperrt „Zuordnen" bis zur Wahl, „Abbrechen" nie', async () => {
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(/** @type {any} */ (buecher([treffer])))
			.mockResolvedValueOnce(/** @type {any} */ (buecher([])))
			.mockResolvedValueOnce(/** @type {any} */ (auflagen(1)));
		const { screen } = await bisZumTreffer();

		const zuordnen = screen.getByRole('button', { name: 'Zuordnen' });
		expect(zuordnen.hasAttribute('disabled')).toBe(true);
		expect(screen.getByRole('button', { name: 'Abbrechen' }).hasAttribute('disabled')).toBe(false);

		await fireEvent.click(screen.getByRole('button', { name: /4\. Aufl\. · 2023/ }));
		await vi.advanceTimersByTimeAsync(0);
		expect(zuordnen.hasAttribute('disabled')).toBe(false);
	});

	it('sagt vor dem Klick, dass Auflagen mitkommen', async () => {
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(/** @type {any} */ (buecher([treffer])))
			.mockResolvedValueOnce(/** @type {any} */ (buecher([])))
			.mockResolvedValueOnce(/** @type {any} */ (auflagen(3)));
		const { screen } = await bisZumTreffer();
		await fireEvent.click(screen.getByRole('button', { name: /4\. Aufl\. · 2023/ }));
		await vi.advanceTimersByTimeAsync(0);
		expect(
			screen.getByText(/schon mit 2\s+anderen Auflagen zusammengefasst; alle kommen mit/)
		).toBeTruthy();
	});

	it('behauptet nichts, wenn die Nachfrage scheitert', async () => {
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(/** @type {any} */ (buecher([treffer])))
			.mockResolvedValueOnce(/** @type {any} */ (buecher([])))
			.mockResolvedValueOnce(/** @type {any} */ ({ ok: false, status: 500 }));
		const { screen } = await bisZumTreffer();
		await fireEvent.click(screen.getByRole('button', { name: /4\. Aufl\. · 2023/ }));
		await vi.advanceTimersByTimeAsync(0);
		expect(screen.getByText(/ließ sich nicht prüfen/)).toBeTruthy();
	});

	it('meldet eine Abweisung des Servers im Dialog und reicht nichts weiter', async () => {
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(/** @type {any} */ (buecher([treffer])))
			.mockResolvedValueOnce(/** @type {any} */ (buecher([])))
			.mockResolvedValueOnce(/** @type {any} */ (auflagen(1)));
		vi.mocked(apiClient.post).mockResolvedValue(/** @type {any} */ ({ ok: false, status: 400 }));
		const { screen, onZugeordnet } = await bisZumTreffer();
		await fireEvent.click(screen.getByRole('button', { name: /4\. Aufl\. · 2023/ }));
		await vi.advanceTimersByTimeAsync(0);
		await fireEvent.click(screen.getByRole('button', { name: 'Zuordnen' }));
		await vi.advanceTimersByTimeAsync(0);

		expect(apiClient.post).toHaveBeenCalledWith('/api/buecher/titel/alt/auflagen', {
			titel_id: 'neu'
		});
		expect(screen.getByRole('alert').textContent).toContain('Fehler 400');
		expect(onZugeordnet).not.toHaveBeenCalled();
	});
});
