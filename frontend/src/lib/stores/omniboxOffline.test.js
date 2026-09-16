import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import 'fake-indexeddb/auto';

// Ein Offline-Scan bei geladenem Schüler ist eine AUSLEIHE (Rasterdurchgang 06.09.2026).
//
// Bis dahin legte die Omnibox JEDEN Offline-Scan als „checkin" ab. Der Payload-Bauer des
// Syncs schickt `active_leser_id` nur bei „checkout" — einem Typ, den niemand je
// einreihte; der Zweig war unerreichbar. Der Server las das Schweigen als Rückgabe: Das
// Buch war schon draußen, die Rückgabe scheiterte mit 400, und der Eintrag flog aus der
// Warteschlange. Das Kind hatte das Buch, das System sagte „verfügbar".
//
// Geprüft wird der ECHTE Weg: submitAction mit einem Netzwerkfehler, so wie er im
// WLAN-Loch entsteht.
vi.mock('../apiFetch.js', () => ({
	apiFetch: vi.fn(async () => {
		throw new TypeError('Failed to fetch');
	}),
	apiClient: { post: vi.fn() }
}));
vi.mock('../audio.js', () => ({
	playSoundSuccess: vi.fn(),
	playSoundError: vi.fn(),
	playSuccessBeep: vi.fn(),
	playErrorBeep: vi.fn()
}));
vi.mock('../../inventur/lib/store.svelte.js', () => ({ showToast: vi.fn() }));
// Die Barcode-Liste wird gestellt: Hier geht es um das EINREIHEN, nicht um das Holen.
vi.mock('./buchBarcodes.svelte.js', () => ({
	buchBarcodes: {
		/** @param {string} n */
		istBuch: (n) => n === '58968',
		bereitstellen: vi.fn(),
		laden: vi.fn(),
		auffrischen: vi.fn(),
		anzahl: 1,
		geholtAm: 0
	}
}));

import { apiClient } from '../apiFetch.js';
import { omniboxStore } from './omnibox.svelte.js';

// Nach jedem Fall die Zeitgeber der Theke stoppen. `scanfeldWiederScharfstellen` plant
// einen Fokussprung über 50 ms, der `document` anfasst — endet die Datei vorher, baut
// Vitest jsdom ab, und der Rückruf reisst den GANZEN Lauf rot („Unhandled Errors:
// document is not defined"), obwohl jeder Test grün ist. Genau so stand die CI am
// 16.09.2026. Belegt in stores/omniboxZeitgeber.test.js.
afterEach(() => omniboxStore.stoppeZeitgeber());

import { loadQueue, dequeueOfflineAction } from '../offlineQueue.js';

async function leere() {
	for (const item of await loadQueue()) await dequeueOfflineAction(item.id);
}

describe('Omnibox offline', () => {
	beforeEach(async () => {
		await leere();
		vi.clearAllMocks();
		// Ein ECHTER Versandfehler, wie er im WLAN-Loch entsteht. Bis zum 15.09.2026 lieferte
		// der Mock undefined, und der TypeError kam aus `res.ok` — der Test maß den Fehler aus
		// Commit 2 (Auswertung im selben catch), nicht den Netzausfall.
		vi.mocked(apiClient.post).mockRejectedValue(new TypeError('Failed to fetch'));
		omniboxStore.activeStudent = null;
		// Der Ausweis-Merker ueberlebt einen Scan mit Absicht — genau deshalb muss ihn
		// jeder Fall ausdruecklich raeumen, sonst misst der naechste die Spuren des vorigen.
		omniboxStore.offlineAusweis = '';
		omniboxStore.queryVal = '';
	});

	// Der Eintrag trägt die Person VOM SCAN, nicht die vom Zeitpunkt des Scheiterns
	// (OFFEN.md 2.2, Commit 1). Bis dahin las speichereOfflineAktion Person und Absicht erst
	// nach der hängenden Anfrage (Timeout 10 s): Escape oder „Theke leeren" in dieser Zeit,
	// und das Buch ging als Rückgabe ohne Person in die Warteschlange.
	it('hält Person und Absicht beim Scan fest, nicht erst beim Scheitern', async () => {
		/** @type {(e: Error) => void} */
		let scheitern = () => {};
		vi.mocked(apiClient.post).mockImplementationOnce(
			() => new Promise((_, rej) => (scheitern = /** @type {any} */ (rej)))
		);
		omniboxStore.activeStudent = { id: 'schueler-7', vorname: 'Anna', nachname: 'Müller' };
		omniboxStore.queryVal = 'B-10234';
		const laeuft = omniboxStore.submitAction(new Event('submit'));
		// Während die Anfrage hängt: Escape an der Theke.
		omniboxStore.activeStudent = null;
		scheitern(new Error('Netzwerk-Timeout: Die Anfrage hat zu lange gedauert.'));
		await laeuft;

		const q = await loadQueue();
		expect(q).toHaveLength(1);
		expect(q[0].leser_id, 'die Person vom Scan').toBe('schueler-7');
		expect(q[0].art, 'die Absicht vom Scan').toBe('ausleihe');
		expect(q[0].gescannt_am).toBeGreaterThan(0);
	});

	// Nur ein gescheiterter VERSAND gehört in die Warteschlange (OFFEN.md 2.2, Commit 2). Bis
	// dahin lag verarbeiteAktionsErgebnis im selben catch: Ein TypeError aus der Auswertung
	// einer gelungenen 200-Antwort wurde eingereiht — mit demselben Idempotenz-Schlüssel, den
	// der Server schon kannte; nach Ablauf des Caches (24 h) wäre neu gebucht worden.
	it('reiht eine gelungene, aber unauswertbare Antwort NICHT ein', async () => {
		vi.mocked(apiClient.post).mockResolvedValueOnce(
			// Eine Fremdrückgabe ohne Buch: Die Auswertung greift auf data.book.titel zu.
			/** @type {any} */ ({
				ok: true,
				json: async () => ({ type: 'rueckgabe', fremdrueckgabe: true })
			})
		);
		omniboxStore.queryVal = 'B-10234';
		await omniboxStore.submitAction(new Event('submit'));
		expect(await loadQueue(), 'eine Antwort kam an — das ist kein Netzausfall').toHaveLength(0);
		expect(omniboxStore.errorMessage, 'der Fehler wird gezeigt, nicht versteckt').toMatch(/Fehler/);
	});

	// „Buch zurückgeben" in der Akte ist eine RÜCKGABE — auch offline (OFFEN.md 2.2, Commit 3).
	// Online entscheidet der Server (das Buch liegt beim geladenen Schüler → Rückgabe); offline
	// entschied bis zum 15.09.2026 nur „Schüler geladen?", und der Knopf reihte eine Ausleihe
	// ein. Beim Nachbuchen wäre daraus eine Rückgabe geworden (Buch schon bei ihm) — beim
	// Doppelklick aber die zweite Ausleihe eine echte, und das Kind hätte das Buch wieder.
	it('„Buch zurückgeben" im Profil ist offline eine Rückgabe, auch beim Doppelklick', async () => {
		omniboxStore.activeStudent = { id: 'schueler-7', vorname: 'Anna', nachname: 'Müller' };
		await omniboxStore.gibZurueck('B-10234');
		await omniboxStore.gibZurueck('B-10234');
		const q = await loadQueue();
		expect(q.map((e) => e.art)).toEqual(['rueckgabe', 'rueckgabe']);
		expect(q.map((e) => e.leser_id)).toEqual(['schueler-7', 'schueler-7']);
	});

	// Mit geladenem Kollegen ist ein Offline-Buch eine Ausleihe an ihn (OFFEN.md 2.2,
	// Commit 4; Entscheidung (b) vom 13.09.2026). Bis dahin entschied nur activeStudent, und
	// das Buch ging mit geladener Lehrkraft als Rückgabe in die Warteschlange.
	//
	// Seit Migration 125 steht ein Kollege in demselben Platz wie ein Schüler — der Eintrag
	// trägt seine Leser-Kennung im selben Feld.
	it('reiht mit geladenem Kollegen eine Ausleihe an ihn ein', async () => {
		omniboxStore.activeStudent = {
			id: 'leser-3',
			vorname: 'Karl',
			nachname: 'Kraft',
			art: 'lehrkraft'
		};
		omniboxStore.queryVal = 'B-10234';
		await omniboxStore.submitAction(new Event('submit'));
		const q = await loadQueue();
		expect(q).toHaveLength(1);
		expect(q[0].art, 'eine Ausleihe, keine Rückgabe').toBe('ausleihe');
		expect(q[0].leser_id).toBe('leser-3');
		omniboxStore.activeStudent = null;
	});

	it('reiht mit geladenem Schüler eine Ausleihe ein, ohne ihn eine Rückgabe', async () => {
		omniboxStore.activeStudent = { id: 'schueler-7', vorname: 'Anna', nachname: 'Müller' };
		omniboxStore.queryVal = 'B-10234';
		await omniboxStore.submitAction(new Event('submit'));

		let q = await loadQueue();
		expect(q, 'der Scan wurde offline gespeichert').toHaveLength(1);
		expect(q[0].art, 'mit Schüler an der Theke ist der Scan eine Ausleihe').toBe('ausleihe');
		expect(q[0].leser_id).toBe('schueler-7');

		// Gegenprobe: ohne Schüler bleibt es eine Rückgabe.
		await leere();
		omniboxStore.activeStudent = null;
		omniboxStore.queryVal = 'B-10235';
		await omniboxStore.submitAction(new Event('submit'));
		q = await loadQueue();
		expect(q).toHaveLength(1);
		expect(q[0].art).toBe('rueckgabe');
		expect(q[0].leser_id).toBeNull();
	});

	// Jede FORM, die an der Theke wirklich ueber den Tisch geht — nicht fuenfmal `B-10234`.
	//
	// Genau daran scheiterte der Nachweis am Stack (16.09.2026): `speichereOfflineAktion`
	// nahm nur `B-` an und warf alles andere mit einem nackten „Netzwerkfehler" weg. Kein
	// Testfall dieser Datei konnte das sehen, weil jeder `B-10234` scannte.
	describe('alle Buchformen, nicht nur B-', () => {
		it('reiht ein LMF-Buch ein — auch ohne Barcode-Liste', async () => {
			omniboxStore.activeStudent = { id: 'schueler-7', vorname: 'Anna', nachname: 'Müller' };
			omniboxStore.queryVal = 'LMF-2025-0007';
			await omniboxStore.submitAction(new Event('submit'));
			const q = await loadQueue();
			expect(q).toHaveLength(1);
			expect(q[0].barcode).toBe('LMF-2025-0007');
			expect(q[0].art).toBe('ausleihe');
		});

		it('reiht eine nackte Littera-Nummer ein, die auf der Liste steht', async () => {
			omniboxStore.queryVal = '58968';
			await omniboxStore.submitAction(new Event('submit'));
			const q = await loadQueue();
			expect(q).toHaveLength(1);
			expect(q[0].barcode).toBe('58968');
		});

		it('bucht ein Littera-Etikett unter der NUMMER, nicht unter dem Strichcode', async () => {
			// Der Aufdruck lautet 58968, der Strichcode traegt die EAN-13 darum herum.
			// Der Server kennt nur die Nummer.
			omniboxStore.queryVal = '5896800039556';
			await omniboxStore.submitAction(new Event('submit'));
			const q = await loadQueue();
			expect(q).toHaveLength(1);
			expect(q[0].barcode).toBe('58968');
		});

		it('bucht einen Ausweis nicht, sondern merkt ihn sich', async () => {
			// Bis zum 16.09.2026 wurde er mit einem nackten „Netzwerkfehler" verworfen;
			// kurz darauf abgewiesen mit Begruendung; seit dem Ausweis-Merker wird er
			// gemerkt. Eine Buchung ist er in keinem Fall.
			Object.defineProperty(navigator, 'onLine', { value: false, configurable: true });
			omniboxStore.queryVal = 'A-00042';
			await omniboxStore.submitAction(new Event('submit'));
			expect(await loadQueue(), 'ein Ausweis ist keine Buchung').toHaveLength(0);
			expect(omniboxStore.offlineAusweis).toBe('A-00042');
			Object.defineProperty(navigator, 'onLine', { value: true, configurable: true });
		});

		it('sagt bei einem getippten Namen, dass es ohne Netz keine Namenssuche gibt', async () => {
			omniboxStore.queryVal = 'Mueller';
			await omniboxStore.submitAction(new Event('submit'));
			expect(await loadQueue()).toHaveLength(0);
			expect(omniboxStore.errorMessage).toMatch(/nach Namen/);
			// Ueber einen Namen von „der Buchliste" zu reden, half niemandem weiter.
			expect(omniboxStore.errorMessage).not.toMatch(/Buchliste/);
		});

		it('nennt eine unbekannte Nummer unklar und bucht sie NICHT', async () => {
			omniboxStore.queryVal = 'B97601826457';
			await omniboxStore.submitAction(new Event('submit'));
			expect(await loadQueue()).toHaveLength(0);
			expect(omniboxStore.errorMessage).toMatch(/nicht eindeutig|NICHT gebucht/);
		});
	});

	// Der ohne Netz gescannte Ausweis (Stufe 3, Commit 15).
	//
	// Die Theke merkt sich die NUMMER und sonst nichts — Personendaten liegen bewusst
	// nicht auf dem Theken-Rechner. Aufloesen kann sie nur der Server beim Nachbuchen;
	// die Tuer kennt dafuer `ausweis_barcode`.
	describe('Ausweis ohne Netz', () => {
		/** @param {boolean} an */
		const netz = (an) =>
			Object.defineProperty(navigator, 'onLine', { value: an, configurable: true });

		beforeEach(() => netz(false));
		afterEach(() => netz(true));

		it('merkt sich den Ausweis und laesst die vorher geladene Person fallen', async () => {
			// Sonst zeigte die Theke einen Namen, waehrend die folgenden Buecher an eine
			// ANDERE Person gingen.
			omniboxStore.activeStudent = { id: 'schueler-7', vorname: 'Anna' };
			omniboxStore.queryVal = 'S-10001';
			await omniboxStore.submitAction(new Event('submit'));

			expect(omniboxStore.offlineAusweis).toBe('S-10001');
			expect(omniboxStore.activeStudent).toBeNull();
			expect(await loadQueue(), 'der Ausweis selbst ist keine Buchung').toHaveLength(0);
		});

		it('schreibt die folgenden Buecher dem gemerkten Ausweis zu', async () => {
			omniboxStore.queryVal = 'S-10001';
			await omniboxStore.submitAction(new Event('submit'));
			omniboxStore.queryVal = 'LMF-2025-0007';
			await omniboxStore.submitAction(new Event('submit'));

			const q = await loadQueue();
			expect(q).toHaveLength(1);
			expect(q[0].ausweis_barcode).toBe('S-10001');
			// Ohne Person UND ohne Merker waere es eine Rueckgabe — mit Merker ist es eine
			// Ausleihe an die Person hinter der Karte.
			expect(q[0].art).toBe('ausleihe');
		});

		it('ein unklarer Scan sperrt die Zuordnung: der Merker faellt', async () => {
			omniboxStore.queryVal = 'S-10001';
			await omniboxStore.submitAction(new Event('submit'));
			omniboxStore.queryVal = 'B97601826457'; // weder Buch noch erkennbarer Ausweis
			await omniboxStore.submitAction(new Event('submit'));

			expect(omniboxStore.offlineAusweis, 'bis zum naechsten eindeutigen Ausweis').toBe('');
		});

		// DER gefaehrliche Fall: Merker steht, Verbindung ist zurueck. Der Online-Weg
		// schickte das Buch ohne Person los, und der Server liest das Schweigen als
		// RUECKGABE — aus einer Ausleihe wuerde still eine Rueckgabe.
		it('bucht bei zurueckgekehrter Verbindung NICHT weiter, sondern bittet um den Ausweis', async () => {
			omniboxStore.queryVal = 'S-10001';
			await omniboxStore.submitAction(new Event('submit'));
			expect(omniboxStore.offlineAusweis).toBe('S-10001');

			netz(true);
			omniboxStore.queryVal = 'LMF-2025-0007';
			await omniboxStore.submitAction(new Event('submit'));

			expect(await loadQueue(), 'nicht gebucht und nicht eingereiht').toHaveLength(0);
			expect(omniboxStore.errorMessage).toMatch(/noch einmal scannen/);
			expect(omniboxStore.errorMessage).toContain('S-10001');
			expect(omniboxStore.offlineAusweis, 'der Merker ist verbraucht').toBe('');
		});

		it('einen Ausweis laesst sie bei zurueckgekehrter Verbindung durch — er ist der Ausweg', async () => {
			omniboxStore.queryVal = 'S-10001';
			await omniboxStore.submitAction(new Event('submit'));

			netz(true);
			omniboxStore.queryVal = 'S-10001';
			await omniboxStore.submitAction(new Event('submit'));
			// Der Online-Weg uebernimmt (hier scheitert er im Test) — entscheidend ist,
			// dass die Bitte um den Ausweis nicht den Ausweis selbst abweist.
			expect(omniboxStore.errorMessage ?? '').not.toMatch(/noch einmal scannen/);
		});
	});
});
