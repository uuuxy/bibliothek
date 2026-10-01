import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('./apiFetch.js', () => ({ apiFetch: vi.fn() }));

import StudentDangerZone from './StudentDangerZone.svelte';
import StudentProfileDeleteModal from './StudentProfileDeleteModal.svelte';

// Gefahrenzone und Lösch-Dialog der Akte sagen beim Kollegium, dass mit dem Eintrag der Zugang
// zu „Mein Portal" erlischt. Praktikum und Fachbereich haben nie einen (Migration 153), eine
// Lehrkraft nicht mehr, wenn die Benutzerverwaltung ihr Konto gelöscht hat. Bei ihnen wäre der
// Satz falsch. `email` im Profil ist die Adresse am Konto; leer heißt kein Konto.

const lehrkraft = {
	vorname: 'Lena',
	nachname: 'Lehr',
	art: 'lehrkraft',
	email: 'lena@schule.invalid'
};
const praktikum = { vorname: 'Paul', nachname: 'Praktikum', art: 'praktikum', email: '' };
const schueler = { vorname: 'Mia', nachname: 'Muster', art: 'schueler', klasse: '07A' };

/** @param {any} profile */
const zone = (profile) =>
	render(StudentDangerZone, { props: { profile, onDelete: () => {} } }).container.textContent ?? '';

// Nur der Text des Dialogs selbst — im selben Test steht die Zone schon im Dokument.
/** @param {any} profile */
const dialog = (profile) =>
	render(StudentProfileDeleteModal, {
		props: { open: true, profile, onclose: () => {}, onsuccess: () => {} }
	}).getByRole('dialog').textContent ?? '';

// Derselbe Text mit einfachen Leerzeichen: Wo die Vorlage umbricht, entscheidet Prettier.
/** @param {any} profile */
const saetze = (profile) => dialog(profile).replace(/\s+/g, ' ');

// Der Knopf ruft DELETE /api/schueler/{id}: Der Leser kommt in den Papierkorb und bleibt
// wiederherstellbar. Nach 180 Tagen anonymisiert der Nachtlauf einen Schüler und löscht einen
// Kollegen endgültig (repository.StandardAnonymisierungSoftDeleteTage). „Bis zu", weil ein
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
		const screen = render(StudentProfileDeleteModal, {
			props: { open: true, profile: schueler, onclose: () => {}, onsuccess: () => {} }
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
});
