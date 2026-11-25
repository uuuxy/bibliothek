import { apiClient, apiFetch, leseApiFehler } from '../../apiFetch.js';
import { snackbars } from '../../stores/snackbars.svelte.js';
import { formatEuro } from '../../utils/format.js';

/**
 * Die Geräteliste (Laptops, Tablets) — eigene Datei, weil GeraeteVerwaltung an der
 * Größen-Ratsche steht (200 Zeilen) und das Laden mit dem Formular daneben nichts zu tun
 * hat. Dieselbe Auslagerung wie schuelerSuche, klassenVorschlaege und eigeneAnliegen.
 *
 * „Noch keine Geräte erfasst" ist eine Aussage über den Schrank. Nach einem gescheiterten
 * Abruf stünde derselbe Satz da, mit der Einladung, Geräte ein zweites Mal anzulegen, die
 * längst im Bestand stehen; deshalb hat die Liste einen eigenen Fehlerzustand.
 */
export function erzeugeGeraeteListe() {
	/** @type {any[]} */
	let liste = $state([]);
	let laedt = $state(true);
	let ladefehler = $state('');

	async function lade() {
		laedt = true;
		try {
			const res = await apiFetch('/api/geraete');
			if (!res.ok) throw new Error(await leseApiFehler(res));
			const daten = await res.json();
			liste = Array.isArray(daten) ? daten : [];
			ladefehler = '';
		} catch (err) {
			// Leer heißt leer — ein Ladefehler heißt Ladefehler.
			liste = [];
			ladefehler =
				err instanceof Error && err.message
					? err.message
					: 'Die Geräte konnten nicht geladen werden';
		} finally {
			laedt = false;
		}
	}

	/**
	 * „Defekt melden“ und „Wieder freigeben“ schalten nur das Kennzeichen um: Die Zeile der Liste
	 * kann älter sein als das Gerät. An einem verlorenen Gerät holt „Wieder freigeben“ es zurück;
	 * der Server storniert dann die Forderung wie beim Buch und sagt es.
	 * @param {{ id: string, ist_ausleihbar: boolean }} g
	 */
	async function schalteAusleihbar(g) {
		try {
			const res = await apiClient.put(`/api/geraete/${g.id}`, {
				ist_ausleihbar: !g.ist_ausleihbar
			});
			if (!res.ok) {
				snackbars.addToast(await leseApiFehler(res, 'Statuswechsel fehlgeschlagen'), 'error');
				return;
			}
			const antwort = await res.json().catch(() => null);
			if (antwort?.zurueckgeholt) snackbars.addToast(zurueckSatz(antwort), 'success');
			lade();
		} catch {
			snackbars.addToast('Netzwerkfehler', 'error');
		}
	}

	return /** @satisfies {object} */ ({
		get liste() {
			return liste;
		},
		get laedt() {
			return laedt;
		},
		get ladefehler() {
			return ladefehler;
		},
		lade,
		schalteAusleihbar
	});
}

/**
 * Der Satz zu einem zurückgeholten Gerät, mit der stornierten Forderung, wenn es eine gab.
 * @param {{ stornierte_forderungen?: number, stornierter_betrag?: number }} antwort
 */
function zurueckSatz(antwort) {
	const n = antwort.stornierte_forderungen ?? 0;
	const betrag = formatEuro(antwort.stornierter_betrag);
	if (n === 0) return 'Das Gerät ist wieder im Schrank';
	if (n === 1) return `Das Gerät ist wieder im Schrank, die Forderung über ${betrag} ist storniert`;
	return `Das Gerät ist wieder im Schrank, ${n} Forderungen über zusammen ${betrag} sind storniert`;
}
