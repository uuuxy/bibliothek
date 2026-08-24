import { apiFetch } from './apiFetch.js';
import { toastStore } from './stores/toastStore.svelte.js';

/**
 * Holt den Schüler-Etikettenbogen als PDF und öffnet ihn.
 *
 * EIN Weg für beide Aufrufer: den Muster-Testdruck des Ausweis-Designers und den
 * Stapeldruck der Schülerdatei. Der Testdruck soll zeigen, was der echte Bogen zeigt —
 * ein eigener Abrufpfad für das Muster wäre genau die zweite Wahrheit, an der im
 * Ausweis-Designer schon Leinwand und Druck auseinandergelaufen sind.
 *
 * Erzeugt wird serverseitig (POST /api/print/schueler-etiketten), wie bei den
 * Buch-Etiketten: Millimetergenaue Klebebögen über die Druck-CSS des Browsers zu
 * treffen, hat noch nie zuverlässig funktioniert — der Skalierungsfaktor im
 * Druckdialog reicht schon, um jedes Etikett danebenzusetzen.
 *
 * @param {{ formatId: string, startPosition?: number, schuelerIds?: string[], muster?: boolean }} auftrag
 * @returns {Promise<boolean>} true, wenn der Bogen geöffnet wurde
 */
export async function oeffneEtikettenbogen({
	formatId,
	startPosition = 1,
	schuelerIds = [],
	muster = false
}) {
	try {
		const res = await apiFetch('/api/print/schueler-etiketten', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ formatId, startPosition, schuelerIds, muster })
		});

		if (!res.ok) {
			// Die Meldung des Servers durchreichen statt sie durch ein allgemeines
			// „Fehler beim Drucken" zu ersetzen: Sie nennt den Grund, den die Theke
			// beheben kann (Startposition außerhalb des Bogens, zu viele Markierte).
			const daten = await res.json().catch(() => null);
			toastStore.addToast(daten?.error || 'Etikettenbogen konnte nicht erzeugt werden.', 'error');
			return false;
		}

		const url = URL.createObjectURL(await res.blob());
		const fenster = window.open(url, '_blank');
		if (!fenster) {
			toastStore.addToast('Der Etikettenbogen wurde vom Browser blockiert (Pop-up).', 'error');
			URL.revokeObjectURL(url);
			return false;
		}
		// Nicht sofort freigeben — das neue Fenster lädt die Adresse erst noch. Eine
		// Minute reicht für jeden Ladevorgang und lässt den Speicher nicht liegen.
		setTimeout(() => URL.revokeObjectURL(url), 60_000);
		return true;
	} catch {
		toastStore.addToast('Netzwerkfehler beim Erzeugen des Etikettenbogens.', 'error');
		return false;
	}
}
