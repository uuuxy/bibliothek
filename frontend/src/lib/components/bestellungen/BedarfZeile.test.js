import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import BedarfZeile from './BedarfZeile.svelte';
import OrderRecommendations from './OrderRecommendations.svelte';

// Der Bestellbedarf zählt am Buch (docs/OFFEN.md 4.18, Stufe 3): Gehört ein Titel zu einem
// Buch mit mehreren Auflagen, ist die Zeile das Buch — Titel und ISBN der neuesten Auflage,
// die Summe als Zahl, darunter die Auflagen einzeln (GET /api/bestellungen, api/reorders.go).

const buch = {
	id: 'neu',
	titel: 'Lambacher Schweizer 7',
	isbn: '9783120000026',
	verlag: 'Klett',
	verfuegbarer_bestand: 3,
	gesamt_bestand: 3,
	auflagen: [
		{
			id: 'neu',
			auflage: '4. Aufl.',
			isbn: '9783120000026',
			erscheinungsjahr: 2023,
			verfuegbarer_bestand: 1,
			gesamt_bestand: 1
		},
		{
			id: 'alt',
			auflage: '3. Aufl.',
			isbn: '9783120000019',
			erscheinungsjahr: 2019,
			verfuegbarer_bestand: 2,
			gesamt_bestand: 2
		}
	]
};
const einzeln = {
	id: 'p8',
	titel: 'Physik 8',
	isbn: '9783120000033',
	verfuegbarer_bestand: 1,
	gesamt_bestand: 1
};

describe('BedarfZeile', () => {
	it('schlüsselt ein Buch mit mehreren Auflagen auf', () => {
		const screen = render(BedarfZeile, { r: buch, onAddToCart: vi.fn() });
		expect(
			screen.getByText('Bestand aus 2 Auflagen: 4. Aufl. · 2023 (1), 3. Aufl. · 2019 (2)')
		).toBeTruthy();
	});

	it('bleibt bei einem Titel ohne andere Auflage, wie sie war', () => {
		const screen = render(BedarfZeile, { r: einzeln, onAddToCart: vi.fn() });
		expect(screen.queryByText(/Bestand aus/)).toBeNull();
	});

	it('legt die Zeile des Buchs in den Warenkorb — die neueste Auflage', async () => {
		const onAddToCart = vi.fn();
		const screen = render(BedarfZeile, { r: buch, onAddToCart });
		await fireEvent.click(
			screen.getByRole('button', { name: 'Lambacher Schweizer 7 zur Bestellung hinzufügen' })
		);
		expect(onAddToCart).toHaveBeenCalledWith(buch);
	});
});

describe('OrderRecommendations: Filter', () => {
	it('findet das Buch auch über die ISBN der alten Auflage', async () => {
		const screen = render(OrderRecommendations, {
			recommendations: [buch, einzeln],
			onAddToCart: vi.fn()
		});
		await fireEvent.input(screen.getByLabelText('Bestellvorschläge filtern'), {
			target: { value: '9783120000019' }
		});
		expect(screen.getByText('Lambacher Schweizer 7')).toBeTruthy();
		expect(screen.queryByText('Physik 8')).toBeNull();
	});
});
