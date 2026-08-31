import { describe, it, expect, beforeEach, vi } from 'vitest';
import { authStore } from './authStore.svelte.js';
import { idleLock } from './idleLock.svelte.js';
import { omniboxStore } from './omnibox.svelte.js';
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
		omniboxStore.activeStudent = { id: 's1', vorname: 'Mia', nachname: 'Muster' };
		omniboxStore.queryVal = 'Mül';
		omniboxStore.blockAlert = { message: 'Gesperrt: Ausweismissbrauch', query: 'S-1' };
		uiStore.requestedStudentId = 's1';
	});

	it('Abmelden leert die Theke — kein Profil für den nächsten Bediener', () => {
		authStore.handleLogout();
		expect(omniboxStore.activeStudent, 'Schülerprofil überlebt das Abmelden').toBeNull();
		expect(omniboxStore.blockAlert, 'Sperrgrund (PII) überlebt das Abmelden').toBeNull();
		expect(omniboxStore.queryVal).toBe('');
		expect(uiStore.requestedStudentId).toBeNull();
	});

	it('der Inaktivitäts-Wächter räumt dieselbe Liste', () => {
		idleLock.thekeLeeren();
		expect(omniboxStore.activeStudent).toBeNull();
		expect(omniboxStore.blockAlert).toBeNull();
		expect(omniboxStore.queryVal).toBe('');
	});
});
