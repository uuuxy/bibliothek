/**
 * Textaufbereitung für den Backup-Wächter — bewusst ein reines Modul ohne Reaktivität:
 * Meldung und Handlungsanweisung sind eine Funktion des Status, nichts weiter. Damit
 * bleiben die Date-Instanzen hier lokal und kurzlebig (im .svelte.js-Store würden sie
 * gegen svelte/prefer-svelte-reactivity laufen), und die Formatierung ist für sich testbar.
 *
 * @typedef {{ last_backup_at: string | null, encryption_key_set: boolean, status: 'ok'|'warning'|'critical' }} BackupStatus
 */

/**
 * Kurzmeldung: im ok-Fall der Zeitpunkt, sonst was fehlt.
 * @param {BackupStatus | null} s
 * @returns {string}
 */
export function backupMessage(s) {
	if (!s) return '';
	if (!s.encryption_key_set) return 'Backup-Verschlüsselungs-Key fehlt';
	if (!s.last_backup_at) return 'Noch kein Backup vorhanden';

	const lastMs = Date.parse(s.last_backup_at);
	if (Number.isNaN(lastMs)) return 'Backup-Zeitpunkt unlesbar';
	const last = new Date(lastMs);

	if (s.status === 'ok') {
		const zeit = last.toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' });
		const heute = new Date().toDateString() === last.toDateString();
		return heute
			? `Letztes Backup: heute ${zeit}`
			: `Letztes Backup: ${last.toLocaleDateString('de-DE')} ${zeit}`;
	}

	const ageH = (Date.now() - lastMs) / 3600000;
	const tage = Math.floor(ageH / 24);
	if (tage < 1) {
		return `Letztes Backup vor ${Math.round(ageH)} Stunden`;
	}
	const dauer = tage === 1 ? 'über 1 Tag' : `${tage} Tagen`;
	return `Seit ${dauer} kein Backup`;
}

/**
 * Handlungsanweisung. Ein Alert ohne Weg zur Behebung ist eine Sackgasse — deshalb
 * trägt jede Meldung ihren eigenen nächsten Schritt.
 * @param {BackupStatus | null} s
 * @returns {string}
 */
export function backupHint(s) {
	if (!s) return '';
	if (!s.encryption_key_set)
		return 'Ohne Schlüssel überspringt der nächtliche Job jedes Backup. Schlüssel in den Einstellungen unter Datenverwaltung hinterlegen.';
	if (!s.last_backup_at)
		return 'Es liegt noch kein Sicherungsstand vor. Backup in den Einstellungen unter Datenverwaltung anstoßen.';
	return 'Der nächtliche Job läuft nicht wie erwartet. Stand und Konfiguration in den Einstellungen unter Datenverwaltung prüfen.';
}
