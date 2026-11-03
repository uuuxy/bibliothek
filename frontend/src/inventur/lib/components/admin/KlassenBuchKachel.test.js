import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import KlassenBuchKachel from './KlassenBuchKachel.svelte';
import BuchKarte, { KARTE } from '../BuchKarte.svelte';

// Die Kachel eines Klassensatzes mit gemischten Auflagen (docs/OFFEN.md 4.18, Stufe 5):
// GetClassGroups liefert `auflagen` nur, wenn Kinder der Klasse eine andere Auflage als die der
// Kachel haben (repository/klassensaetze.go) — dann steht die Aufschlüsselung auf der Kachel.
const buch = (/** @type {any} */ extra = {}) => ({
	id: 'neu',
	isbn: '',
	title: 'Mathe 7',
	subject: 'Mathematik',
	coverUrl: '',
	verfuegbar: 2,
	gesamt: 30,
	quelle: 'ausleihe',
	leser: 7,
	...extra
});

describe('KlassenBuchKachel: Auflagen in der Klasse', () => {
	it('zeigt die Auflagen der Klasse, die meisten Kinder zuerst, die der Kachel markiert', () => {
		const { getByTestId } = render(KlassenBuchKachel, {
			titel: buch({
				auflagen: [
					{ id: 'neu', auflage: '4. Aufl.', erscheinungsjahr: 2023, kinder: 4 },
					{ id: 'alt', auflage: '3. Aufl.', erscheinungsjahr: 2019, kinder: 3 }
				]
			}),
			bearbeitbar: false
		});
		const block = getByTestId('klassensatz-auflagen');
		expect([...block.querySelectorAll('li')].map((li) => li.textContent)).toEqual([
			'4. Aufl. · 2023: 4 Kinder — diese Auflage',
			'3. Aufl. · 2019: 3 Kinder'
		]);
		expect(block.querySelector('p')?.textContent).toBe('Auflagen in der Klasse');
	});

	it('zeigt nichts davon ohne Aufschlüsselung', () => {
		const { queryByTestId } = render(KlassenBuchKachel, { titel: buch(), bearbeitbar: false });
		expect(queryByTestId('klassensatz-auflagen')).toBeNull();
	});
});

// Portal „Schulbücher" (docs/OFFEN.md 4.18, Stufe 6): Eine Zeile für ein Buch in mehreren
// Auflagen trägt die Summe und auflagenBestand (inventur.LernmittelTitel) — die Kachel sagt,
// woraus die Summe besteht. Das Feld heißt anders als `auflagen` (Kinder je Auflage im
// Klassensatz), damit die beiden Aufschlüsselungen sich nicht vermengen.
describe('KlassenBuchKachel: Bestand aus mehreren Auflagen', () => {
	it('zeigt unter der Summe, woraus sie besteht', () => {
		const { getByTestId, queryByTestId } = render(KlassenBuchKachel, {
			titel: buch({
				quelle: undefined,
				verfuegbar: 46,
				gesamt: 58,
				auflagenBestand: [
					{ auflage: '4. Aufl.', erscheinungsjahr: 2023, gesamt_bestand: 16 },
					{ auflage: '3. Aufl.', erscheinungsjahr: 2019, gesamt_bestand: 42 }
				]
			}),
			bearbeitbar: false
		});
		expect(getByTestId('schulbuch-auflagen').textContent).toBe(
			'Bestand aus 2 Auflagen: 4. Aufl. · 2023 (16), 3. Aufl. · 2019 (42)'
		);
		expect(queryByTestId('klassensatz-auflagen')).toBeNull();
	});
});

// Die Klassenkachel ist gebaut wie die Katalog-Kachel: derselbe Rahmen aus einer Quelle (KARTE)
// und der Bestand als Chip auf dem Cover. Titel und Hinweise stehen weiter sichtbar darunter,
// weil es am Tablet kein Zeigen gibt.
describe('KlassenBuchKachel: gebaut wie die Katalog-Kachel', () => {
	it('Katalog- und Klassenkachel tragen denselben Rahmen, bearbeitbar oder nicht', () => {
		const katalog = render(BuchKarte, { titel: { ...buch(), author: '' } });
		expect(katalog.container.firstElementChild?.className).toContain(KARTE);
		katalog.unmount();
		for (const bearbeitbar of [true, false]) {
			const klasse = render(KlassenBuchKachel, { titel: buch(), bearbeitbar });
			expect(klasse.container.firstElementChild?.className).toContain(KARTE);
			klasse.unmount();
		}
	});

	it('zeigt den Bestand als Chip auf dem Cover und liest ihn als ganzen Satz vor', () => {
		const { getByText, getAllByText, getByRole } = render(KlassenBuchKachel, {
			titel: buch(),
			bearbeitbar: false
		});
		const chip = getByText('2 / 30').closest('[data-chip]');
		expect(chip).not.toBeNull();
		// Der Satz steht nur noch für Hilfstechnik da, nicht mehr als Zeile unter dem Titel.
		const saetze = getAllByText('2 von 30 verfügbar');
		expect(saetze).toHaveLength(1);
		expect(saetze[0].classList.contains('sr-only')).toBe(true);
		// Der Chip gehört zum Cover und steht deshalb vor dem Titel darunter. Ohne Bild trägt
		// auch der Platzhalter den Titel, darum die Überschrift.
		const ueberschrift = getByRole('heading', { level: 3, name: 'Mathe 7' });
		const position = chip?.compareDocumentPosition(ueberschrift) ?? 0;
		expect(position & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
	});

	it('zeigt ohne Zahlen im Payload keinen Chip', () => {
		const { container } = render(KlassenBuchKachel, {
			titel: buch({ gesamt: undefined, verfuegbar: undefined }),
			bearbeitbar: false
		});
		expect(container.querySelector('[data-chip]')).toBeNull();
	});
});
