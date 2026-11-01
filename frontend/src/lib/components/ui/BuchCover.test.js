import { describe, it, expect } from 'vitest';
import { fireEvent, render } from '@testing-library/svelte';
import BuchCover from './BuchCover.svelte';

// Welche Quellen ein Cover fragt. Google Books antwortet auf eine ISBN ohne Bild mit einem
// Ersatzbild und Status 200; eine Liste über Bücher im Bestand soll es nicht zeigen.
describe('BuchCover: Quellen', () => {
	const titel = 'Der Zauberberg';

	it('fragt ohne gespeichertes Cover die fremden Quellen über den eigenen Proxy', () => {
		const { container } = render(BuchCover, { isbn: '9783100000000', titel });
		const bild = container.querySelector('img');
		expect(bild?.getAttribute('src')).toContain('/api/images/cover?isbn=9783100000000');
		expect(bild?.getAttribute('src')).toContain('books.google.com');
	});

	it('zeigt mit nurGespeichert ohne gespeichertes Cover die Initiale und fragt niemanden', () => {
		const { container } = render(BuchCover, { isbn: '9783100000000', titel, nurGespeichert: true });
		expect(container.querySelector('img')).toBeNull();
		expect(container.textContent?.trim()).toBe('D');
	});

	it('zeigt mit nurGespeichert das lokal abgelegte Cover', () => {
		const { container } = render(BuchCover, {
			coverUrl: '/uploads/covers/9783100000000.webp',
			isbn: '9783100000000',
			titel,
			nurGespeichert: true
		});
		expect(container.querySelector('img')?.getAttribute('src')).toBe(
			'/uploads/covers/9783100000000.webp'
		);
	});

	it('holt mit nurGespeichert ein fremd gespeichertes Cover über den Proxy und nichts weiter', () => {
		const { container } = render(BuchCover, {
			coverUrl: 'https://portal.dnb.de/opac/mvb/cover?isbn=9783100000000',
			isbn: '9783100000000',
			titel,
			nurGespeichert: true
		});
		const quelle = container.querySelector('img')?.getAttribute('src') ?? '';
		expect(quelle).toContain('/api/images/cover?isbn=9783100000000');
		expect(quelle).toContain(encodeURIComponent('https://portal.dnb.de/'));
		expect(quelle).not.toContain('books.google.com');
	});
});

// Auf der Fläche einer Kachel ginge eine Initiale verloren; dort stehen Fach und Titel.
describe('BuchCover: Platzhalter', () => {
	it('die Kachel zeigt ohne Bild Fach und Titel, die Zeile die Initiale', () => {
		const kachel = render(BuchCover, {
			titel: 'Krabat',
			fach: 'Deutsch',
			groesse: 'kachel',
			nurGespeichert: true
		});
		expect(kachel.container.textContent).toContain('Deutsch');
		expect(kachel.container.textContent).toContain('Krabat');

		const zeile = render(BuchCover, { titel: 'Krabat', nurGespeichert: true });
		expect(zeile.container.textContent?.trim()).toBe('K');
	});
});

// Bleibt ein Bild aus, kommt die nächste Quelle dran und am Ende die Kachel ohne Bild. Das 1×1-GIF
// des Proxys lädt erfolgreich und zählt trotzdem als ausgeblieben.
describe('BuchCover: Ersatz', () => {
	const props = { isbn: '9783100000000', titel: 'Der Zauberberg' };
	const bild = (/** @type {HTMLElement} */ container) =>
		/** @type {HTMLImageElement | null} */ (container.querySelector('img'));

	it('geht bei einem Fehler zur nächsten Quelle und am Ende zur Kachel', async () => {
		const { container } = render(BuchCover, props);
		const erste = bild(container)?.getAttribute('src');
		await fireEvent.error(/** @type {HTMLImageElement} */ (bild(container)));
		expect(bild(container), 'die zweite Quelle kommt noch dran').not.toBeNull();
		expect(bild(container)?.getAttribute('src')).not.toBe(erste);

		for (let i = 0; i < 10 && bild(container); i++) {
			await fireEvent.error(/** @type {HTMLImageElement} */ (bild(container)));
		}
		expect(bild(container)).toBeNull();
	});

	it('sieht das 1×1-Bild als ausgeblieben an, ein echtes Bild nicht', async () => {
		const { container } = render(BuchCover, props);
		const lade = async (/** @type {number} */ breite, /** @type {number} */ hoehe) => {
			const img = /** @type {HTMLImageElement} */ (bild(container));
			Object.defineProperty(img, 'naturalWidth', { value: breite, configurable: true });
			Object.defineProperty(img, 'naturalHeight', { value: hoehe, configurable: true });
			await fireEvent.load(img);
		};

		const erste = bild(container)?.getAttribute('src');
		await lade(200, 300);
		expect(bild(container)?.getAttribute('src')).toBe(erste);
		await lade(1, 1);
		expect(bild(container), 'die zweite Quelle kommt noch dran').not.toBeNull();
		expect(bild(container)?.getAttribute('src')).not.toBe(erste);
	});
});
