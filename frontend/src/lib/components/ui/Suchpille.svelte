<script>
	import { Search, Camera } from '@lucide/svelte';

	/**
	 * Die Suchpille — EIN Bauteil für alle Suchfelder, die das Werkzeug einer Seite sind
	 * (nicht ein Datenfeld in einem Formular).
	 *
	 * Die Bauart gab es schon dreifach: in der Kiosk-Omnibox, in der Medienkatalog-Suche
	 * und, davon abgewichen, im Kollegiums-Portal und im öffentlichen OPAC. Absprache vom
	 * 10.08.2026: „die omnibox bei mein portal und katalog ist eine komplett andere".
	 * Gemessen stimmte das an sieben Stellen gleichzeitig — Höhe, Radius, Fläche,
	 * Rahmen, Fokusfarbe, Schriftgröße und der Platzhaltertext („… suchen …" gegen
	 * „… eingeben …"). Drei Kopien driften; ein Bauteil kann das nicht.
	 *
	 * Material 3: gefüllte Pille auf surface-container, führendes Symbol, im Fokus weiße
	 * Fläche mit Umriss. Der Container trägt Rahmen, Fläche und Fokus — das Feld selbst
	 * trägt nichts und füllt ihn nur (h-full). Deshalb steht die Pille bewusst neben der
	 * 36-px-Control-Skala aus styles/basis.css.
	 *
	 * KEIN `focus-within:shadow-md` (entfernt 11.08.2026). Es stand hier, weil die
	 * Medienkatalog-Fassung es mitbrachte — die Kiosk-Omnibox hatte es nie. Damit sahen die
	 * Pillen im Fokus unterschiedlich aus, und das Gate merkte es nicht: Es verglich die
	 * BREITE des Randes, nicht Farbe und Schatten. Aufgefallen ist es am Bildschirm.
	 * Sachlich gehört es ohnehin nicht dazu — M3 hebt eine Suchleiste beim Fokussieren
	 * nicht an, und Erhebung ist dort ohnehin Farbe (tonal), kein Schlagschatten. Das
	 * Fokus-Signal ist der Umriss.
	 *
	 * @type {{
	 *   id: string,
	 *   wert: string,
	 *   platzhalter: string,
	 *   etikett: string,
	 *   autofokus?: boolean,
	 *   disabled?: boolean,
	 *   element?: HTMLInputElement,
	 *   oninput?: (e: Event) => void,
	 *   onfocus?: (e: FocusEvent) => void,
	 *   onblur?: (e: FocusEvent) => void,
	 *   nachlaufend?: import('svelte').Snippet,
	 *   kamera?: boolean
	 * }}
	 * element: bind:this-Ersatz für Aufrufer, die den Fokus selbst setzen (Inventur-Scan
	 * nach jedem Treffer). disabled: während ein Scan verarbeitet wird.
	 *
	 * WO SIE HINGEHÖRT (Absprache vom 04.09.2026: „eine Leiste! aber nicht 2 … es soll gleich
	 * aussehen"): Jede Seite hat GENAU EINE Suche, und die ist diese Pille — über die volle
	 * Breite, ganz oben im Inhalt. Filter, Auswahlfelder und Knöpfe stehen in einer eigenen
	 * Zeile DARUNTER und bleiben auf der 36-px-Grundlinie (ui/Suchfeld.svelte); nebeneinander
	 * säße die Pille 12 px höher als alles daneben. Der Gegenstand ist der der Seite: Der
	 * Katalog sucht Bücher, die Inventur scannt, das Mahnwesen sucht Schüler. Bis zum
	 * 04.09.2026 stand darüber zusätzlich eine globale Leiste — zwei Suchzeilen je Seite,
	 * und die größte davon konnte am wenigsten.
	 */
	let {
		id,
		wert = $bindable(''),
		platzhalter,
		etikett,
		autofokus = false,
		disabled = false,
		element = $bindable(),
		oninput,
		onfocus,
		onblur,
		nachlaufend,
		kamera = false
	} = $props();

	/** @type {HTMLInputElement | undefined} */
	let feld = $state();
	import CameraScanner from '../../CameraScanner.svelte';

	// Kamera-Scanner (seit 18.09.2026, Schalter `kamera`, Standard aus): Der erkannte Code
	// landet als Suchtext im Feld, dann geht ein input-Ereignis an das Feld — die Suche
	// läuft also exakt so los, als hätte jemand den Code eingetippt. Kein zweiter Suchweg.
	let kameraOffen = $state(false);
	function nachScan() {
		kameraOffen = false;
		feld?.dispatchEvent(new Event('input', { bubbles: true }));
	}

	$effect(() => {
		element = feld;
	});

	// Fokus beim Betreten der Seite.
	//
	// Ohne ihn geht der erste Anschlag ins Leere — bei einem Barcode-Scanner heisst das,
	// dass der Scan verloren geht, ohne dass jemand einen Fehler sieht. Bewusst per
	// .focus() statt per autofocus-Attribut: Das Attribut wirkt nur beim ersten Laden des
	// Dokuments, und diese Oberfläche wechselt die Ansicht ohne Seitenwechsel.
	$effect(() => {
		if (autofokus) feld?.focus();
	});
</script>

<div
	class="group flex items-center w-full h-12 px-5 bg-slate-100 rounded-full border border-transparent transition-all duration-200 focus-within:bg-white focus-within:border-blue-600 focus-within:ring-1 focus-within:ring-blue-600"
>
	<Search
		class="h-5 w-5 shrink-0 text-slate-500 group-focus-within:text-blue-600 transition-colors duration-200"
		aria-hidden="true"
	/>
	<!-- Die vier Abwehr-Attribute gegen Passwortverwalter: LastPass, Dashlane und 1Password
	     halten ein Textfeld in einem Dialog sonst für ein Anmeldeformular und füllen es
	     ungefragt aus. `autocomplete="off"` allein reicht ihnen nicht. Sie standen bisher
	     nur an EINEM Feld (der Suche im Klassensatz-Dialog); hier gelten sie für alle. -->
	<!-- type="search" statt "text" (04.09.2026): Erst damit meldet sich die Pille dem
	     Screenreader und den Tests als Suchfeld (role=searchbox) — dieselbe Rolle, die das
	     kleine Suchfeld-Bauteil längst trägt. Aufgefallen beim Umbau der Verwaltungsseiten
	     auf die Pille: Drei e2e-Tests suchten eine `searchbox` und fanden nichts mehr.
	     Chromes eigenes Löschkreuz wird unten weggeblendet, sonst stünde neben dem
	     nachlaufenden Symbol ein zweites. -->
	<input
		{id}
		name={id}
		type="search"
		autocomplete="off"
		spellcheck="false"
		data-lpignore="true"
		data-form-type="other"
		bind:this={feld}
		bind:value={wert}
		{disabled}
		{oninput}
		{onfocus}
		{onblur}
		aria-label={etikett}
		placeholder={platzhalter}
		class="h-full flex-1 min-w-0 bg-transparent border-none outline-none focus:ring-0 px-3 text-slate-900 placeholder:text-slate-500 text-base [&::-webkit-search-cancel-button]:appearance-none"
	/>
	{#if nachlaufend}
		{@render nachlaufend()}
	{/if}
	<!-- Kamera-Scanner (Mobilgerät) als nachlaufendes Symbol der Suche — dieselbe Bauart wie
	     an der Theke (OmniboxInput). Ein Handscanner braucht ihn nicht: Der tippt wie eine
	     Tastatur ins Feld und löst dieselbe Suche aus. -->
	{#if kamera}
		<button
			type="button"
			onclick={() => (kameraOffen = !kameraOffen)}
			title="Kamera-Scanner (Mobilgerät)"
			data-tip="Kamera-Scanner (Mobilgerät)"
			aria-label="Kamera-Barcode-Scanner ein- oder ausschalten"
			class="h-12 w-12 -mr-4 shrink-0 flex items-center justify-center rounded-full transition-colors {kameraOffen
				? 'bg-secondary-container text-on-secondary-container'
				: 'text-on-surface-variant hover:text-primary'}"
		>
			<Camera class="h-5 w-5" aria-hidden="true" />
		</button>
	{/if}
</div>
{#if kameraOffen}
	<div class="mt-2">
		<CameraScanner
			stopCamera={() => (kameraOffen = false)}
			bind:queryVal={wert}
			submitAction={nachScan}
		/>
	</div>
{/if}
