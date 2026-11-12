import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import EhemaligeListe from './EhemaligeListe.svelte';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	// Die echte Auspack-Logik steht in apiFetch.test.js; hier genügt die Form
	// {"error": …}, die das Backend einheitlich liefert (apierrors.schreibeFehlerJSON).
	leseApiFehler: vi.fn(async (/** @type {any} */ res) => JSON.parse(await res.text()).error)
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
 *  @param {any[]} daten @param {number} [grenze] was der Server im Kopf X-Grenze nennt
 *  @returns {any} */
const antwort = (daten, grenze = 0) => ({
	ok: true,
	headers: new Headers(grenze ? { 'X-Grenze': String(grenze) } : {}),
	json: async () => daten
});

describe('EhemaligeListe', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	it('lädt über /api/schueler?status=ehemalige und zeigt Abgang, Name, offene Bücher', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(ehemalige));
		const { findByRole, getByText } = render(EhemaligeListe, { onSelect: () => {} });

		expect(
			// Der Name kommt aus den Zellen, damit auch der Bildschirmleser die offenen Bücher hört.
			await findByRole('button', { name: /Wera Weggegangen .*Profil anzeigen$/ })
		).toBeTruthy();
		expect(vi.mocked(apiFetch).mock.calls[0][0]).toBe('/api/schueler?status=ehemalige');
		expect(getByText('2026')).toBeTruthy();
		expect(getByText(/1 überfällig/)).toBeTruthy();
		expect(getByText('Sperre aktiv')).toBeTruthy();
	});

	// Ohne Suchwort kappt der Server die Liste und nennt die Grenze. Ohne den Satz sähen 500
	// Ehemalige aus wie alle, und wer jemanden dahinter sucht, hielte ihn für nicht da.
	it('sagt, dass die Liste gekappt ist, und schweigt bei vollständiger Liste', async () => {
		const drei = Array.from({ length: 3 }, (_, i) => ({ ...ehemalige[0], id: `g${i}` }));
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(drei, 3)).mockResolvedValue(antwort(drei));
		const { findByText, getByRole, queryByText } = render(EhemaligeListe, { onSelect: () => {} });

		expect(
			await findByText(/^Die ersten 3 Ehemaligen\. Suche nach Name oder Ausweisnummer/)
		).toBeTruthy();

		// Mit Suchwort nennt der Server keine Grenze: Die Liste ist vollständig, der Satz geht.
		await fireEvent.input(getByRole('searchbox', { name: 'Ehemalige suchen' }), {
			target: { value: 'Wera' }
		});
		await vi.waitFor(() => expect(vi.mocked(apiFetch)).toHaveBeenCalledTimes(2), { timeout: 2000 });
		await vi.waitFor(() => expect(queryByText(/^Die ersten \d+ Ehemaligen/)).toBeNull());
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

	// Ein Handscanner mit amerikanischer Tastatur tippt an einem deutschen Rechner den
	// Bindestrich als ß (FACHKONZEPT §1).
	it('liest in einer gescannten Nummer ein ß hinter der Vorsilbe als Bindestrich', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort([]));
		const { getByRole, findByText } = render(EhemaligeListe, { onSelect: () => {} });
		await findByText('Keine Ehemaligen im Archiv.');

		await fireEvent.input(getByRole('searchbox', { name: 'Ehemalige suchen' }), {
			target: { value: 'Sß10001' }
		});
		await vi.waitFor(() => {
			const urls = vi.mocked(apiFetch).mock.calls.map((c) => c[0]);
			expect(urls).toContain('/api/schueler?status=ehemalige&q=S-10001');
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

	// Auch das Lesen des Rumpfs dauert: Die erste Antwort kann schon da sein, während ihr Rumpf
	// erst nach der Antwort auf die Suche gelesen ist.
	it('zeigt nicht die Liste einer überholten Antwort, deren Rumpf zuletzt gelesen ist', async () => {
		/** @type {(daten: unknown) => void} */
		let gibErste = () => {};
		const liesErste = vi.fn(() => new Promise((ja) => (gibErste = ja)));
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(
				/** @type {any} */ ({ ok: true, headers: new Headers(), json: liesErste })
			)
			.mockResolvedValue(antwort([{ ...ehemalige[0], id: 'e2', vorname: 'Xaver' }]));
		const { getByRole, findByRole } = render(EhemaligeListe, { onSelect: () => {} });
		await vi.waitFor(() => expect(liesErste).toHaveBeenCalled());

		await fireEvent.input(getByRole('searchbox', { name: 'Ehemalige suchen' }), {
			target: { value: 'Xaver' }
		});
		await findByRole('button', { name: /Xaver Weggegangen/ });
		gibErste(ehemalige);
		await new Promise((fertig) => setTimeout(fertig, 20));

		expect(document.body.textContent, 'die Liste der überholten Antwort steht da').not.toContain(
			'Wera Weggegangen'
		);
	});
});
