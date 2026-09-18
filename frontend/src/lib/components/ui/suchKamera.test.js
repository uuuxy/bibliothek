import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import Suchfeld from './Suchfeld.svelte';
import Suchpille from './Suchpille.svelte';

// Kamera-Scanner in den gemeinsamen Suchbauteilen (18.09.2026): ein Schalter `kamera`,
// Standard aus. Der Test hält beides fest — dass der Knopf mit dem Schalter da ist UND
// dass er ohne ihn fehlt: Die Suchpille steht auf gut einem Dutzend Seiten, und ein
// Kamera-Knopf, der überall auftaucht, wäre genau die Nebenwirkung, die der Schalter
// verhindern soll. Die Kamera selbst wird hier nicht gestartet (jsdom hat keine).
const KNOPF = /Kamera-Barcode-Scanner/;
const PROPS = { wert: '', platzhalter: 'Suchen …', etikett: 'Suche' };

describe('Kamera-Schalter der Suchbauteile', () => {
	it('Suchfeld: Knopf nur mit kamera', () => {
		const mit = render(Suchfeld, { ...PROPS, kamera: true });
		expect(mit.queryByRole('button', { name: KNOPF })).toBeTruthy();
		mit.unmount();
		const ohne = render(Suchfeld, { ...PROPS });
		expect(ohne.queryByRole('button', { name: KNOPF })).toBeNull();
	});

	it('Suchpille: Knopf nur mit kamera', () => {
		const mit = render(Suchpille, { ...PROPS, id: 'p1', kamera: true });
		expect(mit.queryByRole('button', { name: KNOPF })).toBeTruthy();
		mit.unmount();
		const ohne = render(Suchpille, { ...PROPS, id: 'p2' });
		expect(ohne.queryByRole('button', { name: KNOPF })).toBeNull();
	});
});
