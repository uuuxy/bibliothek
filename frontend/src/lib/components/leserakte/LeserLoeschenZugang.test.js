import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';

// Das echte Modul bleibt, weil authStore beim Laden einen Haken darin anmeldet.
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import LeserGefahrenzone from './LeserGefahrenzone.svelte';
import LeserLoeschenDialog from './LeserLoeschenDialog.svelte';
import { authStore } from '../../stores/authStore.svelte.js';

// Gefahrenzone und Lösch-Dialog der Akte sagen beim Kollegium, dass mit dem Eintrag der Zugang
// zu „Mein Portal" erlischt. Praktikum und Fachbereich haben nie einen (Migration 153), eine aus
// Littera übernommene Lehrkraft nur ein Konto mit Ersatzadresse, eine Lehrkraft keinen mehr, wenn
// die Benutzerverwaltung ihr Konto gelöscht hat. Bei ihnen wäre der Satz falsch. Ob es einen
// gibt, sagt die Akte in `hat_zugang` (api/leser/leserakte.go), nicht die Adresse am Konto.

const lehrkraft = {
	vorname: 'Lena',
	nachname: 'Lehr',
	art: 'lehrkraft',
	email: 'lena@schule.invalid',
	hat_zugang: true
};
const praktikum = {
	vorname: 'Paul',
	nachname: 'Praktikum',
	art: 'praktikum',
	email: '',
	hat_zugang: false
};
const ausLittera = {
	vorname: 'Lutz',
	nachname: 'Littera',
	art: 'lehrkraft',
	email: 'littera-4711@littera.invalid',
	hat_zugang: false
};
const schueler = {
	vorname: 'Mia',
	nachname: 'Muster',
	art: 'schueler',
	klasse: '07A',
	hat_zugang: false
};

const admin = { rolle: 'admin', permissions: [] };
const mitarbeiter = { rolle: 'mitarbeiter', permissions: ['view_students', 'delete_students'] };

beforeEach(() => {
	authStore.currentUser = admin;
});

/** @param {any} profile */
const zone = (profile) =>
	render(LeserGefahrenzone, { props: { leser: profile, onDelete: () => {} } }).container
		.textContent ?? '';

// Nur der Text des Dialogs selbst — im selben Test steht die Zone schon im Dokument.
/** @param {any} profile */
const dialog = (profile) =>
	render(LeserLoeschenDialog, {
		props: { open: true, leser: profile, onclose: () => {}, onsuccess: () => {} }
	}).getByRole('dialog').textContent ?? '';

// Derselbe Text mit einfachen Leerzeichen: Wo die Vorlage umbricht, entscheidet Prettier.
/** @param {any} profile */
const saetze = (profile) => dialog(profile).replace(/\s+/g, ' ');

// Der Knopf ruft DELETE /api/schueler/{id}: Der Leser kommt in den Papierkorb und bleibt
// wiederherstellbar. Nach 180 Tagen anonymisiert der Nachtlauf einen Schüler und löscht einen
// Kollegen endgültig (repository.AnonymisierungPapierkorbTage). „Bis zu", weil ein
// Ehemaliger im Papierkorb schon mit dem Ende seiner Karenzzeit anonymisiert wird.
describe('Löschen in der Akte: der Dialog sagt, was geschieht', () => {
	it('Schüler: Papierkorb, bis zu 180 Tage wiederherstellbar, danach anonymisiert', () => {
		expect(saetze(schueler)).toContain(
			'Mia Muster kommt in den Papierkorb der Leserdatei und lässt sich dort bis zu 180 Tage ' +
				'lang wiederherstellen. Danach wird der Eintrag anonymisiert.'
		);
	});

	it('Kollegium: danach endgültig gelöscht, nicht anonymisiert', () => {
		const text = saetze(lehrkraft);
		expect(text).toContain(
			'Lena Lehr kommt in den Papierkorb der Leserdatei und lässt sich dort bis zu 180 Tage ' +
				'lang wiederherstellen. Danach wird der Eintrag endgültig gelöscht.'
		);
		expect(text).not.toMatch(/anonymisiert/);
	});

	it.each([
		['Schüler', schueler],
		['Kollegium', lehrkraft]
	])('%s: verspricht nichts Endgültiges', (_wer, profile) => {
		const text = saetze(profile);
		expect(text).not.toMatch(/Ausleihen werden anonymisiert/);
		expect(text).not.toMatch(/nicht rückgängig/);
		expect(text).not.toMatch(/Endgültig archivieren/);
	});

	it('der bestätigende Knopf sagt, was er tut', () => {
		const screen = render(LeserLoeschenDialog, {
			props: { open: true, leser: schueler, onclose: () => {}, onsuccess: () => {} }
		});
		expect(screen.getByRole('button', { name: 'In den Papierkorb' })).toBeTruthy();
	});
});

describe('Löschen in der Akte: vom Zugang nur, wenn es einen gibt', () => {
	it('Lehrkraft mit Konto: Zone und Dialog nennen den Zugang', () => {
		expect(zone(lehrkraft)).toMatch(/mit ihr den Zugang zu „Mein\s+Portal“/);
		expect(dialog(lehrkraft)).toMatch(/Der Zugang zu „Mein Portal“ erlischt dabei/);
	});

	it('Praktikum ohne Konto: kein Wort vom Zugang, aber der Kollegen-Knopf', () => {
		const text = zone(praktikum);
		expect(text).not.toMatch(/Zugang/);
		expect(text).toMatch(/entfernt den Eintrag/);
		expect(text).toMatch(/Kollegen archivieren/);
		expect(dialog(praktikum)).not.toMatch(/Zugang/);
	});

	it('Schüler: der bisherige Satz, ohne Zugang', () => {
		const text = zone(schueler);
		expect(text).toMatch(/Das Löschen dieses Schülerprofils/);
		expect(text).not.toMatch(/Zugang/);
	});

	// Mit der Ersatzadresse meldet sich niemand an; die Akte zeigt die Adresse trotzdem.
	it('aus Littera übernommen: kein Wort vom Zugang, obwohl ein Konto da ist', () => {
		expect(zone(ausLittera)).not.toMatch(/Zugang/);
		expect(dialog(ausLittera)).not.toMatch(/Zugang/);
	});
});

// Einen Zugang löscht nur, wer auch Konten verwaltet (manage_users), wie in der
// Benutzerverwaltung; der Server lehnt sonst ab (api/leser_loeschen_zugang_pg_test.go). Die Zone
// zeigt den Knopf abgeschaltet und den Grund, statt ihn bis zur Ablehnung anzubieten.
describe('Löschen in der Akte: einen Zugang löscht nur, wer Konten verwaltet', () => {
	/** @param {any} profile */
	const knopf = (profile) =>
		/** @type {HTMLButtonElement} */ (
			render(LeserGefahrenzone, { props: { leser: profile, onDelete: () => {} } }).getByRole(
				'button',
				{ name: /archivieren \/ löschen/ }
			)
		);

	it('ohne manage_users: Knopf abgeschaltet, der Grund steht daneben', () => {
		authStore.currentUser = mitarbeiter;
		expect(knopf(lehrkraft).disabled).toBe(true);
		expect(zone(lehrkraft).replace(/\s+/g, ' ')).toContain(
			'Diese Person hat einen Zugang zu „Mein Portal“. Löschen kann sie nur, wer „Benutzer & ' +
				'Rechte verwalten“ darf.'
		);
	});

	it('mit manage_users: Knopf offen, der Satz vom erlöschenden Zugang', () => {
		authStore.currentUser = {
			...mitarbeiter,
			permissions: [...mitarbeiter.permissions, 'manage_users']
		};
		expect(knopf(lehrkraft).disabled).toBe(false);
		expect(zone(lehrkraft)).toMatch(/mit ihr den Zugang zu „Mein\s+Portal“/);
	});

	it.each([
		['Praktikum', praktikum],
		['aus Littera übernommen', ausLittera],
		['Schüler', schueler]
	])('%s: ohne Zugang genügt das Recht zum Löschen', (_wer, profile) => {
		authStore.currentUser = mitarbeiter;
		expect(knopf(profile).disabled).toBe(false);
		expect(zone(profile)).not.toMatch(/Benutzer & Rechte/);
	});
});
