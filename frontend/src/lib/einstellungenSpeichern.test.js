import { describe, it, expect, vi, beforeEach } from 'vitest';
import { speichereKategorie } from './einstellungenSpeichern.js';
import { apiPut } from './apiFetch.js';
import { toastStore } from './stores/toastStore.svelte.js';

// Eine Kategorie der Einstellungen schickt nur, was vom Stand beim Öffnen abweicht. Mit allen
// Feldern schriebe sie zurück, was ein anderer Platz inzwischen in derselben Kategorie
// gespeichert hat. Ohne Änderung geht keine Anfrage hinaus: Der Server lehnt einen leeren
// Rumpf ab.
vi.mock('./apiFetch.js', () => ({ apiPut: vi.fn() }));
vi.mock('./stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

const GELADEN = { lmf_stichtag: '07-31', ferien_leseclub_aktiv: false, frist_buch_tage: 21 };
/** @param {Record<string, unknown>} [anders] die Maske mit den genannten Abweichungen */
const maske = (anders = {}) => {
	const jetzt = { ...GELADEN, ...anders };
	return {
		geladen: GELADEN,
		felder: {
			lmf_stichtag: jetzt.lmf_stichtag,
			ferien_leseclub_aktiv: jetzt.ferien_leseclub_aktiv
		},
		zahlen: [
			{ schluessel: 'frist_buch_tage', label: 'Tage / Buch', wert: jetzt.frist_buch_tage, min: 1 }
		]
	};
};

describe('speichereKategorie', () => {
	beforeEach(() => {
		vi.mocked(apiPut).mockReset();
		vi.mocked(toastStore.addToast).mockReset();
	});

	it('schickt nur das geänderte Feld', async () => {
		expect(await speichereKategorie(maske({ frist_buch_tage: 28 }))).toBe(true);
		expect(apiPut).toHaveBeenCalledExactlyOnceWith('/api/einstellungen', { frist_buch_tage: 28 });
	});

	it('schickt einen abgewählten Schalter und einen geleerten Text ausdrücklich', async () => {
		const geladen = { ...GELADEN, ferien_leseclub_aktiv: true };
		await speichereKategorie({
			...maske({ lmf_stichtag: '' }),
			geladen
		});
		expect(apiPut).toHaveBeenCalledExactlyOnceWith('/api/einstellungen', {
			lmf_stichtag: '',
			ferien_leseclub_aktiv: false
		});
	});

	it('schickt ohne Änderung nichts und meldet trotzdem den Stand als gespeichert', async () => {
		const onSaved = vi.fn();
		expect(await speichereKategorie({ ...maske(), onSaved })).toBe(true);
		expect(apiPut).not.toHaveBeenCalled();
		expect(toastStore.addToast).toHaveBeenCalledExactlyOnceWith('Gespeichert.', 'success');
		expect(onSaved).toHaveBeenCalledOnce();
	});

	it('prüft ein Zahlenfeld auch dann, wenn ein anderes Feld geändert wurde', async () => {
		const eingabe = maske({ lmf_stichtag: '08-15', frist_buch_tage: '' });
		expect(await speichereKategorie(eingabe)).toBe(false);
		expect(apiPut).not.toHaveBeenCalled();
		expect(toastStore.addToast).toHaveBeenCalledExactlyOnceWith(
			expect.stringContaining('Tage / Buch'),
			'warning'
		);
	});

	it('meldet keinen Erfolg, wenn die Anfrage scheitert', async () => {
		vi.mocked(apiPut).mockRejectedValue(new Error('abgelehnt'));
		const onSaved = vi.fn();
		expect(await speichereKategorie({ ...maske({ frist_buch_tage: 28 }), onSaved })).toBe(false);
		expect(toastStore.addToast).not.toHaveBeenCalled();
		expect(onSaved).not.toHaveBeenCalled();
	});
});
