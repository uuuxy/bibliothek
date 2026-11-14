import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import KontoBearbeitenDialog from './KontoBearbeitenDialog.svelte';
import { leeresBenutzerFormular } from './benutzerFormular.js';

/** @param {Record<string, any>} formular */
const zeige = (formular) =>
	render(KontoBearbeitenDialog, {
		props: {
			open: true,
			onclose: () => {},
			istAenderung: false,
			formular,
			speichert: false,
			fehler: null,
			speichereKonto: () => {}
		}
	});

// Die LDAP-Kennung öffnet ein LDAP-Konto wie die E-Mail; ein IMAP-Konto hat keine, und ein
// leeres Feld daneben sähe aus wie eine vergessene Angabe.
describe('KontoBearbeitenDialog: Anmeldeverfahren', () => {
	it('fragt nach dem Verfahren und bei IMAP nach keiner Kennung', () => {
		const k = zeige(leeresBenutzerFormular());
		expect(k.getByLabelText('Anmeldung über')).toBeTruthy();
		expect(k.queryByLabelText('LDAP-Kennung')).toBeNull();
	});

	it('verlangt bei LDAP die Kennung', () => {
		const k = zeige({ ...leeresBenutzerFormular(), anmeldeverfahren: 'ldap' });
		const feld = k.getByLabelText('LDAP-Kennung');
		expect(feld.hasAttribute('required')).toBe(true);
	});
});
