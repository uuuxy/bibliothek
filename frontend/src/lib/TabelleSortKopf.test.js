import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import TabelleSortKopf from './components/ui/TabelleSortKopf.svelte';

/**
 * Der sortierbare Spaltenkopf (Protokoll des Medienzentrums, Punkt 5).
 *
 * Geprüft wird, was ein Screenreader bekommt und was der Titel verspricht — beides
 * lässt sich am Bildschirm nicht ansehen und geht deshalb still kaputt.
 */
const AUS = { spalte: '', absteigend: false };

describe('TabelleSortKopf', () => {
	it('trägt aria-sort an der ZELLE, nicht am Knopf', () => {
		const screen = render(TabelleSortKopf, {
			props: { spalte: 'name', text: 'Name', sortierung: AUS, onsortiere: () => {} }
		});
		const zelle = screen.container.querySelector('th');
		expect(zelle?.getAttribute('aria-sort')).toBe('none');
		expect(screen.getByRole('button').hasAttribute('aria-sort')).toBe(false);
	});

	it('meldet die Richtung der aktiven Spalte', () => {
		const auf = render(TabelleSortKopf, {
			props: {
				spalte: 'name',
				text: 'Name',
				sortierung: { spalte: 'name', absteigend: false },
				onsortiere: () => {}
			}
		});
		expect(auf.container.querySelector('th')?.getAttribute('aria-sort')).toBe('ascending');

		const ab = render(TabelleSortKopf, {
			props: {
				spalte: 'name',
				text: 'Name',
				sortierung: { spalte: 'name', absteigend: true },
				onsortiere: () => {}
			}
		});
		expect(ab.container.querySelector('th')?.getAttribute('aria-sort')).toBe('descending');
	});

	it('nennt im Titel, was der nächste Klick TUT — nicht was gilt', () => {
		const screen = render(TabelleSortKopf, {
			props: {
				spalte: 'klasse',
				text: 'Klasse',
				sortierung: { spalte: 'klasse', absteigend: false },
				onsortiere: () => {}
			}
		});
		// Aufsteigend sortiert → der Klick macht absteigend.
		expect(screen.getByRole('button').getAttribute('title')).toBe(
			'Nach Klasse absteigend sortieren'
		);
	});

	it('meldet den Klick mit der Spalte', async () => {
		const onsortiere = vi.fn();
		const screen = render(TabelleSortKopf, {
			props: { spalte: 'ausgeliehen', text: 'Geliehene Bücher', sortierung: AUS, onsortiere }
		});
		await fireEvent.click(screen.getByRole('button'));
		expect(onsortiere).toHaveBeenCalledWith('ausgeliehen');
	});

	it('ist ohne onsortiere ein gewöhnlicher Kopf — kein Knopf, kein Pfeil', () => {
		const screen = render(TabelleSortKopf, {
			props: { spalte: 'name', text: 'Name', sortierung: AUS }
		});
		expect(screen.queryByRole('button')).toBeNull();
		expect(screen.container.querySelector('th')?.hasAttribute('aria-sort')).toBe(false);
		expect(screen.container.textContent?.trim()).toBe('Name');
	});
});
