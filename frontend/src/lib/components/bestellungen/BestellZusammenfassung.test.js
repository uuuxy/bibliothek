import { describe, it, expect, afterEach } from 'vitest';
import { render } from '@testing-library/svelte';
import BestellZusammenfassung from './BestellZusammenfassung.svelte';
import { bestellStore } from '../../stores/bestellStore.svelte.js';
import { authStore } from '../../stores/authStore.svelte.js';

// „Bestellung auslösen“ schickt POST /api/bestellungen, und die Tür verlangt create_orders. Ohne
// das Recht steht der Knopf nicht da, wie die Knöpfe in BestellDetail.
describe('BestellZusammenfassung: der Knopf „Bestellung auslösen“', () => {
	afterEach(() => {
		bestellStore.cart = [];
	});

	for (const [name, nutzer, sichtbar] of /** @type {const} */ ([
		[
			'mit dem Recht create_orders',
			{ rolle: 'mitarbeiter', permissions: ['view_orders', 'create_orders'] },
			true
		],
		['ohne das Recht create_orders', { rolle: 'mitarbeiter', permissions: ['view_orders'] }, false]
	])) {
		it(name, () => {
			authStore.currentUser = /** @type {any} */ (nutzer);
			bestellStore.cart = /** @type {any} */ ([
				{ id: 't1', titel: 'Titel A', menge: 2, preis: 10, mittel: 'land' }
			]);
			const screen = render(BestellZusammenfassung);
			expect(screen.queryByRole('button', { name: /Bestellung auslösen/ }) !== null).toBe(sichtbar);
		});
	}
});
