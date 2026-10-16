import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { authStore } from './authStore.svelte.js';
import { leerlaufSperre } from './leerlaufSperre.svelte.js';
import { omniboxStore } from './omnibox.svelte.js';

// Nach jedem Fall die Zeitgeber der Theke stoppen. `scanfeldWiederScharfstellen` plant
// einen Fokussprung über 50 ms, der `document` anfasst — endet die Datei vorher, baut
// Vitest jsdom ab, und der Rückruf reisst den GANZEN Lauf rot („Unhandled Errors:
// document is not defined"), obwohl jeder Test grün ist. Genau so stand die CI am
// 16.09.2026. Belegt in stores/omniboxZeitgeber.test.js.
afterEach(() => omniboxStore.stoppeZeitgeber());

import { uiStore } from './uiStore.svelte.js';

// Der Bedienerwechsel an der Theke.
//
// Die Omnibox-Stores sind Modul-Singletons: Sie überleben das Abmelden. Bis zum
// 31.08.2026 leerte nur der Inaktivitäts-Wächter die Theke — wer sich ABMELDETE, ließ
// das geladene Schülerprofil samt Ausleihen, Sperren und Mahnstufen für den nächsten
// Bediener stehen. Genau der Handover-Fall, gegen den der Timer gebaut wurde, nur
// schneller.
describe('Theken-Übergabe', () => {
	beforeEach(() => {
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200 }));
		omniboxStore.aktiverLeser = { id: 's1', vorname: 'Mia', nachname: 'Muster' };
		omniboxStore.eingabe = 'Mül';
		omniboxStore.sperrHinweis = {
			message: 'Gesperrt: Ausweismissbrauch',
			query: 'S-1',
			art: 'leser'
		};
		uiStore.requestedStudentId = 's1';
	});

	it('Abmelden leert die Theke — kein Profil für den nächsten Bediener', () => {
		authStore.handleLogout();
		expect(omniboxStore.aktiverLeser, 'Schülerprofil überlebt das Abmelden').toBeNull();
		expect(omniboxStore.sperrHinweis, 'Sperrgrund (PII) überlebt das Abmelden').toBeNull();
		expect(omniboxStore.eingabe).toBe('');
		expect(uiStore.requestedStudentId).toBeNull();
	});

	it('der Inaktivitäts-Wächter räumt dieselbe Liste', () => {
		leerlaufSperre.thekeLeeren();
		expect(omniboxStore.aktiverLeser).toBeNull();
		expect(omniboxStore.sperrHinweis).toBeNull();
		expect(omniboxStore.eingabe).toBe('');
	});
});
