// Die Sicherung der Offline-Scans als Datei zum Aufheben, bevor jemand den Browser leert.
import { ladeWarteschlange } from './offlineWarteschlange.js';
import { localISO } from '../utils/datum.js';

/**
 * Legt die wartenden Vorgänge als Datei zum Herunterladen an; ohne wartende Vorgänge keine.
 * Der Tag im Namen ist der Kalendertag am Arbeitsplatz (localISO): toISOString() nähme UTC und
 * schriebe zwischen Mitternacht und zwei Uhr den Vortag.
 */
export async function exportiereWarteschlange() {
	const q = await ladeWarteschlange();
	if (q.length === 0) return;
	const blob = new Blob([JSON.stringify(q, null, 2)], { type: 'application/json' });
	const url = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = url;
	a.download = `offline_scans_backup_${localISO(new Date())}.json`;
	document.body.appendChild(a);
	a.click();
	a.remove();
	URL.revokeObjectURL(url);
}
