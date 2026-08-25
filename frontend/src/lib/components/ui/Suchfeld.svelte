<script>
	import { Search } from '@lucide/svelte';

	/**
	 * Das Suchfeld IN einer Werkzeugleiste — die kleine Schwester der Suchpille.
	 *
	 * Der Unterschied ist keine Geschmacksfrage: Die Pille (48 px, rund, gefüllt) ist das
	 * Werkzeug einer ganzen Seite und soll sich abheben. Dieses Feld steht neben Knöpfen
	 * und Auswahlfeldern und gehört deshalb auf die 36-px-Control-Grundlinie aus
	 * styles/basis.css — eine Pille an dieser Stelle säße 12 px höher als alles daneben.
	 *
	 * Sechs Fundstellen trugen dieselbe Bauart mit sieben kleinen Abweichungen: Symbol
	 * 16 gegen 20 px, left-3 gegen left-3.5, pl-9 gegen pl-10, weisse gegen graue Fläche,
	 * Fokusrahmen blue-400 gegen blue-500, `placeholder-slate-400` gegen
	 * `placeholder:text-slate-400` und Auslassungspunkte mal als „…", mal als „...".
	 * Nichts davon war entschieden — es war kopiert und dann auseinandergelaufen.
	 *
	 * Seit dem 25.08.2026 trägt es Rahmen, Fläche und Fokus aus demselben Rezept wie
	 * ui/Feld.svelte und ui/Select.svelte (outline-variant, surface-container-lowest,
	 * primary) — ein Suchfeld neben einem Textfeld in derselben Leiste muss als EIN
	 * Vokabular lesen; vorher stand es als einziges Feld noch auf slate/blue.
	 *
	 * @type {{
	 *   wert: string,
	 *   platzhalter: string,
	 *   etikett: string,
	 *   id?: string,
	 *   klasse?: string,
	 *   oninput?: (e: Event) => void,
	 *   onfocus?: (e: FocusEvent) => void,
	 *   onblur?: (e: FocusEvent) => void,
	 *   nachlaufend?: import('svelte').Snippet
	 * }}
	 */
	let {
		wert = $bindable(''),
		platzhalter,
		etikett,
		id = undefined,
		klasse = '',
		oninput,
		onfocus,
		onblur,
		nachlaufend
	} = $props();
</script>

<div class="relative {klasse}">
	<Search
		class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-on-surface-variant pointer-events-none"
		aria-hidden="true"
	/>
	<input
		{id}
		type="search"
		autocomplete="off"
		bind:value={wert}
		{oninput}
		{onfocus}
		{onblur}
		aria-label={etikett}
		placeholder={platzhalter}
		class="h-9 w-full rounded-xl border border-outline-variant bg-surface-container-lowest pl-9 {nachlaufend
			? 'pr-10'
			: 'pr-3'} text-sm text-on-surface transition-colors placeholder:text-outline focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
	/>
	{#if nachlaufend}
		<div class="absolute right-3 top-1/2 -translate-y-1/2">{@render nachlaufend()}</div>
	{/if}
</div>
