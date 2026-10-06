import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('./apiFetch.js', () => ({
	apiClient: { patch: vi.fn() }
}));

import { apiClient } from './apiFetch.js';
import { useStudentEditForm } from './useStudentEditForm.svelte.js';

const patchMock = vi.mocked(apiClient.patch);

/**
 * Ein geräumtes Feld muss den Server als LEERER STRING erreichen, nicht als null.
 *
 * Der Hintergrund steht im Backend (api/student_update.go): Die Stammdatenfelder sind
 * dort *string, und JSON-null bedeutet nil — "nicht mitgeschickt, Spalte in Ruhe
 * lassen". Bis zum 23.08.2026 baute dieses Formular seine Nutzlast als
 * `strasse: formData.strasse || null`; wer eine Adresse oder die Eltern-Mail löschte,
 * bekam "Änderungen gespeichert" zu sehen, während in der Datenbank der alte Wert
 * stehen blieb. Betroffen war genau das, dessen Entfernung jemand verlangen kann.
 *
 * Der Test steht hier und nicht nur als PG-Test im Backend, weil der Rückfall auf
 * dieser Seite passiert: Ein `|| null` ist eine Zeile, die beim nächsten Refactoring
 * harmlos aussieht. Das Gegenstück (Pflichtfelder lassen sich nicht leeren) sichert
 * api/schueler_feld_leeren_pg_test.go ab.
 */
describe('useStudentEditForm.save', () => {
	beforeEach(() => vi.clearAllMocks());

	const schueler = {
		id: 'abc',
		vorname: 'Mia',
		nachname: 'Muster',
		geburtsdatum: '2012-04-05',
		klasse: '7a',
		abgaenger_jahr: 2031,
		barcode_id: 'S-1',
		strasse: 'Hauptstr',
		hausnummer: '12',
		plz: '60311',
		ort: 'Frankfurt',
		eltern_email: 'eltern@example.org'
	};

	function baueFormular() {
		const hook = useStudentEditForm({
			getStudent: () => schueler,
			onSave: () => {},
			showSnackbar: () => {}
		});
		hook.syncData();
		return hook;
	}

	it('schickt geräumte Stammdatenfelder als leeren String, nicht als null', async () => {
		patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
		const hook = baueFormular();

		for (const feld of ['strasse', 'hausnummer', 'plz', 'ort', 'eltern_email']) {
			hook.formData[feld] = '';
		}
		await hook.save();

		const [, payload] = patchMock.mock.calls[0];
		for (const feld of ['strasse', 'hausnummer', 'plz', 'ort', 'eltern_email']) {
			expect(payload[feld], `${feld} muss als '' rausgehen — null hiesse "nicht anfassen"`).toBe(
				''
			);
		}
	});

	// Ein Klassenwechsel rechnet das Abgangsjahr neu — das tut der Server, sobald eine
	// Klasse OHNE Abgangsjahr ankommt (calculateAbgaengerJahr). Bis zum 17.09.2026 kam
	// nie eine an: Dieses Formular schickte immer den geladenen Wert mit. Ein Kind, das
	// von der 7 in die 10 wechselt, behielt das Abgangsjahr des alten Jahrgangs — und
	// daran hängen die Abgängerliste, die Versetzung und die Löschuhr.
	describe('Abgangsjahr beim Klassenwechsel', () => {
		it('lässt das Abgangsjahr weg, wenn die Klasse wechselt und niemand es angefasst hat', async () => {
			patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
			const hook = baueFormular();

			hook.formData.klasse = '10a';
			await hook.save();

			const [, payload] = patchMock.mock.calls[0];
			expect(payload.klasse).toBe('10a');
			expect(
				'abgaenger_jahr' in payload,
				'der Server leitet das Jahr nur ab, wenn keines mitkommt'
			).toBe(false);
		});

		it('schickt ein von Hand gesetztes Abgangsjahr mit — auch beim Klassenwechsel', async () => {
			patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
			const hook = baueFormular();

			hook.formData.klasse = '10a';
			hook.formData.abgaenger_jahr = '2033';
			await hook.save();

			const [, payload] = patchMock.mock.calls[0];
			expect(payload.abgaenger_jahr, 'wer beides setzt, meint beides').toBe(2033);
		});

		it('lässt Klasse und Abgangsjahr weg, solange niemand sie anfasst', async () => {
			patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
			const hook = baueFormular();

			hook.formData.vorname = 'Mira';
			await hook.save();

			const [, payload] = patchMock.mock.calls[0];
			expect(payload).toEqual({ vorname: 'Mira' });
		});

		it('schickt ein von Hand geändertes Abgangsjahr allein mit', async () => {
			patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
			const hook = baueFormular();

			hook.formData.abgaenger_jahr = '2032';
			await hook.save();

			const [, payload] = patchMock.mock.calls[0];
			expect(payload).toEqual({ abgaenger_jahr: 2032 });
		});
	});

	it('schickt geleerte Pflichtfelder ebenfalls als leeren String — der Server lehnt sie ab', async () => {
		patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
		const hook = baueFormular();

		hook.formData.vorname = '';
		hook.formData.klasse = '';
		await hook.save();

		const [, payload] = patchMock.mock.calls[0];
		expect(payload.vorname).toBe('');
		expect(payload.klasse).toBe('');
	});

	// ── Ein Kollege ist kein Schüler mit leeren Feldern ──────────────────────────
	//
	// Bis zum 16.09.2026 baute save() die Nutzlast für JEDEN gleich: klasse, lusd_id,
	// abgaenger_jahr und eltern_email standen immer drin. Bei einem Kollegen waren sie
	// leer — und der leere String heisst bei den Pflichtfeldern "räum das weg", was der
	// Server mit 400 „Klasse darf nicht leer sein." beantwortet. Der Befund war
	// deshalb kein fehlendes Feld, sondern ein Formular, das gar nicht speichern KONNTE:
	// „ich kann dort aber keine adressedaten etc nachtragen."
	//
	// Der Test prüft die PAARUNG, nicht den Wert — dieselbe Bugklasse wie bei der
	// LUSD-Klasse am 14.09.2026. „Fehlt im Payload" und „steht als '' im Payload" sehen
	// im Code fast gleich aus und bedeuten im Backend das Gegenteil.
	describe('Kollege (Lehrkraft/LiV)', () => {
		const kollege = {
			id: 'k1',
			vorname: 'Katrin',
			nachname: 'Wendlandt',
			art: 'lehrkraft',
			klasse: null,
			barcode_id: null,
			abgaenger_jahr: null,
			lusd_id: null,
			geburtsdatum: null,
			strasse: null,
			hausnummer: null,
			plz: null,
			ort: null,
			eltern_email: null
		};

		function baueKollegenFormular() {
			const hook = useStudentEditForm({
				getStudent: () => kollege,
				onSave: () => {},
				showSnackbar: () => {}
			});
			hook.syncData();
			return hook;
		}

		it('lässt die Schülerfelder ganz weg, statt sie leer mitzuschicken', async () => {
			patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
			const hook = baueKollegenFormular();

			hook.formData.strasse = 'Kleegartenstr.';
			hook.formData.hausnummer = '8';
			hook.formData.plz = '61381';
			hook.formData.ort = 'Friedrichsdorf';
			await hook.save();

			const [, payload] = patchMock.mock.calls[0];
			for (const feld of ['klasse', 'abgaenger_jahr', 'lusd_id']) {
				expect(
					Object.hasOwn(payload, feld),
					`${feld} ist beim Kollegen verschlossen — leer mitgeschickt wäre es eine 400`
				).toBe(false);
			}
			// Hinaus geht, was eingetragen wurde, und sonst nichts.
			expect(payload).toEqual({
				strasse: 'Kleegartenstr.',
				hausnummer: '8',
				plz: '61381',
				ort: 'Friedrichsdorf'
			});
		});

		// Die Eltern-Adresse ist beim Kollegen nicht verschlossen: Die Maske hat für jeden
		// dieselbe Form, und ein Feld, das offen steht, muss auch ankommen.
		it('schickt eine beim Kollegen eingetragene Eltern-Adresse mit', async () => {
			patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
			const hook = baueKollegenFormular();
			hook.formData.eltern_email = 'privat@example.org';
			await hook.save();

			const [, payload] = patchMock.mock.calls[0];
			expect(payload).toEqual({ eltern_email: 'privat@example.org' });
		});

		// UMGEKEHRT am 16.09.2026 (abends), nach der Blick auf die fertige Maske: Bis
		// dahin liess das Formular die leere Ausweisnummer WEG, weil der Server jedes leere
		// Pflichtfeld mit 400 abwies. Der Preis war ein stilles No-op — wer beim Kollegen
		// eine falsch eingetragene Nummer räumte, bekam „Änderungen gespeichert" und fand
		// sie beim nächsten Öffnen wieder vor. Der Hinweis unter dem Feld („Leer lassen,
		// solange kein Ausweis gedruckt ist") war damit eine Zusage, die die Maske nicht
		// hielt. Seit die Pflicht im Server an die ART gepaart ist (pruefeAusweisLeerung,
		// api/student_schul_email.go — TestAusweisnummerLeeren belegt beide Seiten), geht
		// das leere Feld mit und leert die Spalte wirklich.
		it('schickt die geleerte Ausweisnummer beim Kollegen MIT — sonst wäre das Leeren ein stilles No-op', async () => {
			patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
			const hook = useStudentEditForm({
				getStudent: () => ({ ...kollege, barcode_id: 'L-0007' }),
				onSave: () => {},
				showSnackbar: () => {}
			});
			hook.syncData();
			hook.formData.barcode_id = '';
			await hook.save();

			const [, payload] = patchMock.mock.calls[0];
			expect(payload).toEqual({ barcode_id: '' });
		});

		// Die Schul-Adresse ist das Gegenstück: Sie gehört dem KONTO, nicht der Leserzeile,
		// und geht nur beim Kollegium mit. Beim Schüler wäre schon der leere String eine
		// Aussage, die der Server mit 400 abweist („Ein Schüler bekommt kein Konto").
		it('schickt die Schul-Adresse beim Kollegen mit', async () => {
			patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
			const hook = baueKollegenFormular();
			hook.formData.email = 'neu.kollegin@schule.example';
			await hook.save();

			const [, payload] = patchMock.mock.calls[0];
			expect(payload.email).toBe('neu.kollegin@schule.example');
		});

		it('schickt eine eingetragene Ausweisnummer sehr wohl mit', async () => {
			patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
			const hook = baueKollegenFormular();
			hook.formData.barcode_id = 'L-0007';
			await hook.save();

			const [, payload] = patchMock.mock.calls[0];
			expect(payload.barcode_id).toBe('L-0007');
		});

		it('schickt beim Schüler die geänderten Schülerfelder mit, die Schul-Adresse nie', async () => {
			patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
			const hook = baueFormular();
			hook.formData.klasse = '7b';
			hook.formData.eltern_email = 'neu@example.org';
			// Ein Schüler hat kein Konto: Schon der leere String wäre eine Aussage, die der
			// Server mit 400 abweist. Auch eine getippte Adresse geht nicht mit.
			hook.formData.email = 'kind@schule.example';
			await hook.save();

			const [, payload] = patchMock.mock.calls[0];
			expect(payload).toEqual({ klasse: '7b', eltern_email: 'neu@example.org' });
		});
	});

	// Der Server schreibt jedes Feld, das der Rumpf nennt. Schickte die Maske alle Felder mit
	// dem Stand vom Laden der Akte, schriebe sie zurück, was die Versetzung, der LUSD-Import
	// oder ein anderer Platz inzwischen geändert hat — mit der Meldung „gespeichert".
	describe('nur die geänderten Felder', () => {
		it('nennt nach einer Änderung an der Eltern-Adresse kein anderes Feld', async () => {
			patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
			const hook = baueFormular();

			hook.formData.eltern_email = 'neu@example.org';
			await hook.save();

			expect(patchMock).toHaveBeenCalledWith('/api/schueler/abc', {
				eltern_email: 'neu@example.org'
			});
		});

		it('schickt ohne Änderung nichts und schließt wie nach dem Speichern', async () => {
			const onSave = vi.fn();
			const showSnackbar = vi.fn();
			const hook = useStudentEditForm({ getStudent: () => schueler, onSave, showSnackbar });
			hook.syncData();

			await hook.save();

			expect(patchMock).not.toHaveBeenCalled();
			expect(onSave).toHaveBeenCalledTimes(1);
			expect(showSnackbar).toHaveBeenCalledWith('Änderungen gespeichert.', 'success');
		});

		it('schickt ein Feld nicht, das getippt und wieder zurückgestellt wurde', async () => {
			patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
			const hook = baueFormular();

			hook.formData.ort = 'Offenbach';
			hook.formData.ort = 'Frankfurt';
			hook.formData.plz = '63065';
			await hook.save();

			const [, payload] = patchMock.mock.calls[0];
			expect(payload).toEqual({ plz: '63065' });
		});

		it('misst nach dem Neufüllen am neuen Stand', async () => {
			patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
			let leser = { ...schueler };
			const hook = useStudentEditForm({
				getStudent: () => leser,
				onSave: () => {},
				showSnackbar: () => {}
			});
			hook.syncData();
			// Die Akte lädt neu, die Versetzung hat die Klasse inzwischen geändert.
			leser = { ...schueler, klasse: '8a', abgaenger_jahr: 2031 };
			hook.syncData();

			hook.formData.strasse = 'Nebenstr';
			await hook.save();

			const [, payload] = patchMock.mock.calls[0];
			expect(payload).toEqual({ strasse: 'Nebenstr' });
		});
	});

	it('lässt ein nie gesetztes Geburtsdatum als null durch (Altdaten bleiben speicherbar)', async () => {
		patchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
		const hook = baueFormular();

		hook.formData.geburtsdatum = '';
		await hook.save();

		const [, payload] = patchMock.mock.calls[0];
		expect(payload.geburtsdatum).toBeNull();
	});
});
