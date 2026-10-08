import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', async (importOriginal) => ({
	...(await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('../../../../lib/stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn() }));
vi.mock('$lib/store.svelte.js', () => ({ appState: { bookToEdit: null }, showToast: vi.fn() }));

import { apiFetch } from '../../../../lib/apiFetch.js';
import BuchFormular from './BuchFormular.svelte';
import { geaenderteFelder } from '../../buch_felder.js';
import { titelFuerMaske } from '../../buch_speichern.js';

/** @param {number} status @param {any} koerper */
const antwort = (status, koerper) =>
	/** @type {any} */ ({ ok: status < 400, status, json: async () => koerper });

// Zwei Titel, wie der Einzelabruf sie liefert: ein gepflegtes Lernmittel und ein Titel, wie
// ihn ein Import hinterlässt, mit leeren Feldern und der Klasse 0.
const GEPFLEGT = {
	id: 'titel-1',
	isbn: '9783060130764',
	title: 'Natura 2',
	author: 'Beyer, Irmtraud',
	coverUrl: '/covers/natura.webp',
	subject: 'Biologie',
	track: 'G',
	signatur: 'Bio 7',
	istLernmittel: true,
	stock: 30,
	verfuegbar: 12,
	gesamt: 30,
	imZulauf: 0,
	lastCounted: '2026-01-15',
	sortOrder: 4,
	medientyp: 'Buch',
	jahrgangVon: 7,
	jahrgangBis: 10,
	mehrjahresband: true,
	untertitel: 'Biologie für Gymnasien',
	auflage: '1. Aufl. 2019',
	listenpreis: 31.5,
	verlag: 'Klett',
	erscheinungsjahr: 2019,
	erweiterteEigenschaften: { littera_id: 4711 },
	schlagworte: ['Biologie', 'Zelle']
};
const AUS_DEM_IMPORT = {
	id: 'titel-2',
	isbn: '',
	title: 'Bild der Wissenschaft',
	author: '',
	coverUrl: '',
	subject: '',
	track: '',
	signatur: '',
	istLernmittel: false,
	stock: 0,
	verfuegbar: 0,
	gesamt: 0,
	imZulauf: 0,
	lastCounted: null,
	sortOrder: 9,
	medientyp: 'Zeitschrift',
	jahrgangVon: 5,
	jahrgangBis: 10,
	mehrjahresband: false,
	untertitel: '',
	auflage: '',
	listenpreis: null,
	verlag: '',
	erscheinungsjahr: 0,
	erweiterteEigenschaften: {},
	schlagworte: []
};

/** Öffnet die Maske wie die Seite: Einzelabruf, Stand vom Öffnen, dann die Maske darüber.
 * @param {any} titel */
async function oeffne(titel) {
	vi.mocked(apiFetch).mockImplementation(async (url) =>
		String(url) === `/api/books/${titel.id}` ? antwort(200, titel) : antwort(200, [])
	);
	const formular = $state(await titelFuerMaske({ id: titel.id }));
	const screen = render(BuchFormular, {
		formular,
		onClose: () => {},
		onSave: () => {},
		onCoverUpload: () => {},
		onCoverNeuHolen: () => {},
		onAssignClass: () => {}
	});
	// Was die Maske nach dem Aufbau noch nachlädt und einträgt, ist dann angekommen.
	await new Promise((r) => setTimeout(r, 50));
	return { formular, screen };
}

beforeEach(() => vi.clearAllMocks());

// Ein Feld, das die Maske beim Aufbau selbst belegt, gälte als geändert und ginge beim
// Speichern mit dem Stand vom Öffnen hinaus, über das, was ein anderer Platz dort
// inzwischen gespeichert hat.
describe('BuchFormular: eine geöffnete Maske ohne Eingabe', () => {
	it.each([
		['ein gepflegtes Lernmittel', GEPFLEGT],
		['ein Titel aus dem Import', AUS_DEM_IMPORT]
	])('%s: kein Feld gilt als geändert', async (_name, titel) => {
		const { formular } = await oeffne(titel);
		expect(geaenderteFelder(formular)).toEqual({});
	});

	it('eine Eingabe im Feld „Verlag": nur dieses Feld', async () => {
		const { formular, screen } = await oeffne(GEPFLEGT);
		await fireEvent.input(screen.getByLabelText('Verlag'), { target: { value: 'Cornelsen' } });
		expect(geaenderteFelder(formular)).toEqual({ verlag: 'Cornelsen' });
	});
});
