import { describe, it, expect, vi, beforeAll } from 'vitest';
import { fireEvent, render } from '@testing-library/svelte';
import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import MahnwesenAktionen from './MahnwesenAktionen.svelte';
import { apiFetch } from '../../apiFetch.js';
import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';

// Nur apiFetch ersetzen, den Rest des Moduls behalten: authStore importiert daraus auch
// registriereSitzungAbgelaufenHandler — ein Mock ohne diesen Export lässt schon das Laden
// des Stores scheitern, und der Test wäre rot aus dem falschen Grund.
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	...(await importOriginal()),
	apiFetch: vi.fn()
}));

/**
 * „Alle anmahnen" schickt an POST /api/mail/send-bulk-overdue, und diese Route verlangt
 * create_orders (api/routes_students.go). Die Seite selbst hängt an view_students — wer
 * sie sehen darf, darf also nicht zwangsläufig mahnen.
 *
 * Bis zum 18.09.2026 stand der Knopf trotzdem für jeden da: Er erschien allein, sobald
 * überfällige Schüler geladen waren. Wer ihn ohne das Recht drückte, bekam nach dem
 * Versanddialog samt Klassenauswahl einen 403 — nachgestellt am 18.09.2026 an der echten
 * Route mit der echten Middleware. Ab Werk trifft es niemanden (alle Rollen mit
 * view_students haben auch create_orders); die Berechtigungsseite erlaubt die Kombination
 * aber, und „mahnen ja, bestellen nein" ist eine, die eine Schule plausibel will.
 *
 * Dieselbe Lücke wie beim Abgänger-Mail-Knopf (AbgaengerKopfzeile.test.js, 12.09.2026),
 * dieselbe Regel: Sichtbarkeit einer Aktion = hatRecht(user, '<Recht der Route>').
 */
const PROPS = { onMahnlauf: () => {}, onBescheid: () => {} };

beforeAll(async () => {
	// Der Knopf setzt geladene Überfällige voraus — ohne sie wäre „kein Knopf" auch ohne
	// Rechteprüfung wahr, und der Test misst nichts.
	/** @type {any} */ (apiFetch).mockResolvedValue({
		ok: true,
		json: async () => ({
			klassen: [
				{
					klasse: '5a',
					schueler: [
						{
							schueler_id: 's1',
							name: 'Test Schüler',
							klasse: '5a',
							medien: [{ titel: 'Testbuch', faellig_am: '24.09.2026', tage_ueberfaellig: 11 }]
						}
					]
				}
			]
		})
	});
	await mahnwesenStore.fetchData();
	if (mahnwesenStore.klassen.length === 0) throw new Error('Testdaten nicht geladen');
});

describe('MahnwesenAktionen', () => {
	it('zeigt „Alle anmahnen" nur mit dem Recht der Route (darfMahnlauf)', () => {
		const mit = render(MahnwesenAktionen, { ...PROPS, darfBescheid: false, darfMahnlauf: true });
		expect(mit.queryByRole('button', { name: /Alle anmahnen/ })).toBeTruthy();
		mit.unmount();

		const ohne = render(MahnwesenAktionen, { ...PROPS, darfBescheid: false, darfMahnlauf: false });
		expect(ohne.queryByRole('button', { name: /Alle anmahnen/ })).toBeNull();
		// Die Seite bleibt bedienbar: Neu laden und Drucken hängen an view_students, also
		// am Recht, mit dem sie überhaupt offen ist. Papier ist hier der Notweg.
		expect(ohne.queryByRole('button', { name: 'Daten neu laden' })).toBeTruthy();
	});

	it('druckt mit „Liste drucken" die Liste, wie sie gerade dasteht', async () => {
		const fenster = {
			document: { open: vi.fn(), write: vi.fn(), close: vi.fn() },
			focus: vi.fn(),
			print: vi.fn()
		};
		const oeffnen = vi.spyOn(window, 'open').mockReturnValue(/** @type {any} */ (fenster));
		try {
			const zeile = render(MahnwesenAktionen, {
				...PROPS,
				darfBescheid: false,
				darfMahnlauf: false
			});
			await fireEvent.click(zeile.getByRole('button', { name: 'Liste drucken' }));

			const html = fenster.document.write.mock.calls[0][0];
			expect(html).toContain('<h1>Mahnliste</h1>');
			expect(html).toContain(
				'<td class="schmal">5a</td><td>Test Schüler</td><td>Testbuch</td><td class="schmal">24.09.2026</td>'
			);
			expect(fenster.print).toHaveBeenCalledTimes(1);
			zeile.unmount();

			// Nichts in der Liste, nichts zu drucken: Die Suche trifft kein Kind.
			mahnwesenStore.searchQuery = 'gibt es nicht';
			const leer = render(MahnwesenAktionen, {
				...PROPS,
				darfBescheid: false,
				darfMahnlauf: false
			});
			expect(leer.getByRole('button', { name: 'Liste drucken' })).toHaveProperty('disabled', true);
		} finally {
			mahnwesenStore.searchQuery = '';
			oeffnen.mockRestore();
		}
	});

	it('bindet den Knopf in Mahnwesen.svelte an create_orders — das Recht der Route', () => {
		// Quelltext-Prüfung, weil die Seite selbst (Live-Ereignisse, Laden beim Mount) hier
		// nicht gerendert wird. Der Knopf hängt an einem Prop; ohne diese Prüfung könnte die
		// Seite ihn an ein beliebiges Recht binden, und der Test oben bliebe grün.
		let verzeichnis = process.cwd();
		while (!existsSync(resolve(verzeichnis, 'src/lib/Mahnwesen.svelte'))) {
			const eltern = dirname(verzeichnis);
			if (eltern === verzeichnis) throw new Error('Mahnwesen.svelte nicht gefunden');
			verzeichnis = eltern;
		}
		const seite = readFileSync(resolve(verzeichnis, 'src/lib/Mahnwesen.svelte'), 'utf8');
		// Bekannte Zeile als Nicht-leer-Garantie: Sie steht seit dem Bescheid-Knopf dort.
		expect(seite).toContain("hatRecht(authStore.currentUser, 'edit_students')");
		expect(seite).toContain(
			"const darfMahnlauf = $derived(hatRecht(authStore.currentUser, 'create_orders'))"
		);
		expect(seite).toContain('{darfMahnlauf}');
		// Der Versand-Dialog bietet nur Klassen an, nicht die Gruppe der Ehemaligen.
		expect(seite).toContain('klassen={mahnwesenStore.versandKlassen}');
	});

	it('bietet „Alle anmahnen" nicht an, wenn nur Ehemalige überfällig sind', async () => {
		// Die Gruppe der Ehemaligen geht an keine Klassenleitung; der Mahnlauf hätte niemanden.
		/** @type {any} */ (apiFetch).mockResolvedValue({
			ok: true,
			json: async () => ({
				klassen: [
					{
						klasse: 'Ehemalige',
						ehemalige: true,
						schueler: [{ schueler_id: 's9', name: 'Erik Test', klasse: 'Ehemalige', medien: [] }]
					}
				]
			})
		});
		await mahnwesenStore.fetchData();
		expect(mahnwesenStore.klassen).toHaveLength(1);
		expect(mahnwesenStore.versandKlassen).toEqual([]);

		const zeile = render(MahnwesenAktionen, { ...PROPS, darfBescheid: false, darfMahnlauf: true });
		expect(zeile.queryByRole('button', { name: /Alle anmahnen/ })).toBeNull();
		expect(zeile.queryByRole('button', { name: 'Daten neu laden' })).toBeTruthy();
	});
});
