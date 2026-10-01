import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
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
