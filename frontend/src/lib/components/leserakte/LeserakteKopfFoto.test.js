import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import LeserakteKopf from './LeserakteKopf.svelte';

// Nach einem neuen Foto lädt die Akte desselben Lesers neu, ohne den Kopf neu aufzubauen
// (useLeserakte: derselbe Leser bleibt stehen). Ein Passbild, das vorher nicht lud, darf das
// neue nicht verdecken: Der Fehler gilt nur für die Adresse, die nicht lud.
describe('LeserakteKopf: Passbild', () => {
	const props = (/** @type {number} */ timestamp) => ({
		leser: { id: 'A', vorname: 'Anna', nachname: 'Beispiel', foto_url: '/api/fotos/A' },
		timestamp,
		showWebcam: false,
		onDeselect: vi.fn()
	});

	it('zeigt nach einem Fehler das neue Foto mit neuem Stand', async () => {
		const { getByAltText, queryByAltText, rerender } = render(LeserakteKopf, { props: props(1) });
		await fireEvent.error(getByAltText('Passbild'));
		expect(queryByAltText('Passbild'), 'das Bild, das nicht lud, bleibt verborgen').toBeNull();

		await rerender(props(2));
		const bild = getByAltText('Passbild');
		expect(bild.getAttribute('src')).toBe('/api/fotos/A?t=2');
	});
});
