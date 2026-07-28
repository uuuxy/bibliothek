import { apiFetch } from '../apiFetch.js';
import { backupMessage, backupHint } from '../backupStatusText.js';

/**
 * Backup-Wächter: der nächtliche Job überspringt sich still, wenn
 * BACKUP_ENCRYPTION_KEY fehlt. Der Status wird an zwei Stellen gebraucht —
 * als ruhige Bestätigungszeile im Sidebar-Fuß und, bei Handlungsbedarf, als
 * Alert über dem Inhalt. Beide lesen aus diesem Store, damit der Endpunkt
 * einmal pro Sitzung abgefragt wird und nicht zweimal.
 */
export const backupStatus = new (class {
	/** @type {import('../backupStatusText.js').BackupStatus | null} */
	data = $state(null);

	#loading = false;

	/** Lädt einmalig; weitere Aufrufe sind wirkungslos. */
	async load() {
		if (this.data || this.#loading) return;
		this.#loading = true;
		try {
			const res = await apiFetch('/api/admin/system/backup-status');
			if (res.ok) this.data = await res.json();
		} catch {
			/* Netzfehler: kein Badge statt falscher Entwarnung */
		} finally {
			this.#loading = false;
		}
	}

	/** Handlungsbedarf — der Alert erscheint nur dafür. */
	get needsAction() {
		return !!this.data && this.data.status !== 'ok';
	}

	get message() {
		return backupMessage(this.data);
	}

	get hint() {
		return backupHint(this.data);
	}
})();
