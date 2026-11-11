import { describe, it, expect, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import Eigenschaften from './Eigenschaften.svelte';
import CanvasFlaeche from './CanvasFlaeche.svelte';
import { idStore, resetDesign, fuegeTextHinzu } from './idDesignerStore.svelte.js';
import { elementName } from './elementName.js';

// Ein ausgeblendetes Element zeichnet die Leinwand nicht, und gewählt wird nur durch Anklicken
// auf der Leinwand. War die Auswahl einmal weg, gab es keinen Weg zurück außer „Standardwerte
// wiederherstellen", das alle Anpassungen verwirft. Ohne Auswahl zeigt das Eigenschaften-Feld
// deshalb die Elemente der Seite mit ihrem Schalter.
describe('Ausweis-Designer: ausgeblendetes Element', () => {
	beforeEach(() => resetDesign());

	const adresse = () => idStore.front.elements.find((e) => e.id === 'address');

	it('lässt sich ohne Auswahl wieder einblenden und steht danach auf der Leinwand', async () => {
		const el = adresse();
		if (!el) throw new Error('Die Vorderseite hat keine Adresse.');
		el.show = false;
		const leinwand = render(CanvasFlaeche, {
			props: {
				side: 'front',
				gewaehlteKennung: null,
				onSelect: () => {},
				leser: { id: 's-1', vorname: 'Max', nachname: 'Mustermann', barcode_id: 'DEMO-S-001' },
				zoom: 100,
				barcodeType: 'code39'
			}
		});
		const elementeVorher = leinwand.container.querySelectorAll(
			'[data-ausweis-vorschau] > div > *'
		).length;

		const feld = render(Eigenschaften, { gewaehlteKennung: null, side: 'front' });
		const schalter = feld.getByLabelText('Adresse');
		expect(schalter.getAttribute('aria-checked')).toBe('false');

		await fireEvent.click(schalter);

		expect(adresse()?.show).toBe(true);
		expect(schalter.getAttribute('aria-checked')).toBe('true');
		expect(leinwand.container.querySelectorAll('[data-ausweis-vorschau] > div > *').length).toBe(
			elementeVorher + 1
		);
	});

	it('nennt jedes Element der Seite beim Namen, nicht bei der Kennung', () => {
		fuegeTextHinzu('front');
		const feld = render(Eigenschaften, { gewaehlteKennung: null, side: 'front' });
		const liste = feld.getByTestId('designer-elemente');
		expect(liste.querySelectorAll('[role="switch"]').length).toBe(idStore.front.elements.length);
		for (const name of ['Schulname', 'Adresse', 'Logo', 'Passbild', 'Name', 'Strichcode']) {
			expect(feld.getByLabelText(name), name).toBeTruthy();
		}
		expect(liste.textContent).not.toMatch(/\bheader\b|\bel-\d+/);
	});

	it('zeigt im Kopf des gewählten Elements denselben Namen', () => {
		const feld = render(Eigenschaften, { gewaehlteKennung: 'address', side: 'front' });
		expect(feld.getByRole('heading', { name: 'Adresse' })).toBeTruthy();
	});

	it('bietet ohne das Recht zum Ändern keine Schalter an', () => {
		const feld = render(Eigenschaften, { gewaehlteKennung: null, side: 'front', gesperrt: true });
		expect(feld.queryByTestId('designer-elemente')).toBeNull();
		expect(feld.getByText(/Das Design ändern kann/)).toBeTruthy();
	});
});

describe('elementName', () => {
	it('nennt einen freien Text nach seinem Inhalt und sonst nach der Art', () => {
		expect(elementName({ type: 'text', content: ' Gefördert vom Verein ' })).toBe(
			'Gefördert vom Verein'
		);
		expect(elementName({ type: 'text', content: '' })).toBe('Text');
		expect(elementName({ type: 'box' })).toBe('Fläche');
		expect(elementName({ type: 'unbekannt' })).toBe('Element');
	});
});
