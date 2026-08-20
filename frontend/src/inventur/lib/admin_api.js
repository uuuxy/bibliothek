import { apiFetch } from '../../lib/apiFetch.js';
import { appState } from './store.svelte.js';

export async function holeBuecherListe() {
	const suchParameter = appState.searchQuery
		? `?q=${encodeURIComponent(appState.searchQuery)}`
		: '';
	const res = await apiFetch(`/api/books${suchParameter}`, {
		credentials: 'include'
	});
	if (!res.ok) {
		if (res.status === 401) {
			appState.adminAuthenticated = false;
			throw new Error('UNAUTHORIZED');
		}
		throw new Error('Fehler beim Laden der Bücher');
	}
	const json = await res.json();
	return json.data || [];
}

/**
 * Lädt EIN Buch vollständig (inkl. beschreibung/erweiterteEigenschaften).
 * Die Katalogliste ist bewusst schlank und liefert diese Felder LEER — jedes
 * Formular, das per PUT das ganze Objekt zurückschickt, MUSS hierüber befüllt
 * werden, sonst leert Speichern die Felder still (Upsert-Blanking-Bugklasse).
 * @param {string} id
 * @returns {Promise<any>} das vollständige Buch (Antwort ist das nackte Objekt)
 */
export async function holeBuchDetail(id) {
	const res = await apiFetch(`/api/books/${encodeURIComponent(id)}`, {
		credentials: 'include'
	});
	if (!res.ok) {
		if (res.status === 401) {
			appState.adminAuthenticated = false;
			throw new Error('UNAUTHORIZED');
		}
		throw new Error('Buch konnte nicht geladen werden');
	}
	return await res.json();
}

/** @param {File} datei */
export async function importiereListe(datei) {
	const formData = new FormData();
	formData.append('file', datei);
	const res = await apiFetch('/api/books/import', {
		method: 'POST',
		credentials: 'include',
		headers: {},
		body: formData
	});
	if (!res.ok) {
		const errJson = await res.json().catch(() => ({}));
		throw new Error(errJson.error || errJson.message || 'Import fehlgeschlagen');
	}
	return true;
}

/** @param {string[]} ids */
export async function loescheBuecher(ids) {
	const res = await apiFetch('/api/books', {
		method: 'DELETE',
		credentials: 'include',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ ids })
	});
	if (!res.ok) {
		const errJson = await res.json().catch(() => ({}));
		throw new Error(errJson.error || 'Löschen fehlgeschlagen');
	}
	return true;
}

export async function holeExterneCover() {
	const res = await apiFetch('/api/admin/books/external-covers', {
		credentials: 'include'
	});
	if (!res.ok) throw new Error('Externe Cover konnten nicht geladen werden');
	const json = await res.json();
	return json.data || [];
}

/** @param {string[]} ids */
export async function retryExterneCover(ids = []) {
	const res = await apiFetch('/api/admin/books/retry-covers', {
		method: 'POST',
		credentials: 'include',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ ids, limit: 300 })
	});
	if (!res.ok) throw new Error('Cover-Retry fehlgeschlagen');
	const json = await res.json();
	return json.data;
}

export async function exportiereCSV() {
	const res = await apiFetch('/api/admin/books/export', {
		method: 'GET',
		credentials: 'include'
	});
	if (!res.ok) {
		throw new Error('Export fehlgeschlagen');
	}
	const blob = await res.blob();
	const url = window.URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = url;
	a.download = `bestand_export_${new Date().toISOString().split('T')[0]}.csv`;
	document.body.appendChild(a);
	a.click();
	window.URL.revokeObjectURL(url);
	a.remove();
}
