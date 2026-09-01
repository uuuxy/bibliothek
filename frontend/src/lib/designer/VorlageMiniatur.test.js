import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import VorlageMiniatur from './VorlageMiniatur.svelte';
import { AUSWEIS_VORLAGEN, vorlage } from './ausweisVorlagen.js';

// Die Galerie soll zeigen, was man bekäme: JEDES Element der Vorlagen-Vorderseite
// hat in der Miniatur einen Platz — sonst zeigt die Vorschau ein anderes Layout als
// das, was der Klick anwendet.
describe('VorlageMiniatur', () => {
	it.each(AUSWEIS_VORLAGEN.map((v) => v.value))(
		'%s: zeichnet jedes Element der Vorderseite',
		(kennung) => {
			const { container } = render(VorlageMiniatur, { props: { kennung } });
			const gezeichnet = container.querySelectorAll('[data-miniatur-element]').length;
			expect(gezeichnet).toBe(vorlage(kennung)?.front.elements.length);
		}
	);

	it('zeichnet Farbflächen in ihrer Farbe', () => {
		const { container } = render(VorlageMiniatur, { props: { kennung: 'schwarz-gruen' } });
		const styles = Array.from(container.querySelectorAll('[style*="background-color"]')).map((e) =>
			e.getAttribute('style')
		);
		// #76b82a = rgb(118, 184, 42) — die Grünlinie
		expect(styles.some((s) => s?.includes('rgb(118, 184, 42)'))).toBe(true);
	});

	it('bleibt bei unbekannter Kennung leer statt zu werfen', () => {
		const { container } = render(VorlageMiniatur, { props: { kennung: 'gibt-es-nicht' } });
		expect(container.querySelectorAll('[data-miniatur-element]').length).toBe(0);
	});
});
