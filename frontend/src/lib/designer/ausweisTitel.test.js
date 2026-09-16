import { describe, it, expect, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import CardFace from './CardFace.svelte';
import { idStore, defaultFrontElements } from './idDesignerStore.svelte.js';
import { heileAltBaender } from './idDesignAltbestand.js';

/**
 * Die Aufschrift des Ausweises kommt aus der Art des Lesers.
 *
 * Gemeldet am 16.09.2026 : „wenn ein Lehrer einen Ausweis drucken möchte, sollte
 * da natürlich Lehrerausweis stehen anstatt schülerausweis". Der Titel war bis dahin ein
 * gewöhnliches Textelement mit fest eingetipptem Inhalt — die Karte einer Lehrkraft trug
 * deshalb das Wort „Schülerausweis".
 *
 * Gemessen wird an der gerenderten Karte, nicht an der Hilfsfunktion: Der Weg von der
 * Art bis auf das Papier führt über CardFace, und genau der war unterbrochen.
 */

/** @param {string} art */
function leser(art) {
	return {
		id: 'l-1',
		vorname: 'Anna',
		nachname: 'Beispiel',
		barcode_id: 'A-000123',
		art,
		ausweis_gueltig_bis: 2027
	};
}

beforeEach(() => {
	idStore.front.elements = defaultFrontElements();
});

describe('Aufschrift des Ausweises', () => {
	it('nennt den Ausweis einer Lehrkraft „Lehrerausweis"', () => {
		const { container } = render(CardFace, {
			props: { side: 'front', student: leser('lehrkraft'), barcodeType: 'code39' }
		});
		expect(container.textContent).toContain('Lehrerausweis');
		expect(container.textContent).not.toContain('Schülerausweis');
	});

	it('nennt den Ausweis einer LiV ebenfalls „Lehrerausweis"', () => {
		const { container } = render(CardFace, {
			props: { side: 'front', student: leser('liv'), barcodeType: 'code39' }
		});
		expect(container.textContent).toContain('Lehrerausweis');
	});

	it('bleibt beim Schüler „Schülerausweis"', () => {
		const { container } = render(CardFace, {
			props: { side: 'front', student: leser('schueler'), barcodeType: 'code39' }
		});
		expect(container.textContent).toContain('Schülerausweis');
		expect(container.textContent).not.toContain('Lehrerausweis');
	});

	it('nennt eine Zeile ohne Art „Schülerausweis" — das ist die Vorgabe der Spalte', () => {
		const ohneArt = { ...leser('schueler'), art: undefined };
		const { container } = render(CardFace, {
			props: { side: 'front', student: ohneArt, barcodeType: 'code39' }
		});
		expect(container.textContent).toContain('Schülerausweis');
	});
});

describe('Gültigkeit auf der Karte', () => {
	it('steht beim Schüler', () => {
		const { container } = render(CardFace, {
			props: { side: 'front', student: leser('schueler'), barcodeType: 'code39' }
		});
		expect(container.textContent).toContain('Gültig bis: 31.07.2027');
	});

	it('fehlt beim Kollegen — sein Ausweis läuft mit keinem Schuljahr ab', () => {
		const { container } = render(CardFace, {
			props: {
				side: 'front',
				student: { ...leser('lehrkraft'), ausweis_gueltig_bis: null },
				barcodeType: 'code39'
			}
		});
		expect(container.textContent).not.toContain('Gültig bis');
	});
});

describe('Gespeichertes Design von früher', () => {
	it('macht aus dem fest eingetippten Titel ein Dokumenttyp-Feld', () => {
		const alt = [
			{
				id: 'title',
				type: 'text',
				content: 'Schülerausweis',
				x: 5,
				y: 8,
				width: 58,
				height: 4.5,
				zIndex: 1,
				show: true,
				style: { fontSize: 6.5 }
			}
		];
		const geheilt = heileAltBaender(alt);
		expect(geheilt[0].type).toBe('dokumenttyp');
		expect(geheilt[0].style.textTransform).toBeUndefined();
	});

	it('behält die Großschreibung der Vorlagen als Stilangabe', () => {
		const alt = [
			{
				id: 'title',
				type: 'text',
				content: 'SCHÜLERAUSWEIS',
				x: 30,
				y: 16.5,
				width: 50,
				height: 4,
				zIndex: 1,
				show: true,
				style: { fontSize: 6 }
			}
		];
		const geheilt = heileAltBaender(alt);
		expect(geheilt[0].type).toBe('dokumenttyp');
		expect(geheilt[0].style.textTransform).toBe('uppercase');
	});

	it('lässt jeden anderen Text in Ruhe', () => {
		const alt = [
			{
				id: 'subtitle',
				type: 'text',
				content: 'School ID · Carné Escolar',
				x: 5,
				y: 12,
				width: 58,
				height: 3,
				zIndex: 1,
				show: true,
				style: {}
			}
		];
		expect(heileAltBaender(alt)[0].type).toBe('text');
	});

	it('trägt die Aufschrift der Lehrkraft bis auf die gerenderte Karte', () => {
		idStore.front.elements = heileAltBaender([
			{
				id: 'title',
				type: 'text',
				content: 'Schülerausweis',
				x: 5,
				y: 8,
				width: 58,
				height: 4.5,
				zIndex: 1,
				show: true,
				style: { fontSize: 6.5 }
			}
		]);
		const { container } = render(CardFace, {
			props: { side: 'front', student: leser('lehrkraft'), barcodeType: 'code39' }
		});
		expect(container.textContent).toContain('Lehrerausweis');
	});
});
