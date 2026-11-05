import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import Bestandsbuch from './Bestandsbuch.svelte';
import { serverAntwort } from '../../serverAntwort.js';

const LAND = 'Lernmittelfreiheit (Land)';
const TRAEGER = 'Schülerbücherei (Schulträger)';
const OHNE = 'ohne Zuordnung';
const LEER = 'Kein Zugang in diesem Zeitraum';

const spalten = [
	{ kopf: 'Zugang', feld: 'datum' },
	{ kopf: 'Titel', feld: 'titel' },
	{ kopf: 'Anzahl', feld: 'anzahl' },
	{ kopf: 'Barcodes', feld: 'barcodes' }
];

/** @param {number} anzahl @param {string} kuerzel */
const zeilen = (anzahl, kuerzel) =>
	Array.from({ length: anzahl }, (_, i) => ({
		datum: '2026-09-20T00:00:00Z',
		titel: `Titel ${kuerzel} ${i + 1}`,
		anzahl: 1,
		barcodes: `${kuerzel}-${i + 1}`
	}));

/** Die Antwort des Servers: Land und Schulträger stehen immer da, „ohne Zuordnung" nur mit Zeilen. */
const antwort = (/** @type {{ land?: number, traeger?: number, ohne?: number }} */ zahl) => {
	const abschnitte = [
		{ topf: 'land', titel: LAND, exemplare: zahl.land ?? 0, zeilen: zeilen(zahl.land ?? 0, 'L') },
		{
			topf: 'schultraeger',
			titel: TRAEGER,
			exemplare: zahl.traeger ?? 0,
			zeilen: zeilen(zahl.traeger ?? 0, 'T')
		}
	];
	if (zahl.ohne)
		abschnitte.push({
			topf: '',
			titel: OHNE,
			exemplare: zahl.ohne,
			zeilen: zeilen(zahl.ohne, 'O')
		});
	return /** @type {any} */ ({
		ok: true,
		json: async () => ({ von: '2026-09-16', bis: '2027-03-15', abschnitte })
	});
};

/** @param {{ land?: number, traeger?: number, ohne?: number }} zahl */
async function zugangsbuch(zahl) {
	vi.mocked(apiFetch).mockResolvedValue(antwort(zahl));
	const screen = render(Bestandsbuch, {
		pfad: 'zugangsbuch',
		buchname: 'Zugangsbuch',
		wortSingular: 'Zugang',
		spalten
	});
	await screen.findByRole('group', { name: 'Nach Mittelherkunft filtern' });
	return screen;
}

/**
 * Die Namen der Tabellen, die zu sehen sind.
 * @param {{ queryAllByRole: (rolle: string) => HTMLElement[] }} screen
 */
const tabellen = (screen) =>
	screen.queryAllByRole('table').map((t) => t.querySelector('caption')?.textContent);

/**
 * Die Zahl der Zeilen, die eine Tabelle zeichnet.
 * @param {{ getByRole: (rolle: string, optionen: { name: string }) => HTMLElement }} screen
 * @param {string} topf
 */
const gezeichnet = (screen, topf) =>
	screen.getByRole('table', { name: `Zugangsbuch — ${topf}` }).querySelectorAll('tbody tr').length;

beforeEach(() => vi.clearAllMocks());

// Die Töpfe stehen als Felder mit ihrer Zahl unter dem Zeitraum. Ein leerer Topf belegt unten
// keinen Abschnitt; dass nichts zuging, sagt die Null im Feld.
describe('Bestandsbuch: die Töpfe als Felder mit Zahl', () => {
	it('nennt jeden Topf mit seiner Zahl und zeigt unten nur Töpfe mit Einträgen', async () => {
		const screen = await zugangsbuch({ ohne: 2 });

		for (const name of [`${LAND} · 0`, `${TRAEGER} · 0`, `${OHNE} · 2`])
			expect(screen.getByRole('button', { name }), name).toBeTruthy();
		expect(tabellen(screen)).toEqual([`Zugangsbuch — ${OHNE}`]);
		expect(screen.queryByText(LEER)).toBeNull();
		expect(screen.getByRole('heading', { name: `${OHNE} · 2 Exemplare` })).toBeTruthy();
	});

	// Eine Zeile trägt alle Exemplare eines Titels von einem Tag: Feld und Überschrift zählen
	// Exemplare, nicht Zeilen.
	it('zählt Exemplare, auch wenn eine Zeile mehrere trägt', async () => {
		const klassensatz = {
			datum: '2026-09-20T00:00:00Z',
			titel: 'Mathebuch 7',
			anzahl: 28,
			barcodes: 'B-00100 – B-00127'
		};
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({
					von: '2026-09-16',
					bis: '2027-03-15',
					abschnitte: [
						{ topf: 'land', titel: LAND, exemplare: 28, zeilen: [klassensatz] },
						{ topf: 'schultraeger', titel: TRAEGER, exemplare: 0, zeilen: [] }
					]
				})
			})
		);
		const screen = render(Bestandsbuch, {
			pfad: 'zugangsbuch',
			buchname: 'Zugangsbuch',
			wortSingular: 'Zugang',
			spalten
		});

		expect(await screen.findByRole('button', { name: `${LAND} · 28` })).toBeTruthy();
		expect(screen.getByRole('heading', { name: `${LAND} · 28 Exemplare` })).toBeTruthy();
		expect(gezeichnet(screen, LAND)).toBe(1);
		// Testing Library liest den Text mit gewöhnlichen Leerzeichen; im Dokument sind sie geschützt.
		expect(screen.getByText(klassensatz.barcodes.replaceAll('\u00a0', ' '))).toBeTruthy();
	});

	// Der Browser bricht an einem Bindestrich um; ein Barcode stünde dann als „B-" und „00200" auf
	// zwei Zeilen. Jeder Barcode und jeder Bereich steht deshalb für sich und bricht nicht um.
	it('bricht einen Barcode nicht in sich um, nur zwischen zweien', async () => {
		const zeile = {
			datum: '2026-09-20T00:00:00Z',
			titel: 'Deutschbuch 7',
			anzahl: 5,
			barcodes: 'B-00200, B-00201, B-00300\u00a0–\u00a0B-00302'
		};
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({
					von: '2026-09-16',
					bis: '2027-03-15',
					abschnitte: [
						{ topf: 'land', titel: LAND, exemplare: 5, zeilen: [zeile] },
						{ topf: 'schultraeger', titel: TRAEGER, exemplare: 0, zeilen: [] }
					]
				})
			})
		);
		const screen = render(Bestandsbuch, {
			pfad: 'zugangsbuch',
			buchname: 'Zugangsbuch',
			wortSingular: 'Zugang',
			spalten
		});
		const tabelle = await screen.findByRole('table', { name: `Zugangsbuch — ${LAND}` });
		const zelle = /** @type {HTMLElement} */ (tabelle.querySelectorAll('tbody td')[3]);
		const teile = [...zelle.querySelectorAll('span.whitespace-nowrap')].map((t) => t.textContent);
		expect(teile).toEqual(['B-00200', 'B-00201', 'B-00300\u00a0–\u00a0B-00302']);
		expect(zelle.textContent?.trim()).toBe(zeile.barcodes);
	});

	it('zeigt nach einem Klick nur diesen Topf und nach dem zweiten wieder alle', async () => {
		const screen = await zugangsbuch({ land: 1, traeger: 2 });
		expect(tabellen(screen)).toEqual([`Zugangsbuch — ${LAND}`, `Zugangsbuch — ${TRAEGER}`]);

		const feld = screen.getByRole('button', { name: `${TRAEGER} · 2` });
		await fireEvent.click(feld);
		expect(tabellen(screen)).toEqual([`Zugangsbuch — ${TRAEGER}`]);
		expect(feld.getAttribute('aria-pressed')).toBe('true');

		await fireEvent.click(feld);
		expect(tabellen(screen)).toEqual([`Zugangsbuch — ${LAND}`, `Zugangsbuch — ${TRAEGER}`]);
	});

	it('sagt bei einem leeren Topf erst nach dem Klick darauf, dass nichts zuging', async () => {
		const screen = await zugangsbuch({ traeger: 1 });
		expect(screen.queryByText(LEER)).toBeNull();

		await fireEvent.click(screen.getByRole('button', { name: `${LAND} · 0` }));
		expect(tabellen(screen)).toEqual([]);
		expect(screen.getByRole('heading', { name: `${LAND} · 0 Exemplare` })).toBeTruthy();
		expect(screen.getByText(LEER)).toBeTruthy();
	});

	it('sagt es einmal, wenn in keinem Topf etwas steht', async () => {
		const screen = await zugangsbuch({});
		expect(tabellen(screen)).toEqual([]);
		expect(screen.getAllByText(LEER)).toHaveLength(1);
	});

	// Eine ausgeblendete Liste bleibt im Dokument: Zurückgeschaltet steht sie da, wie sie
	// verlassen wurde, mit den Zeilen, die „Mehr laden" dazugeholt hat.
	it('baut eine ausgeblendete Liste beim Zurückschalten nicht neu auf', async () => {
		const screen = await zugangsbuch({ land: 1, traeger: 60 });
		const vorher = screen.getByRole('table', { name: `Zugangsbuch — ${TRAEGER}` });
		await fireEvent.click(screen.getByRole('button', { name: 'Mehr laden (10 weitere Zeilen)' }));

		const feld = screen.getByRole('button', { name: `${LAND} · 1` });
		await fireEvent.click(feld);
		expect(tabellen(screen)).toEqual([`Zugangsbuch — ${LAND}`]);
		await fireEvent.click(feld);

		expect(screen.getByRole('table', { name: `Zugangsbuch — ${TRAEGER}` })).toBe(vorher);
		expect(gezeichnet(screen, TRAEGER)).toBe(60);
	});

	// Alle Zeilen zu zeichnen hielte die Seite bei Zehntausenden Zugängen Sekunden an. Die Zahl
	// in Feld und Überschrift bleibt die ganze.
	it('zeichnet je Topf zuerst 50 Zeilen und mit jedem „Mehr laden" 50 weitere', async () => {
		const screen = await zugangsbuch({ land: 3, traeger: 120 });
		expect(gezeichnet(screen, LAND)).toBe(3);
		expect(gezeichnet(screen, TRAEGER)).toBe(50);
		expect(screen.getByRole('button', { name: `${TRAEGER} · 120` })).toBeTruthy();
		expect(screen.getByRole('heading', { name: `${TRAEGER} · 120 Exemplare` })).toBeTruthy();
		expect(screen.getAllByRole('button', { name: /^Mehr laden/ })).toHaveLength(1);

		await fireEvent.click(screen.getByRole('button', { name: 'Mehr laden (70 weitere Zeilen)' }));
		expect(gezeichnet(screen, TRAEGER)).toBe(100);
		await fireEvent.click(screen.getByRole('button', { name: 'Mehr laden (20 weitere Zeilen)' }));
		expect(gezeichnet(screen, TRAEGER)).toBe(120);
		expect(screen.queryByRole('button', { name: /^Mehr laden/ })).toBeNull();
	});

	it('beginnt nach „Anzeigen" wieder mit den ersten 50 Zeilen', async () => {
		const screen = await zugangsbuch({ traeger: 120 });
		await fireEvent.click(screen.getByRole('button', { name: 'Mehr laden (70 weitere Zeilen)' }));
		expect(gezeichnet(screen, TRAEGER)).toBe(100);

		await fireEvent.click(screen.getByRole('button', { name: 'Anzeigen' }));
		await screen.findByRole('button', { name: 'Mehr laden (70 weitere Zeilen)' });
		expect(gezeichnet(screen, TRAEGER)).toBe(50);
	});

	it('schreibt große Zahlen mit Tausenderpunkt, im Feld, in der Überschrift und im Knopf', async () => {
		const screen = await zugangsbuch({ ohne: 1050 });
		expect(screen.getByRole('button', { name: `${OHNE} · 1.050` })).toBeTruthy();
		expect(screen.getByRole('heading', { name: `${OHNE} · 1.050 Exemplare` })).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Mehr laden (1.000 weitere Zeilen)' })).toBeTruthy();
	});

	// Der Satz erklärt die Liste „ohne Zuordnung“; ohne die Liste erklärt er nichts.
	it('erklärt „ohne Zuordnung“ nur, solange diese Liste zu sehen ist', async () => {
		const screen = await zugangsbuch({ land: 1, ohne: 1 });
		expect(screen.getByText(/keine Bestellung hinterlegt/)).toBeTruthy();

		await fireEvent.click(screen.getByRole('button', { name: `${LAND} · 1` }));
		expect(screen.queryByText(/keine Bestellung hinterlegt/)).toBeNull();
	});

	// Ein neuer Zeitraum kann die Liste „ohne Zuordnung“ entfallen lassen. Dann gilt die Wahl
	// nicht mehr, und die Seite zeigt wieder alle Töpfe mit Einträgen.
	it('zeigt wieder alle Töpfe, wenn der gewählte im neuen Zeitraum nicht vorkommt', async () => {
		const screen = await zugangsbuch({ land: 1, ohne: 1 });
		await fireEvent.click(screen.getByRole('button', { name: `${OHNE} · 1` }));
		expect(tabellen(screen)).toEqual([`Zugangsbuch — ${OHNE}`]);

		vi.mocked(apiFetch).mockResolvedValue(antwort({ land: 3 }));
		await fireEvent.click(screen.getByRole('button', { name: 'Anzeigen' }));
		await screen.findByRole('button', { name: `${LAND} · 3` });
		expect(tabellen(screen)).toEqual([`Zugangsbuch — ${LAND}`]);
	});
});

// Einen Zeitraum über ein Jahr lehnt der Server mit einem Satz ab. Die Seite zeigt ihn, und der
// Ausdruck bleibt beim zuletzt geladenen Zeitraum.
describe('Bestandsbuch: Zeitraum über ein Jahr', () => {
	it('zeigt den Satz des Servers und druckt weiter den geladenen Zeitraum', async () => {
		const screen = await zugangsbuch({ land: 1 });
		const satz = 'Der Zeitraum darf höchstens ein Jahr umfassen';
		vi.mocked(apiFetch).mockResolvedValue(serverAntwort(400, { error: satz }));

		await fireEvent.input(screen.getByLabelText('Von'), { target: { value: '2025-09-16' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Anzeigen' }));

		expect(await screen.findByText(satz)).toBeTruthy();
		expect(apiFetch).toHaveBeenLastCalledWith(
			'/api/bestand/zugangsbuch?von=2025-09-16&bis=2027-03-15'
		);
		expect(screen.getByRole('link', { name: 'Ausdrucken' }).getAttribute('href')).toBe(
			'/api/bestand/zugangsbuch/pdf?von=2026-09-16&bis=2027-03-15'
		);
	});
});
