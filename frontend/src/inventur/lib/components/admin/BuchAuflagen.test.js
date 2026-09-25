import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', () => ({
	apiFetch: vi.fn(),
	apiClient: { post: vi.fn() },
	extractApiError: vi.fn(async (/** @type {any} */ res) => `Fehler ${res.status}`)
}));
vi.mock('$lib/store.svelte.js', () => ({ showToast: vi.fn() }));

import { apiFetch } from '../../../../lib/apiFetch.js';
import BuchAuflagen from './BuchAuflagen.svelte';

// Der Abschnitt „Auflagen" der Titelmaske (docs/OFFEN.md 4.18, Stufe 2). GET
// /api/buecher/titel/{id}/auflagen antwortet mit { id, auflagen } — die Auflagen des Buchs,
// die neueste zuerst, ohne Gruppe nur der Titel selbst (api/auflagen_handler.go).

const auflage = (id, auflage, jahr, gesamt = 30, verfuegbar = 28) => ({
	id,
	titel: 'Lambacher Schweizer 7',
	auflage,
	isbn: `978-${id}`,
	verlag: 'Klett',
	erscheinungsjahr: jahr,
	ist_lernmittel: true,
	gesamt,
	verfuegbar,
	im_zulauf: 0
});
const antwort = (liste) => ({ ok: true, status: 200, json: async () => ({ auflagen: liste }) });
const stillhalten = async () => {
	for (let i = 0; i < 6; i++) await Promise.resolve();
};

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
});

describe('BuchAuflagen', () => {
	it('bietet bei einem Lernmittel ohne andere Auflage das Zuordnen an', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ (antwort([auflage('t1', '3. Aufl.', 2019)]))
		);
		const screen = render(BuchAuflagen, { formular: { id: 't1', istLernmittel: true } });

		expect(await screen.findByText('Keine andere Auflage zugeordnet.')).toBeTruthy();
		expect(screen.getByRole('heading', { name: 'Auflagen' })).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Andere Auflage zuordnen' })).toBeTruthy();
		expect(apiFetch).toHaveBeenCalledWith('/api/buecher/titel/t1/auflagen');
	});

	it('bleibt bei einem Bibliotheksbuch ohne Auflagen weg', async () => {
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ (antwort([auflage('t1', '', 2019)])));
		const screen = render(BuchAuflagen, { formular: { id: 't1', istLernmittel: false } });
		await stillhalten();
		expect(screen.queryByRole('heading', { name: 'Auflagen' })).toBeNull();
	});

	it('zeigt die Auflagen mit Summe — und einen Titel, der kein Lernmittel mehr ist, damit er sich lösen lässt', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ (
				antwort([auflage('t2', '4. Aufl.', 2023, 30, 28), auflage('t1', '3. Aufl.', 2019, 65, 60)])
			)
		);
		const screen = render(BuchAuflagen, { formular: { id: 't1', istLernmittel: false } });

		expect(await screen.findByText('4. Aufl. · 2023')).toBeTruthy();
		// Über den ganzen Absatz: Der Zusatz steht in einem eigenen span, und genau an der Naht
		// fehlte am 25.09.2026 das Leerzeichen („2019— diese Auflage").
		expect(screen.getByText(/3\. Aufl\. · 2019/).closest('p')?.textContent).toMatch(
			/2019 — diese Auflage/
		);
		expect(screen.getByText('Zusammen: 88 von 95 verfügbar')).toBeTruthy();
	});

	it('löst eine Auflage über ihre eigene Kennung und liest die Liste neu', async () => {
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(
				/** @type {any} */ (
					antwort([auflage('t2', '4. Aufl.', 2023), auflage('t1', '3. Aufl.', 2019)])
				)
			)
			.mockResolvedValueOnce(/** @type {any} */ (antwort([auflage('t2', '4. Aufl.', 2023)])))
			.mockResolvedValueOnce(/** @type {any} */ (antwort([auflage('t1', '3. Aufl.', 2019)])));
		const screen = render(BuchAuflagen, { formular: { id: 't1', istLernmittel: true } });

		await fireEvent.click(
			await screen.findByRole('button', { name: '4. Aufl. · 2023 aus den Auflagen lösen' })
		);
		await stillhalten();

		expect(apiFetch).toHaveBeenNthCalledWith(2, '/api/buecher/titel/t2/auflagen', {
			method: 'DELETE'
		});
		expect(apiFetch).toHaveBeenNthCalledWith(3, '/api/buecher/titel/t1/auflagen');
		expect(await screen.findByText('Keine andere Auflage zugeordnet.')).toBeTruthy();
	});
});
