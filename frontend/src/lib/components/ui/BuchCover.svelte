<script>
	/**
	 * @component BuchCover — das Coverbild eines Titels, an EINER Stelle.
	 *
	 * Es gab davon fünf Größen und vier Kopien der Ausweich-Logik: `w-7`, `w-8`, `w-10`,
	 * `w-12` und `w-16` in Zeilen und Kacheln, und `BuchKarte`, `KlassenBuchKachel`,
	 * `BookTableZeile` sowie `IsbnLookupDialog` bauten jede für sich denselben
	 * Kandidaten-Durchlauf nach. Genau die Ausgangslage der Suchfelder, die in zehn
	 * Fassungen mit sieben Maßen endeten.
	 *
	 * Warum eine Komponente und kein Snippet: Ein Cover trägt Zustand JE VORKOMMEN — den
	 * Index in der Kandidatenliste und die Feststellung, dass alle Quellen versagt haben.
	 * Snippets sind reine Markup-Vorlagen ohne eigenen Zustand; sie könnten das nicht.
	 *
	 * Drei Dinge, die nicht offensichtlich sind:
	 *
	 * 1. `loading="lazy"`. Eine Bestellbedarfs-Liste hat auf dem Zielsystem 247 Zeilen;
	 *    ohne lazy wären das 247 Anfragen beim Laden. Der Browser holt nur, was sichtbar
	 *    ist. Genau diese Last war 2026 der Grund, das Cover dort ganz wegzulassen und
	 *    nur `CoverPeek` anzubieten — mit lazy entfällt der Grund.
	 * 2. Die `naturalWidth`-Prüfung. Der Cover-Proxy antwortet auf eine Adresse, die er
	 *    nicht bedienen darf, mit einem transparenten 1×1-GIF und Status 200 (bewusst,
	 *    gegen Konsolen-Spam). Ein `onerror` feuert dafür NIE. Ohne diese Prüfung stünde
	 *    im Layout ein leeres Kästchen statt des Platzhalters.
	 * 3. Der Platzhalter ist die Initiale, kein leeres Bildsymbol. Das ist die Hausfassung
	 *    (zehn Dateien) und zugleich das M3-Muster: `list-item-leading-avatar-label`.
	 *
	 * Maß: M3 gibt der Listenzeile `list-item-leading-image` mit 56 px. Diese Anwendung
	 * fährt kompaktere Dichte (siehe e2e/control-hoehen.spec.js), deshalb ist `zeile`
	 * mit 40 px die Vorgabe für Arbeitslisten; `liste` trägt das volle M3-Maß.
	 *
	 * @prop {string} [coverUrl] Wert aus `buecher_titel.cover_url`
	 * @prop {string} [isbn] Schlüssel für den Proxy und die Ausweichquellen
	 * @prop {string} [titel] Liefert die Initiale des Platzhalters und den Alternativtext
	 * @prop {'zeile'|'liste'|'gross'} [groesse]
	 * @prop {string} [klasse]
	 */
	import { coverKandidaten } from '../../utils/coverSrc.js';

	/** @type {{ coverUrl?: string, isbn?: string, titel?: string, groesse?: 'zeile'|'liste'|'gross', klasse?: string }} */
	let { coverUrl = '', isbn = '', titel = '', groesse = 'zeile', klasse = '' } = $props();

	const MASSE = {
		zeile: 'h-10', // 40 px — Arbeitslisten in kompakter Dichte
		liste: 'h-14', // 56 px — M3 list-item-leading-image
		gross: 'h-20' // 80 px — Karten und Detailflächen
	};

	/** Reihenfolge: gespeichertes Cover, dann Google Books, dann OpenLibrary — alle über
	 *  den eigenen Proxy, nie per Hotlink (siehe utils/coverSrc.js). */
	const kandidaten = $derived(coverKandidaten(coverUrl, isbn));

	let index = $state(0);
	let alleGescheitert = $state(false);

	// Schreibt nur, liest den eigenen Zustand NICHT — sonst wäre es eine Endlosschleife,
	// die im Browser die ganze Seite lahmlegt und von keinem Gate gefangen würde.
	$effect(() => {
		// `void` statt des nackten Ausdrucks: Den meldet ESLint als wirkungslos — dasselbe
		// Muster wie in OrderRecommendations. Gelesen wird nur, um die Abhaengigkeit zu
		// setzen; geschrieben wird der eigene Zustand.
		void kandidaten;
		index = 0;
		alleGescheitert = false;
	});

	const quelle = $derived(kandidaten[index] ?? '');
	const zeigeBild = $derived(quelle !== '' && !alleGescheitert);

	function naechsteQuelle() {
		if (index < kandidaten.length - 1) index += 1;
		else alleGescheitert = true;
	}

	/** @param {Event} ereignis */
	function pruefeGeladenes(ereignis) {
		const bild = /** @type {HTMLImageElement} */ (ereignis.currentTarget);
		// Das 1×1-GIF des Proxys lädt erfolgreich — erkennbar nur an seiner Größe.
		if (bild.naturalWidth < 10 || bild.naturalHeight < 10) naechsteQuelle();
	}
</script>

{#if zeigeBild}
	<img
		src={quelle}
		alt={titel ? `Cover von ${titel}` : ''}
		loading="lazy"
		decoding="async"
		onerror={naechsteQuelle}
		onload={pruefeGeladenes}
		class="{MASSE[groesse]} aspect-3/4 shrink-0 rounded-sm object-cover {klasse}"
	/>
{:else}
	<div
		class="{MASSE[
			groesse
		]} bg-surface-container-highest text-on-surface-variant flex aspect-3/4 shrink-0 items-center justify-center rounded-sm text-sm font-semibold uppercase {klasse}"
		aria-hidden="true"
	>
		{(titel || '?').charAt(0)}
	</div>
{/if}
