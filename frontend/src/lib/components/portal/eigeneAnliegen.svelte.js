import { apiFetch } from '../../apiFetch.js';

/**
 * Die eigenen Anliegen einer Lehrkraft im Kollegiums-Portal: die Liste unter der Suche, die
 * nach jeder abgeschickten Meldung neu gelesen wird.
 *
 * Scheitert ein Abruf, bleibt der alte Stand stehen. Geleert, verschwände eine gerade
 * abgeschickte Meldung beim Nachladen wieder vom Bildschirm. Ein gescheiterter Abruf weiß
 * nichts über die Anliegen; er darf also auch nichts über sie behaupten.
 */
export function erzeugeEigeneAnliegen() {
	/** @type {any[]} */
	let liste = $state([]);
	// Beim ersten Laden gibt es keinen alten Stand, auf den man zurückfallen könnte: Die
	// Liste ist leer, weil noch nichts da war — und „leer" liest sich wie „du hast keine
	// Anliegen". Wer gestern eine Meldung geschickt hat, hält sie für verloren und schickt
	// sie noch einmal. Deshalb merkt sich der Zustand, ob je ein Abruf gelungen ist;
	// scheitert der erste, sagt das Portal es, statt „nichts da" zu zeigen.
	let jeGeladen = $state(false);
	let fehler = $state(false);

	async function lade() {
		try {
			const res = await apiFetch('/api/anliegen/eigene');
			if (!res.ok) {
				fehler = !jeGeladen;
				return;
			}
			const daten = await res.json();
			if (Array.isArray(daten)) {
				liste = daten;
				jeGeladen = true;
				fehler = false;
			}
		} catch {
			/* Netz weg: Der alte Stand bleibt stehen, der nächste Aufruf holt ihn nach. */
			fehler = !jeGeladen;
		}
	}

	return {
		get liste() {
			return liste;
		},
		/** Der erste Abruf ist gescheitert — es gibt keinen Stand, über den man etwas sagen kann. */
		get fehler() {
			return fehler;
		},
		lade
	};
}
