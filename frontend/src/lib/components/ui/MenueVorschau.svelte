<!-- @component MenueVorschau — eine Fläche neben dem markierten Eintrag einer Menüfläche
     (SelectListe), die zeigt, was hinter dem Eintrag steht: die Cover unter einer Signatur.

     Kein Tooltip, denn M3 sieht darin kein Bild vor („They can optionally contain a subhead,
     buttons, and hyperlinks“). Sie geht auf wie ein Untermenü: neben dem Eintrag, oben bündig,
     nach der Wartezeit von material-web (md-sub-menu, hover-open-delay 400 ms). So fragt ein
     Zeiger, der über die Liste fährt, nicht jede Zeile ab. Sie folgt der Markierung, also Maus
     und Pfeiltasten, und lässt sich nicht bedienen. -->
<script>
	import { berechneVorschauBox } from './menueGeometrie.js';
	import { folgeLage } from './lageFolgen.js';

	/**
	 * liste: die Menüfläche; aktiv: Index des markierten Eintrags (data-i); inhalt zeichnet die
	 * Vorschau zum Wert und darf leer bleiben, dann bleibt auch die Fläche unsichtbar.
	 * @type {{
	 *   liste: HTMLElement | undefined,
	 *   aktiv: number,
	 *   wert: any,
	 *   inhalt: import('svelte').Snippet<[any]>
	 * }}
	 */
	let { liste, aktiv, wert, inhalt } = $props();

	const WARTEZEIT = 400;

	let gezeigt = $state(/** @type {{ wert: any } | null} */ (null));
	let breite = $state(0);
	let hoehe = $state(0);
	let lage = $state({ left: 0, top: 0 });

	// Ein neuer Eintrag nimmt die alte Vorschau sofort weg und startet die Wartezeit neu.
	$effect(() => {
		const w = wert;
		gezeigt = null;
		const uhr = setTimeout(() => (gezeigt = { wert: w }), WARTEZEIT);
		return () => clearTimeout(uhr);
	});

	$effect(() => {
		if (!gezeigt || !liste) return;
		const flaeche = { breite, hoehe };
		const messen = () => {
			const eintrag = liste.querySelector(`[data-i="${aktiv}"]`);
			if (!eintrag) return;
			lage = berechneVorschauBox(
				eintrag.getBoundingClientRect(),
				liste.getBoundingClientRect(),
				flaeche,
				{
					breite: window.innerWidth,
					hoehe: window.innerHeight
				}
			);
		};
		return folgeLage(() => liste.querySelector(`[data-i="${aktiv}"]`), messen);
	});
</script>

<!-- Unsichtbar, bis die Maße feststehen: Sonst stünde sie einen Augenblick an der falschen Seite. -->
{#if gezeigt}
	<div
		bind:offsetWidth={breite}
		bind:offsetHeight={hoehe}
		data-menue-vorschau
		aria-hidden="true"
		class="pointer-events-none"
		style="position:fixed; left:{lage.left}px; top:{lage.top}px; z-index:60; visibility:{breite
			? 'visible'
			: 'hidden'};"
	>
		{@render inhalt(gezeigt.wert)}
	</div>
{/if}
