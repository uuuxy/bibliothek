<script>
	import { Search, Camera } from '@lucide/svelte';
	import { tick } from 'svelte';

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
	 *   nachlaufend?: import('svelte').Snippet,
	 *   kamera?: boolean,
	 *   autofokus?: boolean,
	 *   onscan?: (code: string) => void
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
		nachlaufend,
		kamera = false,
		autofokus = false,
		onscan
	} = $props();

	// Fokus beim Betreten — dieselbe Begründung wie in Suchpille: Ohne ihn geht der erste
	// Anschlag ins Leere, und bei einem Handscanner heißt das, der Scan ist weg, ohne dass
	// jemand einen Fehler sieht. Gemessen am 18.09.2026: Auf /bestellungen, /medienkatalog
	// und /schuelerdatei lag der Fokus auf <body>, blind getipptes landete nirgends.
	$effect(() => {
		if (autofokus) feld?.focus();
	});

	/** @type {HTMLInputElement | undefined} */
	let feld = $state();
	import CameraScanner from '../../CameraScanner.svelte';

	// Kamera-Scanner (seit 18.09.2026, Schalter `kamera`, Standard aus): Der erkannte Code
	// landet als Suchtext im Feld, dann geht ein input-Ereignis an das Feld — die Suche
	// läuft also exakt so los, als hätte jemand den Code eingetippt. Kein zweiter Suchweg.
	let kameraOffen = $state(false);
	async function nachScan() {
		kameraOffen = false;
		// Mit `onscan` entscheidet der Aufrufer, was ein Scan auslöst — die Titelsuche der
		// Bestellung legt den eindeutigen Treffer direkt in die Übernahme, statt eine Liste
		// zum Antippen zu zeigen (18.09.2026). Ohne `onscan` bleibt es beim Alten: tippen,
		// als hätte es jemand eingegeben.
		if (onscan) {
			onscan(wert);
			return;
		}
		// `await tick()` vor dem Ereignis: An DEMSELBEN input-Ereignis hängt Svelte die
		// Rückschreibung von bind:value. Ohne das Warten liest sie den noch leeren DOM-Wert
		// zurück und löscht den gescannten Code — Begründung und Messung in Suchpille.
		await tick();
		feld?.dispatchEvent(new Event('input', { bubbles: true }));
	}
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
		bind:this={feld}
		bind:value={wert}
		{oninput}
		{onfocus}
		{onblur}
		aria-label={etikett}
		placeholder={platzhalter}
		class="h-9 w-full rounded-xl border border-outline bg-surface-container-lowest pl-9 {nachlaufend ||
		kamera
			? 'pr-10'
			: 'pr-3'} text-sm text-on-surface transition-colors placeholder:text-outline focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
	/>
	<div class="absolute right-1 top-1/2 flex -translate-y-1/2 items-center gap-1">
		{#if nachlaufend}
			<span class="mr-2">{@render nachlaufend()}</span>
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
				class="h-9 w-9 shrink-0 flex items-center justify-center rounded-full transition-colors {kameraOffen
					? 'bg-secondary-container text-on-secondary-container'
					: 'text-on-surface-variant hover:text-primary'}"
			>
				<Camera class="h-5 w-5" aria-hidden="true" />
			</button>
		{/if}
	</div>
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
