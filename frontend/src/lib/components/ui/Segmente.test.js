import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import Segmente from './Segmente.svelte';

/**
 * Hintergrund (04.09.2026): Im Druck-Center stand ein handgebauter Umschalter aus einer
 * anderen Designsprache — 38 px hoch statt 40, Ecken 12 px statt voll gerundet, und das
 * gewählte Segment lag auf `inverse-surface` (fast schwarz). Damit widersprach er der
 * Regel, die seit dem 04.08.2026 in styles/rollen.css steht: „In M3 markiert NICHT die
 * Primärfarbe eine Auswahl, sondern der secondary-container."
 *
 * Diese Tests halten die Bauform fest — die Farbrolle, weil sie die Regel ist, und das
 * Häkchen, weil es bei getönten Flächen die einzige eindeutige Auskunft darüber ist,
 * welches Segment gewählt ist.
 */
const optionen = [
	{ wert: 'offen', text: 'Offen' },
	{ wert: 'erledigt', text: 'Erledigt' },
	{ wert: 'alle', text: 'Alle' }
];

const aufbau = (wert) =>
	render(Segmente, { optionen, wert, onwahl: () => {}, etikett: 'Welche Exemplare' });

describe('Segmente', () => {
	it('markiert die Auswahl mit secondary-container, nicht mit der Primärfarbe', () => {
		const { getByRole } = aufbau('erledigt');
		const klassen = (getByRole('button', { name: /Erledigt/ }).getAttribute('class') || '').split(
			/\s+/
		);
		expect(klassen).toContain('bg-secondary-container');
		expect(klassen).toContain('text-on-secondary-container');
		expect(klassen).not.toContain('bg-primary');
	});

	it('meldet die Auswahl auch ohne Farbe — über aria-pressed', () => {
		const { getByRole } = aufbau('alle');
		expect(getByRole('button', { name: /Alle/ }).getAttribute('aria-pressed')).toBe('true');
		expect(getByRole('button', { name: /Offen/ }).getAttribute('aria-pressed')).toBe('false');
	});

	it('hält den Platz des Häkchens in jedem Segment frei, damit nichts springt', () => {
		const { container } = aufbau('offen');
		// Ein Icon je Segment — die unbenutzten sind unsichtbar, aber im Fluss.
		const haken = container.querySelectorAll('svg');
		expect(haken.length).toBe(optionen.length);
		const unsichtbar = [...haken].filter((s) => s.getAttribute('class')?.includes('invisible'));
		expect(unsichtbar.length).toBe(optionen.length - 1);
	});

	it('ist 40 dp hoch und voll gerundet — die M3-Maße des Segmented Button', () => {
		const { getByRole } = aufbau('offen');
		const klassen = (getByRole('group').getAttribute('class') || '').split(/\s+/);
		expect(klassen).toContain('h-10');
		expect(klassen).toContain('rounded-full');
	});
});
