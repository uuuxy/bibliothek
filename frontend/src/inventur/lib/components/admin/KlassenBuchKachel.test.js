import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import KlassenBuchKachel from './KlassenBuchKachel.svelte';

// Die Kachel eines Klassensatzes mit gemischten Auflagen (docs/OFFEN.md 4.18, Stufe 5):
// GetClassGroups liefert `auflagen` nur, wenn Kinder der Klasse eine andere Auflage als die der
// Kachel haben (inventur/datenbank_klassen.go) — dann steht die Aufschlüsselung auf der Kachel.
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
			book: buch({
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
		const { queryByTestId } = render(KlassenBuchKachel, { book: buch(), bearbeitbar: false });
		expect(queryByTestId('klassensatz-auflagen')).toBeNull();
	});
});
