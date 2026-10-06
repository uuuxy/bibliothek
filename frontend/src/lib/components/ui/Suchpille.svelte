<script>
	import { Search, Camera } from '@lucide/svelte';
	import { tick } from 'svelte';

	/**
	 * Die Suchpille — ein Bauteil für alle Suchfelder, die das Werkzeug einer Seite sind
	 * (nicht ein Datenfeld in einem Formular). Mehrere Kopien liefen in Höhe, Radius, Fläche,
	 * Rahmen, Fokusfarbe, Schriftgröße und Platzhaltertext auseinander; ein Bauteil kann das
	 * nicht.
	 *
	 * Farben nach material-web, Token v0_192, search-bar: Fläche surface-container-high,
	 * führendes Symbol on-surface, Eingabe on-surface, Platzhalter on-surface-variant. Im Fokus
	 * weiße Fläche mit Umriss in primary. Der Container trägt Rahmen, Fläche und Fokus — das
	 * Feld selbst trägt nichts und füllt ihn nur (h-full). Deshalb steht die Pille neben der
	 * 36-px-Control-Skala aus styles/basis.css.
	 *
	 * Kein Schatten im Fokus: M3 hebt eine Suchleiste beim Fokussieren nicht an; das
	 * Fokus-Signal ist der Umriss. Sein zweiter Pixel liegt innen (ring-inset): Eine Pille, die
	 * bündig an der Kante eines Rollbereichs sitzt, verlöre einen außen liegenden Ring.
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
	 * Wo sie hingehört: Jede Seite hat genau eine Suche, und die ist diese Pille — über die
	 * volle Breite, ganz oben im Inhalt. Filter, Auswahlfelder und Knöpfe stehen in einer
	 * eigenen Zeile darunter und bleiben auf der 36-px-Grundlinie (ui/Suchfeld.svelte);
	 * nebeneinander säße die Pille 12 px höher als alles daneben. Der Gegenstand ist der der
	 * Seite: Der Katalog sucht Bücher, die Inventur scannt, das Mahnwesen sucht Schüler.
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

	// Kamera-Scanner (Schalter `kamera`, Standard aus): Der erkannte Code landet als Suchtext
	// im Feld, dann geht ein input-Ereignis an das Feld — die Suche läuft also so los, als
	// hätte jemand den Code eingetippt. Kein zweiter Suchweg.
	//
	// `await tick()` ist nötig: An demselben input-Ereignis hängt Svelte die Rückschreibung von
	// bind:value. Ohne das Warten steht im DOM-Feld noch der alte Wert, Svelte liest ihn
	// zurück und überschreibt den gescannten Code mit Leer.
	let kameraOffen = $state(false);
	async function nachScan() {
		kameraOffen = false;
		await tick();
		feld?.dispatchEvent(new Event('input', { bubbles: true }));
	}

	$effect(() => {
		element = feld;
	});

	// Fokus beim Betreten der Seite.
	//
	// Ohne ihn geht der erste Anschlag ins Leere — bei einem Barcode-Scanner heißt das,
	// dass der Scan verloren geht, ohne dass jemand einen Fehler sieht. Bewusst per
	// .focus() statt per autofocus-Attribut: Das Attribut wirkt nur beim ersten Laden des
	// Dokuments, und diese Oberfläche wechselt die Ansicht ohne Seitenwechsel.
	$effect(() => {
		if (autofokus) feld?.focus();
	});
</script>

<div
	class="group flex items-center w-full h-12 px-5 bg-surface-container-high rounded-full border border-transparent ring-inset transition-all duration-200 focus-within:bg-surface-container-lowest focus-within:border-primary focus-within:ring-1 focus-within:ring-primary"
>
	<Search
		class="h-5 w-5 shrink-0 text-on-surface group-focus-within:text-primary transition-colors duration-200"
		aria-hidden="true"
	/>
	<!-- Die vier Abwehr-Attribute gegen Passwortverwalter: LastPass, Dashlane und 1Password
	     halten ein Textfeld in einem Dialog sonst für ein Anmeldeformular und füllen es
	     ungefragt aus. `autocomplete="off"` allein reicht ihnen nicht. -->
	<!-- type="search": Damit meldet sich die Pille dem Screenreader und den Tests als
	     Suchfeld (role=searchbox) wie das kleine Suchfeld-Bauteil. Chromes eigenes Löschkreuz
	     wird unten weggeblendet, sonst stünde neben dem nachlaufenden Symbol ein zweites. -->
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
		class="h-full flex-1 min-w-0 bg-transparent border-none outline-none focus:ring-0 px-3 text-on-surface placeholder:text-on-surface-variant text-base [&::-webkit-search-cancel-button]:appearance-none"
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
