import { describe, it, expect, vi, afterEach } from 'vitest';
import { backupMessage, backupHint } from './backupStatusText.js';

/**
 * Der Backup-Wächter ist die einzige Stelle, an der jemand bemerkt, dass die
 * nächtliche Sicherung ausbleibt — und das ist genau der Fall, in dem er stimmen muss.
 * Die Textaufbereitung hatte bisher keinen Test; diese Tests kamen beim Auflösen der
 * verschachtelten Ternäroperation dazu (sonarjs S3358/S4624) und sichern sie ab.
 */

/** @param {string} iso @param {'ok'|'warning'|'critical'} status */
const status = (iso, status = 'ok') => ({
	last_backup_at: iso,
	encryption_key_set: true,
	status
});

afterEach(() => {
	vi.useRealTimers();
});

describe('backupMessage', () => {
	it('meldet nichts ohne Status', () => {
		expect(backupMessage(null)).toBe('');
	});

	it('nennt den fehlenden Schlüssel zuerst — ohne ihn läuft gar kein Backup', () => {
		expect(
			backupMessage({ last_backup_at: null, encryption_key_set: false, status: 'critical' })
		).toBe('Backup-Verschlüsselungs-Key fehlt');
	});

	it('unterscheidet „noch nie" von „veraltet"', () => {
		expect(
			backupMessage({ last_backup_at: null, encryption_key_set: true, status: 'critical' })
		).toBe('Noch kein Backup vorhanden');
	});

	it('gibt einen unlesbaren Zeitpunkt zu, statt ihn zu verschlucken', () => {
		expect(backupMessage(status('kein Datum'))).toBe('Backup-Zeitpunkt unlesbar');
	});

	it('sagt „heute" für eine Sicherung vom selben Tag', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-07-30T18:00:00'));
		expect(backupMessage(status('2026-07-30T02:30:00'))).toMatch(
			/^Letztes Backup: heute \d{2}:\d{2}$/
		);
	});

	it('nennt bei älteren Sicherungen das Datum', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-07-30T18:00:00'));
		const got = backupMessage(status('2026-07-28T02:30:00'));
		expect(got).toContain('Letztes Backup: ');
		expect(got).not.toContain('heute');
	});

	// Die drei folgenden Fälle sind die aufgelöste Ternäroperation. Genau hier stand
	// vorher ein verschachtelter Ausdruck samt verschachteltem Template-Literal.
	it('rechnet unter einem Tag in Stunden', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-07-30T18:00:00'));
		expect(backupMessage(status('2026-07-30T13:00:00', 'warning'))).toBe(
			'Letztes Backup vor 5 Stunden'
		);
	});

	it('sagt bei genau einem Tag „über 1 Tag", nicht „1 Tagen"', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-07-30T18:00:00'));
		expect(backupMessage(status('2026-07-29T12:00:00', 'warning'))).toBe(
			'Seit über 1 Tag kein Backup'
		);
	});

	it('zählt ab zwei Tagen in Tagen', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-07-30T18:00:00'));
		expect(backupMessage(status('2026-07-27T12:00:00', 'critical'))).toBe(
			'Seit 3 Tagen kein Backup'
		);
	});
});

describe('backupHint', () => {
	it('gibt zu jeder Störung einen nächsten Schritt', () => {
		const faelle = [
			{
				last_backup_at: null,
				encryption_key_set: false,
				status: /** @type {const} */ ('critical')
			},
			{ last_backup_at: null, encryption_key_set: true, status: /** @type {const} */ ('critical') },
			status('2026-07-01T02:00:00', 'warning')
		];
		for (const f of faelle) {
			expect(backupHint(f)).toContain('Einstellungen');
		}
	});

	it('schweigt ohne Status', () => {
		expect(backupHint(null)).toBe('');
	});
});
