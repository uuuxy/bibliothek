import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot, ohneKommentare } from './hygiene-quellen.js';
import { authStore } from './stores/authStore.svelte.js';
import { appState } from '../inventur/lib/store.svelte.js';

// Die Rolle Leitung (Migration 121/122). Sie ist im Server vollständig, sobald der
// ENUM-Wert und die Rechte stehen — aber genau so war die Rolle Helfer monatelang
// „fertig" und trotzdem unerreichbar: Die Oberfläche bot sie nicht an. Diese Datei prüft
// die drei Stellen, an denen eine Rolle in der Oberfläche vorkommen MUSS, damit sie
// vergeben und gepflegt werden kann.
//
// Geprüft wird am Quelltext, nicht am gerenderten Bauteil: Die Rollenliste und die
// Spalten der Rechte-Matrix sind Konstanten in den Komponenten. Ein Render-Test würde
// dieselbe Konstante über drei Ebenen Svelte-Kompilat zurücklesen und dabei nur
// beweisen, dass Svelte funktioniert.
function quelle(pfad) {
	return readFileSync(join(srcRoot, pfad), 'utf8');
}

describe('Rolle Leitung in der Oberfläche', () => {
	it('die Benutzerverwaltung bietet die Rolle zur Auswahl an', () => {
		const src = quelle('lib/UserManagementEditModal.svelte');
		expect(src).toContain("value: 'leitung'");
		expect(src).toMatch(/label: '[^']*Leitung/);
	});

	it('Kollegium steht VOR den Rollen — es ist der Grundzustand, keine Rolle', () => {
		// Absprache vom 16.09.2026: „Die Selbstregistration ist in dem Sinn keine Rolle. Das sind
		// einfach alle, alle Lehrer. Nur einige werden anhand ihrer E-Mail zu höheren
		// Berufen." Steht Kollegium zwischen Mitarbeiter und Administrator, liest die
		// Liste sich als Rangfolge, in der das Kollegium eine Stufe wäre.
		const src = quelle('lib/UserManagementEditModal.svelte');
		const kollegium = src.indexOf("value: 'kollegium'");
		const leitung = src.indexOf("value: 'leitung'");
		const admin = src.indexOf("value: 'admin'");
		expect(kollegium).toBeGreaterThan(-1);
		expect(kollegium).toBeLessThan(leitung);
		expect(kollegium).toBeLessThan(admin);
	});

	it('die Rechte-Matrix hat eine Spalte für die Leitung', () => {
		// Ohne die Spalte wäre die Rolle nur über die Vorgabe im Seed steuerbar — genau
		// der Audit-Befund vom 01.08.2026 zur Rolle Helfer.
		const src = quelle('lib/PermissionsEditor.svelte');
		expect(src).toContain("'LEITUNG', 'leitung'");
	});

	it('die Matrix belegt die Spalte vor, damit sie auch ohne Server-Zeile erscheint', () => {
		const src = quelle('lib/PermissionManager.svelte');
		expect(src).toMatch(/const newState = \{[^;]*\bleitung:/);
	});

	// Die Gegenrichtung zum Test darüber, und der Grund, warum beide hier stehen:
	// Absprache vom 16.09.2026, beim Blick auf die Matrix — „was soll der Scheiss dass wir auf
	// einmal jetzt Kollegium bei Rollen haben? das braucht doch niemand!"
	//
	// Die Spalte war nicht neu (sie stand dort seit dem 10.08.2026, als die Rolle
	// „lehrer" in „kollegium" umbenannt wurde, 15d2806e), aber sie widersprach dem
	// Modell: Kollegium ist der Grundzustand jeder Lehrkraft, keine Rolle neben Leitung
	// und Mitarbeiter. Seine Rechte stehen fest in db/seed.go und sind eine
	// Produktentscheidung, kein Schalter je Schule.
	//
	// ohneKommentare ist hier Pflicht und keine Sorgfalt: In PermissionsEditor.svelte
	// steht ein Kommentar, der die entfernte Spalte beim Namen nennt und begründet. Ohne
	// das Strippen prüfte dieser Test die Begründung statt des Bauteils und bliebe
	// für immer rot — dieselbe Falle wie bei einer Ratsche, die einen Kommentar liest.
	it('die Rechte-Matrix hat KEINE Spalte für das Kollegium', () => {
		const src = ohneKommentare(quelle('lib/PermissionsEditor.svelte'));
		expect(src).not.toContain('KOLLEGIUM');
		expect(src).not.toContain("'kollegium'");
	});
});

describe('Login-Weiche', () => {
	beforeEach(() => {
		authStore.handleLogout();
		vi.clearAllMocks();
		// @ts-expect-error  Test-Double: Stub statt vollständiger EventSource
		globalThis.EventSource = vi.fn(function () {
			return { addEventListener: vi.fn(), close: vi.fn() };
		});
	});

	afterEach(() => {
		// Ohne den Abbau laufen SSE-Verbindung und Sitzungs-Auffrischung weiter und
		// melden nach dem Ende der Testdatei einen Unhandled Error (siehe
		// docs/sweeps.md: „Timer überlebt den Abbau").
		authStore.handleLogout();
	});

	// Die Rechte kommen im Login mit (auth/handlers.go lädt sie aus role_permissions) —
	// ohne sie prüfte der Test eine Weiche, die es so nie sieht.
	async function meldeAn(rolle, permissions) {
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({
			ok: true,
			json: async () => ({
				id: 'u-1',
				rolle,
				vorname: 'Test',
				nachname: 'Person',
				permissions
			}),
			text: async () => ''
		}));
		authStore.loginEmail = 'test@example.com';
		authStore.loginPassword = 'geheim';
		await authStore.handleLogin(null);
	}

	it('eine Leitung kommt in die Verwaltung, nicht ins Portal', async () => {
		// Rechte einer Leitung ab Werk: alles außer manage_users/manage_settings.
		await meldeAn('leitung', ['perform_actions', 'view_students', 'view_books']);
		expect(appState.adminAuthenticated).toBe(true);
	});

	it('das Kollegium bekommt weiter nur sein Portal', async () => {
		// Rechte des Kollegiums ab Werk: nur create_reservations (db/seed.go).
		await meldeAn('kollegium', ['create_reservations']);
		expect(appState.adminAuthenticated).toBe(false);
		expect(appState.guestAuthenticated).toBe(true);
	});

	it('die Weiche kennt keine Rollennamen mehr — die nächste Rolle braucht sie nicht', () => {
		// Der Kern: Die Weiche fragt, ob jemand KEIN Kollegium ist, statt drei
		// Rollennamen aufzuzählen. Sonst ist beim Bau jeder weiteren Rolle wieder
		// genau diese Stelle zu pflegen, und wer sie vergisst, schickt die neue Rolle
		// still ins Portal.
		// OHNE Kommentare: Der Umbau erklärt sich an Ort und Stelle und ZITIERT dabei den
		// alten Vergleich. Ein Detektor, der die Begründung für die Sache hält, ist selbst
		// eine lügende Ratsche (Bugklasse in docs/sweeps.md) — hier wäre er rot geblieben,
		// obwohl der Code sauber ist, und wer ihn dann „reparierte", hätte am Kommentar
		// gemessen statt am Code.
		const code = ohneKommentare(quelle('lib/stores/authStore.svelte.js'));
		expect(code).not.toMatch(/rolle\s*===\s*'mitarbeiter'/);
		expect(code).not.toMatch(/rolle\s*===\s*'admin'/);
	});
});
