import { describe, it, expect, vi, afterEach } from 'vitest';
import { fireEvent, render } from '@testing-library/svelte';
import FehlbestandBericht from './FehlbestandBericht.svelte';
import { toastStore } from '../../stores/toastStore.svelte.js';
import { FENSTER_BLOCKIERT } from '../../utils/listenDruck.js';

const PROPS = {
	label: 'Lehrbuchsammlung',
	onSchliessen: () => {},
	onGefunden: async () => {},
	onEndgueltigLoeschen: async () => {}
};
const OFFEN = {
	exemplar_id: 'e-1',
	barcode_id: '10004200',
	signatur: 'Ma 7.1',
	titel: 'Elemente der Mathematik 7',
	autor: 'Griesel, Heinz'
};
const GEFUNDEN = {
	exemplar_id: 'e-2',
	barcode_id: '10004201',
	signatur: 'Bel SCH',
	titel: 'Der Vorleser',
	autor: 'Schlink, Bernhard',
	gefunden_am: '2026-10-05T09:00:00+02:00'
};
// Ohne exemplar_id: Das Exemplar ist schon endgültig gelöscht, zu suchen gibt es nichts mehr.
const GELOESCHT = { barcode_id: '10004202', signatur: 'Ek 0.1', titel: 'Diercke Weltatlas' };

/** Ein Fenster, wie window.open es liefert, soweit der Druck es anfasst. */
function druckfenster() {
	return {
		document: { open: vi.fn(), write: vi.fn(), close: vi.fn() },
		focus: vi.fn(),
		print: vi.fn()
	};
}

afterEach(() => {
	vi.restoreAllMocks();
});

describe('FehlbestandBericht', () => {
	it('druckt mit „Liste drucken" die Exemplare, die noch fehlen', async () => {
		const fenster = druckfenster();
		vi.spyOn(window, 'open').mockReturnValue(/** @type {any} */ (fenster));
		const bericht = render(FehlbestandBericht, {
			...PROPS,
			eintraege: [OFFEN, GEFUNDEN, GELOESCHT]
		});

		await fireEvent.click(bericht.getByRole('button', { name: 'Liste drucken' }));

		const html = fenster.document.write.mock.calls[0][0];
		expect(html).toContain('<h1>Fehlbestand — Lehrbuchsammlung</h1>');
		expect(html).toContain('10004200');
		// Gefundenes und endgültig Gelöschtes steht nur als Zahl auf dem Blatt.
		expect(html).not.toContain('10004201');
		expect(html).not.toContain('10004202');
		expect(html).toContain('Als Verlust gebucht: 3 | bereits geklärt: 2 | noch offen: 1');
		expect(fenster.print).toHaveBeenCalledTimes(1);
	});

	it('sperrt „Liste drucken", wenn nichts mehr zu suchen ist', () => {
		const bericht = render(FehlbestandBericht, { ...PROPS, eintraege: [GEFUNDEN, GELOESCHT] });
		expect(bericht.getByRole('button', { name: 'Liste drucken' })).toHaveProperty('disabled', true);
	});

	it('meldet, wenn der Browser das Druckfenster nicht öffnet', async () => {
		vi.spyOn(window, 'open').mockReturnValue(null);
		const meldung = vi.spyOn(toastStore, 'addToast').mockImplementation(() => {});
		const bericht = render(FehlbestandBericht, { ...PROPS, eintraege: [OFFEN] });

		await fireEvent.click(bericht.getByRole('button', { name: 'Liste drucken' }));

		expect(meldung).toHaveBeenCalledWith(FENSTER_BLOCKIERT, 'warning');
	});
});
