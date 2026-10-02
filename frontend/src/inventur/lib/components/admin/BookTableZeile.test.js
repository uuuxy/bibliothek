import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import BookTableZeile from './BookTableZeile.svelte';

/** @param {any} [zusatz] */
const buch = (zusatz = {}) => ({
	id: 't-1',
	isbn: '',
	title: 'Die Räuber',
	author: 'Friedrich Schiller',
	subject: 'Deutsch',
	gradeLevel: 9,
	istLernmittel: true,
	stock: 12,
	verfuegbar: 10,
	gesamt: 12,
	coverUrl: '',
	lastCounted: '2026-09-01T10:00:00Z',
	erweiterteEigenschaften: { standort: 'Raum 12' },
	...zusatz
});

/** @param {any} book @param {boolean} [isSelected] */
function zeile(book, isSelected = false) {
	const nichts = () => {};
	return render(BookTableZeile, {
		book,
		index: 0,
		dragOverIndex: null,
		isSelected,
		onOpenDetail: nichts,
		onToggleSelect: nichts,
		onDragStart: nichts,
		onDragOver: nichts,
		onDragLeave: nichts,
		onDrop: nichts,
		onDragEnd: nichts
	});
}

/** Der Text aller Schildchen der Zeile (ui/StatusChip trägt data-chip). */
const schildchen = (/** @type {HTMLElement} */ container) =>
	[...container.querySelectorAll('[data-chip]')].map((c) => (c.textContent ?? '').trim());

// Die Zeile der Titel-Verwaltung: Was man nur liest, steht als Text, auch der Bestand. Ein
// Schildchen (ui/StatusChip) trägt nur die Art „Lernmittel“, wie in der Leserakte.
describe('Titel-Verwaltung: Zeile', () => {
	it('zeigt Fach, Standort, Prüfdatum und Bestand als Text und die Art als Schildchen', () => {
		const { container } = zeile(buch());
		expect(schildchen(container)).toEqual(['Lernmittel']);
		const text = container.textContent ?? '';
		for (const wort of [
			'Die Räuber',
			'Friedrich Schiller',
			'Deutsch',
			'Kl. 9',
			'Raum 12',
			'1.9.2026',
			'12'
		])
			expect(text, wort).toContain(wort);
	});

	it('nennt ein Buch der Bücherei ohne Schildchen und setzt einen Strich, wo nichts steht', () => {
		const { container } = zeile(
			buch({
				istLernmittel: false,
				subject: '',
				gradeLevel: 0,
				lastCounted: '',
				erweiterteEigenschaften: {}
			})
		);
		expect(schildchen(container)).toEqual([]);
		expect(container.textContent ?? '').toContain('Bibliothek');
		expect((container.textContent ?? '').match(/–/g) ?? []).toHaveLength(4);
	});

	// Unter fünf Exemplaren stand die Zahl rot hinterlegt. Das traf fast jeden Titel der
	// Bücherei, und Rot heißt in Material 3 Fehler.
	it('färbt einen kleinen Bestand nicht', () => {
		const { container } = zeile(buch({ gesamt: 1, istLernmittel: false }));
		const bestand = container.querySelectorAll('td')[8];
		expect(bestand?.textContent?.trim()).toBe('1');
		expect(bestand?.children.length, 'die Zahl steht ohne eigenes Element in der Zelle').toBe(0);
	});

	// Die gewählte Zeile trägt die Fläche der Auswahl wie jede Liste der Anwendung
	// (ui/Tabelle: aria-selected).
	it('kennzeichnet die gewählte Zeile', () => {
		expect(zeile(buch(), true).container.querySelector('tr')?.getAttribute('aria-selected')).toBe(
			'true'
		);
		expect(zeile(buch(), false).container.querySelector('tr')?.getAttribute('aria-selected')).toBe(
			'false'
		);
	});

	// Der Titel steht in der Nachbarzelle; das Cover sagt ihn kein zweites Mal an.
	it('zeigt ohne Cover die Initiale des Titels', () => {
		const { container } = zeile(buch());
		expect(container.querySelector('img')).toBeNull();
		expect(container.querySelectorAll('td')[1]?.textContent?.trim()).toBe('D');
	});
});
