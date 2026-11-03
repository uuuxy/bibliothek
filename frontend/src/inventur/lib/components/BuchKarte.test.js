import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import BuchKarte from './BuchKarte.svelte';

// Zwei Kommentare widersprachen sich (OFFEN.md 5.9): BuchKarte.svelte sagte, der Stift
// öffne trotz stopPropagation auch die Akte, weil der Flächen-Handler „als direkter
// Listener vor Sveltes Delegation feuert"; e2e/cover-aendern.spec.js sagte, der Stift
// öffne die Titel-Verwaltung, nicht die Akte. Gemessen am gebauten Bauteil: Welche der
// beiden Rückrufe feuern bei einem Klick auf den Stift?
const buch = {
	id: 'b-1',
	isbn: '9783123456789',
	title: 'Mathematik Neue Wege 9',
	author: 'Lergenmüller',
	subject: 'Mathematik',
	coverUrl: '',
	gesamt: 28,
	verfuegbar: 27
};

describe('BuchKarte — wohin die Klicks gehen', () => {
	it('der Stift öffnet NUR die Bearbeitung, nicht auch die Akte', async () => {
		const akte = vi.fn();
		const bearbeiten = vi.fn();
		const { getByRole } = render(BuchKarte, {
			titel: buch,
			onclick: akte,
			onEditClick: bearbeiten
		});

		await fireEvent.click(getByRole('button', { name: 'Buch schnell bearbeiten' }));

		expect(bearbeiten).toHaveBeenCalledTimes(1);
		expect(akte).not.toHaveBeenCalled();
	});

	it('der Titel öffnet die Akte über die Fläche — genau einmal', async () => {
		const akte = vi.fn();
		const { getByRole } = render(BuchKarte, { titel: buch, onclick: akte });

		await fireEvent.click(getByRole('button', { name: buch.title }));

		expect(akte).toHaveBeenCalledTimes(1);
	});
});

// Ein Buch in mehreren Auflagen (docs/OFFEN.md 4.18, Stufe 6): Die Kachel zählt die Summe
// (book.buch, aus buecherJeBuch) und sagt darunter, woraus sie besteht.
describe('BuchKarte — ein Buch in mehreren Auflagen', () => {
	it('zeigt die Summe und die Aufschlüsselung', () => {
		const { getByText } = render(BuchKarte, {
			titel: {
				...buch,
				auflage: '3. Aufl.',
				buch: {
					gesamt: 46,
					verfuegbar: 41,
					imZulauf: 0,
					auflagen: [
						{ id: 'neu', auflage: '4. Aufl.', erscheinungsjahr: 2023, gesamt_bestand: 4 },
						{ id: 'b-1', auflage: '3. Aufl.', erscheinungsjahr: 2019, gesamt_bestand: 42 }
					]
				}
			}
		});
		expect(getByText('41 von 46 verfügbar')).toBeTruthy();
		expect(
			getByText('Bestand aus 2 Auflagen: 4. Aufl. · 2023 (4), 3. Aufl. · 2019 (42)')
		).toBeTruthy();
	});

	it('ohne weitere Auflagen bleibt es bei den eigenen Zahlen', () => {
		const { getByText, queryByText } = render(BuchKarte, { titel: buch });
		expect(getByText('27 von 28 verfügbar')).toBeTruthy();
		expect(queryByText(/Bestand aus/)).toBeNull();
	});
});

// Unter dem Cover stehen zwei Zeilen fester Höhe: der ISBN-Balken und die Fußzeile. Beide
// sind auch ohne ISBN und ohne Prüfdatum gleich hoch, damit das Raster nicht verrutscht.
describe('BuchKarte — ISBN-Balken, Prüfdatum und Bestand', () => {
	it('der ISBN-Balken kopiert, ohne die Akte zu öffnen', async () => {
		const akte = vi.fn();
		const { getByRole } = render(BuchKarte, { titel: buch, onclick: akte });

		await fireEvent.click(getByRole('button', { name: 'ISBN 9783123456789 kopieren' }));

		expect(akte).not.toHaveBeenCalled();
	});

	it('ohne ISBN steht „Keine ISBN" an seiner Stelle, ohne Knopf', () => {
		const { getByText, queryByRole } = render(BuchKarte, { titel: { ...buch, isbn: '' } });
		expect(getByText('Keine ISBN')).toBeTruthy();
		expect(queryByRole('button', { name: /kopieren/ })).toBeNull();
	});

	it('das Prüfdatum steht mit Häkchen; ohne Datum steht dort nichts', () => {
		const mit = render(BuchKarte, { titel: { ...buch, lastCounted: '2026-03-26' } });
		expect(mit.getByText('26.03.2026')).toBeTruthy();
		mit.unmount();

		const ohne = render(BuchKarte, { titel: buch });
		expect(ohne.queryByText(/geprüft/i)).toBeNull();
	});

	it('der Bestand steht kurz im Chip und als Satz für Hilfstechnik', () => {
		const { getByText } = render(BuchKarte, { titel: buch });
		expect(getByText('27 / 28')).toBeTruthy();
		expect(getByText('27 von 28 verfügbar')).toBeTruthy();
	});
});
