import { describe, it, expect, beforeEach } from 'vitest';
import {
	idStore,
	resetDesign,
	applyDesign,
	defaultFrontElements,
	wendeSchulstammdatenAn,
	PLATZHALTER_SCHULNAME,
	PLATZHALTER_ADRESSE,
	fuegeRahmenHinzu,
	serializeDesign
} from './idDesignerStore.svelte.js';
import { designStand, designWurdeGeladen, vergissDesignLadezustand } from './designSitzung.js';

function header() {
	return idStore.front.elements.find((e) => e.id === 'header');
}
function address() {
	return idStore.front.elements.find((e) => e.id === 'address');
}

describe('wendeSchulstammdatenAn', () => {
	beforeEach(() => {
		resetDesign();
	});

	it('ersetzt den Platzhalter durch die echten Schul-Stammdaten', () => {
		applyDesign({
			schule: { name: 'Philipp-Reis-Schule', adresse: 'Schulstraße 1, 61476 Kronberg' }
		});

		expect(header().content).toBe('Philipp-Reis-Schule');
		expect(address().content).toBe('Schulstraße 1, 61476 Kronberg');
	});

	// „Standardwerte wiederherstellen" und die Vorlagen setzen den Kopf auf den Platzhalter zurück.
	it('kennt die Schule auch nach dem Zurücksetzen noch', () => {
		applyDesign({
			schule: { name: 'Philipp-Reis-Schule', adresse: 'Schulstraße 1, 61476 Kronberg' }
		});
		resetDesign();
		expect(header().content).toBe(PLATZHALTER_SCHULNAME);

		wendeSchulstammdatenAn();

		expect(header().content).toBe('Philipp-Reis-Schule');
	});

	it('rührt einen bereits selbst eingetragenen Kopf nicht an', () => {
		header().content = 'Von Hand eingetragener Schulname';

		applyDesign({
			schule: { name: 'Philipp-Reis-Schule', adresse: 'Schulstraße 1, 61476 Kronberg' }
		});

		expect(header().content).toBe('Von Hand eingetragener Schulname');
		// Die Adresse ist weiterhin der Platzhalter, weil sie einzeln geprüft wird —
		// ein bearbeiteter Kopf darf die Adresse nicht mit heilen.
		expect(address().content).toBe('Schulstraße 1, 61476 Kronberg');
	});

	it('lässt den Platzhalter stehen, wenn die Einstellungen selbst leer sind', () => {
		applyDesign({ schule: { name: '', adresse: '' } });

		expect(header().content).toBe(PLATZHALTER_SCHULNAME);
		expect(address().content).toBe(PLATZHALTER_ADRESSE);
	});

	it('ist ein no-op ohne Header-/Adress-Element auf der Seite (defensiv)', () => {
		idStore.front.elements = idStore.front.elements.filter((e) => e.id !== 'header');
		expect(() =>
			applyDesign({ schule: { name: 'Philipp-Reis-Schule', adresse: 'X' } })
		).not.toThrow();
	});
});

// Die Klassenzeile gehört nicht auf den Ausweis: Eine Karte, die die Klasse trägt,
// wäre nach jedem Schuljahreswechsel falsch und müsste neu gedruckt werden — für ein
// Dokument, das sonst die ganze Schulzeit gilt, ist das sinnlos. Die Klasse steht im
// System, wo sie sich ohne Nachdruck ändert.
//
// Geprüft wird nicht die Vorgabe allein, sondern der Ladeweg: Das Design ist ZENTRAL
// gespeichert, und applyDesign() überschreibt die Vorgaben mit dem Stand aus der
// Datenbank. Eine bestehende Installation trüge das Element sonst für immer weiter —
// derselbe Weg, auf dem der Musterstadt-Kopf hängenblieb.
describe('Klassenzeile auf dem Ausweis', () => {
	beforeEach(() => {
		resetDesign();
	});

	it('kommt in den Standardwerten nicht mehr vor', () => {
		expect(defaultFrontElements().some((e) => e.type === 'details')).toBe(false);
	});

	it('wird aus einem zentral gespeicherten Altdesign beim Laden entfernt', () => {
		applyDesign({
			front: {
				elements: [
					{ id: 'name', type: 'name', content: '', show: true },
					{ id: 'details', type: 'details', content: '', show: true },
					{ id: 'barcode', type: 'barcode', content: '', show: true }
				],
				theme: 'bg-white text-black border-slate-200'
			}
		});

		expect(idStore.front.elements.some((e) => e.type === 'details')).toBe(false);
		// Die übrigen Elemente bleiben unangetastet — gefiltert wird genau eines.
		expect(idStore.front.elements.map((e) => e.id)).toEqual(['name', 'barcode']);
	});
});

// Eine Fläche kommt auf die Seite, die gerade offen ist.
describe('fuegeRahmenHinzu', () => {
	beforeEach(() => resetDesign());

	it('legt die Fläche auf die genannte Seite und lässt die andere', () => {
		const vorn = idStore.front.elements.length;
		const hinten = idStore.back.elements.length;

		fuegeRahmenHinzu('back');
		expect(idStore.back.elements).toHaveLength(hinten + 1);
		expect(idStore.front.elements).toHaveLength(vorn);

		fuegeRahmenHinzu('front');
		expect(idStore.front.elements).toHaveLength(vorn + 1);
	});
});

// Ein zentral gespeichertes Design kann noch den abgeschafften A4-Bogen tragen. Es wird auf den
// Kartendrucker gelesen; sonst stünde beim nächsten Speichern 'a4' wieder in der Datenbank.
describe('applyDesign: die Druckart', () => {
	beforeEach(() => resetDesign());

	it('liest den alten A4-Bogen als Kartendrucker', () => {
		idStore.printMode = 'etikett';

		applyDesign({ printMode: 'a4' });

		expect(idStore.printMode).toBe('card');
	});

	it('übernimmt Kartendrucker und Etikettenbogen, ein Design ohne Druckart lässt sie stehen', () => {
		applyDesign({ printMode: 'etikett' });
		expect(idStore.printMode).toBe('etikett');

		applyDesign({});
		expect(idStore.printMode).toBe('etikett');

		applyDesign({ printMode: 'card' });
		expect(idStore.printMode).toBe('card');
	});
});

// Der Änderungsstand kommt mit dem Design vom Server und geht beim Speichern zurück
// (idDesignPersistenz). Er steht neben dem Store, nicht darin: Das Design, das an den Server
// geht, trägt ihn nicht.
describe('applyDesign: Änderungsstand', () => {
	beforeEach(() => {
		vergissDesignLadezustand();
	});

	it('merkt sich den Stand der Antwort und hält ihn aus dem Design heraus', () => {
		applyDesign({ stand: '2026-11-11T21:15:03.123456Z', printMode: 'card' });
		expect(designStand()).toBe('2026-11-11T21:15:03.123456Z');
		expect(designWurdeGeladen()).toBe(true);
		expect(JSON.stringify(serializeDesign())).not.toContain('2026-11-11T21:15:03');
	});

	it('nimmt eine Antwort ohne Stand als „noch keines gespeichert"', () => {
		applyDesign({ stand: 'alt' });
		applyDesign({});
		expect(designStand()).toBe('');
	});
});
