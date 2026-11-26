import { describe, it, expect, afterEach } from 'vitest';
import { backupStatus } from './backupStatus.svelte.js';

/** @param {'ok'|'warning'|'critical'|'aus'} status @returns {any} */
const stand = (status) => ({ last_backup_at: null, encryption_key_set: true, status });

// Der Alert über dem Inhalt erscheint genau bei Handlungsbedarf: bei Warnung und kritischem
// Stand, nicht bei „ok“, nicht bei abgeschalteter eigener Sicherung und nicht vor der Antwort.
describe('backupStatus.needsAction', () => {
	afterEach(() => {
		backupStatus.data = null;
	});

	it('meldet Warnung und kritischen Stand', () => {
		for (const status of /** @type {const} */ (['warning', 'critical'])) {
			backupStatus.data = stand(status);
			expect(backupStatus.needsAction, status).toBe(true);
		}
	});

	it('schweigt bei ok, bei abgeschalteter Sicherung und ohne Antwort', () => {
		for (const status of /** @type {const} */ (['ok', 'aus'])) {
			backupStatus.data = stand(status);
			expect(backupStatus.needsAction, status).toBe(false);
		}
		backupStatus.data = null;
		expect(backupStatus.needsAction).toBe(false);
	});
});
