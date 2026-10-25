// stores/programmstand.svelte.js — läuft in diesem Fenster die Oberfläche, die der Server
// ausliefert?
//
// Nach einem Update arbeitet ein offenes Fenster sonst mit dem alten Programm gegen den neuen
// Server weiter und liest eine geänderte Antwort falsch, ohne Meldung. Der Stand ist der Name der
// Programmdatei, mit der die Startseite die Oberfläche lädt (/assets/index-….js); der Server nennt
// seinen in jeder Antwort (pkg/oberflaechenstand). Weicht er ab, steht über dem Inhalt der
// Hinweis „Neu laden" (NeueFassung.svelte).
//
// Von selbst lädt ein Fenster nur neu, wenn darin noch niemand etwas bedient hat: Die Theke darf
// sich nicht mitten in einer Ausleihe erneuern und eine Maske nicht mit ihren Eingaben.

import { abonniere } from '../liveEvents.js';

export const KOPF_OBERFLAECHENSTAND = 'X-Oberflaechenstand';

export class Programmstand {
	/** Mit diesem Stand wurde das Fenster geladen; leer im Entwicklungsserver. */
	eigener = $state('');
	/** Diesen Stand nannte der Server zuletzt. */
	server = $state('');
	#ladeNeu;

	/** @param {() => void} [ladeNeu] */
	constructor(ladeNeu = () => location.reload()) {
		this.#ladeNeu = ladeNeu;
	}

	get veraltet() {
		return this.eigener !== '' && this.server !== '' && this.server !== this.eigener;
	}

	/** @param {string} eigener Name der Programmdatei dieses Fensters (main.js) */
	starte(eigener) {
		this.eigener = eigener;
		if (!eigener) return;
		// Nach einem Update baut sich die Live-Leitung auf, sobald der Server wieder da ist. Gefragt
		// wird dann gleich und nicht erst mit der nächsten Anfrage, denn die könnte schon ein Scan sein.
		abonniere('connected', () => void this.frageServer());
	}

	/** @param {string | null | undefined} stand */
	merke(stand) {
		if (stand) this.server = stand;
	}

	/** @param {Response} res eine Antwort des Servers (apiFetch) */
	merkeAus(res) {
		this.merke(res.headers?.get(KOPF_OBERFLAECHENSTAND));
	}

	async frageServer() {
		try {
			this.merkeAus(await fetch('/health', { cache: 'no-store' }));
		} catch {
			// Ohne Antwort bleibt der bekannte Stand; die nächste Anfrage nennt ihn.
		}
	}

	/**
	 * Ein neuer Service Worker ist aktiv (main.js, registerSW). Ohne diese Stelle lädt
	 * vite-plugin-pwa jedes offene Fenster sofort neu, auch wenn ein anderes Fenster desselben
	 * Browsers das Update geholt hat. Kennt der Browser `userActivation` nicht, gilt das Fenster
	 * als bedient.
	 */
	neuerServiceWorker() {
		if (navigator.userActivation?.hasBeenActive === false) this.#ladeNeu();
	}

	ladeNeu() {
		this.#ladeNeu();
	}
}

export const programmstand = new Programmstand();
