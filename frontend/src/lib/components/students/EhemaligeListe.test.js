import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import EhemaligeListe from './EhemaligeListe.svelte';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	// Die echte Auspack-Logik steht in apiFetch.test.js; hier genügt die Form
	// {"error": …}, die das Backend einheitlich liefert (apierrors.writeJSONError).
	extractApiError: vi.fn(async (/** @type {any} */ res) => JSON.parse(await res.text()).error)
}));

// Der Reiter „Ehemalige / Archiv" muss die WEGGEGANGENEN laden (status=ehemalige) —
// nicht die Abgängerliste, die er bis zum 05.09.2026 eingebettet hatte und die seitdem
// die Abschlussklassen meint. Geprüft wird die Tür, durch die er geht, und was er zeigt.
const ehemalige = [
	{
		id: 'e1',
		vorname: 'Wera',
		nachname: 'Weggegangen',
		barcode_id: 'S-EHEM-1',
		abgaenger_jahr: 2026,
		ausgeliehen_count: 1,
		ueberfaellig_count: 1,
		ist_gesperrt: true
	}
];

/** Antwort wie apiFetch sie liefert — nur das, was die Komponente liest.
 *  @param {any[]} daten @returns {any} */
const antwort = (daten) => ({ ok: true, json: async () => daten });

describe('EhemaligeListe', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	it('lädt über /api/schueler?status=ehemalige und zeigt Abgang, Name, offene Bücher', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(ehemalige));
		const { findByRole, getByText } = render(EhemaligeListe, { onSelect: () => {} });

		expect(
			await findByRole('button', { name: 'Profil von Wera Weggegangen anzeigen' })
		).toBeTruthy();
		expect(vi.mocked(apiFetch).mock.calls[0][0]).toBe('/api/schueler?status=ehemalige');
		expect(getByText('2026')).toBeTruthy();
		expect(getByText(/1 überfällig/)).toBeTruthy();
		expect(getByText('Sperre aktiv')).toBeTruthy();
	});

	it('Klick auf die Zeile öffnet das Profil', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(ehemalige));
		const onSelect = vi.fn();
		const { findByRole } = render(EhemaligeListe, { onSelect });
		await fireEvent.click(await findByRole('button', { name: /Wera Weggegangen/ }));
		expect(onSelect).toHaveBeenCalledWith(ehemalige[0]);
	});

	it('Suche läuft auf dem Server mit demselben Status', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort([]));
		const { getByRole, findByText } = render(EhemaligeListe, { onSelect: () => {} });
		await findByText('Keine Ehemaligen im Archiv.');

		await fireEvent.input(getByRole('searchbox', { name: 'Ehemalige suchen' }), {
			target: { value: 'Wera' }
		});
		await findByText('Keine Ehemaligen gefunden.');
		// Die Suche ist entprellt (250 ms) — auf den Aufruf warten, nicht auf den Text.
		await vi.waitFor(() => {
			const urls = vi.mocked(apiFetch).mock.calls.map((c) => c[0]);
			expect(urls).toContain('/api/schueler?status=ehemalige&q=Wera');
		});
	});

	// Ein gescheiterter Abruf ist KEINE Trefferliste. Bis zum 12.09.2026 schrieb `if
	// (res.ok && nr === ladeNr)` die Liste nur im Erfolgsfall — scheiterte die Suche,
	// blieben die Treffer der VORIGEN Suche stehen, unter dem neuen Suchtext. Genau die
	// Form, mit der der Sweep „verschluckte Fehlantwort" angefangen hat (die Theke zeigte
	// unter dem neuen Text die Schüler des alten); der Detektor sah sie nur nicht, weil
	// die Bedingung ein zweites Glied trägt.
	it('zeigt nach einem gescheiterten Suchlauf nicht die Treffer von vorher', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(ehemalige));
		const { getByRole, findByRole, findByText } = render(EhemaligeListe, { onSelect: () => {} });
		await findByRole('button', { name: /Wera Weggegangen/ });

		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: false,
				status: 500,
				text: async () => JSON.stringify({ error: 'Datenbank nicht erreichbar' })
			})
		);
		await fireEvent.input(getByRole('searchbox', { name: 'Ehemalige suchen' }), {
			target: { value: 'Xaver' }
		});

		expect(await findByText('Datenbank nicht erreichbar')).toBeTruthy();
		expect(document.body.textContent).not.toContain('Wera Weggegangen');
	});
});
