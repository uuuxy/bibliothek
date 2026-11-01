import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import Tabelle from './Tabelle.svelte';

// Der Screenreader nennt zu einer Zelle ihre Spalte und Zeile über den scope der Kopfzellen; die
// Aufrufer schreiben ihn nicht selbst.
describe('Tabelle: scope der Kopfzellen', () => {
	/** @param {string} html */
	const zeige = (html) =>
		render(Tabelle, {
			beschriftung: 'Probe',
			children: createRawSnippet(() => ({ render: () => html }))
		}).container;

	it('setzt col im Kopf und lässt einen gesetzten stehen', () => {
		const tabelle = zeige('<thead><tr><th>Name</th><th scope="colgroup">Gruppe</th></tr></thead>');
		const scopes = [...tabelle.querySelectorAll('th')].map((th) => th.getAttribute('scope'));
		expect(scopes).toEqual(['col', 'colgroup']);
	});

	it('setzt row im Rumpf', () => {
		const tabelle = zeige('<tbody><tr><th>Zeile</th><td>1</td></tr></tbody>');
		expect(tabelle.querySelector('th')?.getAttribute('scope')).toBe('row');
	});
});
