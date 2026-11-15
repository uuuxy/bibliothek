import { apiFetch } from '../apiFetch.js';
import { backupMessage, backupHint } from '../components/system/backupStatusText.js';

/**
 * Backup-Wächter: der nächtliche Job überspringt sich still, wenn
 * BACKUP_ENCRYPTION_KEY fehlt. Der Status wird an zwei Stellen gebraucht —
 * als ruhige Bestätigungszeile im Seitenleiste-Fuß und, bei Handlungsbedarf, als
 * Alert über dem Inhalt. Beide lesen aus diesem Store, damit der Endpunkt
 * einmal pro Sitzung abgefragt wird und nicht zweimal.
 */
export const backupStatus = new (class {
	/** @type {import('../components/system/backupStatusText.js').BackupStatus | null} */
	data = $state(null);

	#loading = false;

	/** Lädt einmalig; weitere Aufrufe sind wirkungslos. */
	async load() {
		if (this.data || this.#loading) return;
		this.#loading = true;
		try {
			await this.#hole();
		} catch {
			/* Netzfehler: kein Badge statt falscher Entwarnung */
		} finally {
			this.#loading = false;
		}
	}

	/** Lädt neu, nachdem die Sicherung in den Einstellungen umgeschaltet wurde. */
	async aktualisiere() {
		try {
			await this.#hole();
		} catch {
			/* Netzfehler: Der bisherige Stand bleibt stehen */
		}
	}

	async #hole() {
		const res = await apiFetch('/api/admin/system/backup-status');
		if (res.ok) this.data = await res.json();
	}

	/** Handlungsbedarf — der Alert erscheint nur dafür. */
	get needsAction() {
		// "aus": Die Schule hat die eigene Sicherung abgeschaltet (Einstellungen → Sicherung).
		return !!this.data && this.data.status !== 'ok' && this.data.status !== 'aus';
	}

	get message() {
		return backupMessage(this.data);
	}

	get hint() {
		return backupHint(this.data);
	}
})();
