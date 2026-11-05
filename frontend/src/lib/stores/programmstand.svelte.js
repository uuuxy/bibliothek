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

/** So lange wartet „Neu laden" höchstens auf den neuen Service Worker. Danach lädt es trotzdem
 *  neu: Ein hängender Knopf wäre schlimmer als ein zweiter Klick. */
export const WARTEZEIT_NEUE_FASSUNG_MS = 30_000;

export class Programmstand {
	/** Mit diesem Stand wurde das Fenster geladen; leer im Entwicklungsserver. */
	eigener = $state('');
	/** Diesen Stand nannte der Server zuletzt. */
	server = $state('');
	/** „Neu laden" wartet, bis der Service Worker das neue Programm geholt hat. */
	laedt = $state(false);
	#ladeNeu;
	#holeNeueFassung;

	/**
	 * @param {() => void} [ladeNeu]
	 * @param {() => Promise<void>} [holeNeueFassung] erfüllt, sobald ein Neuladen das neue Programm bekommt
	 */
	constructor(ladeNeu = () => location.reload(), holeNeueFassung = holeNeuenServiceWorker) {
		this.#ladeNeu = ladeNeu;
		this.#holeNeueFassung = holeNeueFassung;
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

	/**
	 * Der Knopf „Neu laden". Sofort neu geladen, bekäme die Seite die alte Kopie aus dem Service
	 * Worker, solange der neue sich noch einrichtet, und der Hinweis stünde wieder da. Deshalb erst
	 * die neue Fassung holen und dann genau einmal neu laden.
	 */
	async ladeNeu() {
		if (this.laedt) return;
		this.laedt = true;
		/** @type {ReturnType<typeof setTimeout> | undefined} */
		let zeitgeber;
		const frist = new Promise((fertig) => {
			zeitgeber = setTimeout(fertig, WARTEZEIT_NEUE_FASSUNG_MS);
		});
		try {
			await Promise.race([this.#holeNeueFassung(), frist]);
		} catch {
			// Ohne neue Fassung lädt die Seite trotzdem neu; schlimmstenfalls steht der Hinweis wieder da.
		} finally {
			clearTimeout(zeitgeber);
		}
		this.#ladeNeu();
		// Hält die Rückfrage zu ungespeicherten Eingaben das Neuladen an (verlassenSchutz,
		// offlineSync), steht der Knopf wieder bereit.
		this.laedt = false;
	}
}

/**
 * Lässt den Browser nach dem Service Worker des Servers fragen und wartet, bis ein neuer aktiv
 * ist. Ohne Service Worker oder ohne neuen gibt es nichts abzuwarten.
 * @returns {Promise<void>}
 */
async function holeNeuenServiceWorker() {
	const registrierung = await navigator.serviceWorker?.getRegistration();
	if (!registrierung) return;
	await registrierung.update();
	const neu = registrierung.installing ?? registrierung.waiting;
	if (!neu) return;
	await new Promise((fertig) => {
		const pruefe = () => {
			if (neu.state === 'activated' || neu.state === 'redundant') fertig(undefined);
		};
		neu.addEventListener('statechange', pruefe);
		pruefe();
	});
}

export const programmstand = new Programmstand();
