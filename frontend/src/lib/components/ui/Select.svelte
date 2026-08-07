<!-- @component Select — Material-3-Auswahlfeld („exposed dropdown menu").

     Ersetzt das native <select>. Warum überhaupt: Das Browser-Widget folgt
     keiner Designsprache — Systemoptik, Systemschrift, Systempfeil — und es
     SCHNEIDET lange Werte hart ab („Cornelsen ist abgeschnitten").

     TASTATURBEDIENUNG ist hier Pflicht, kein Extra: Ein natives select kann
     jeder ohne Maus bedienen; ein nachgebautes muss das auch. Pfeiltasten
     wandern, Pos1/Ende springen, Enter und Leertaste wählen, Escape schließt
     und gibt den Fokus zurück, Tippen springt zum ersten passenden Eintrag.

     Die Menüfläche liegt in SelectListe.svelte. -->
<script>
	import { ChevronDown } from '@lucide/svelte';
	import SelectListe from './SelectListe.svelte';
	import { berechneBox } from './selectGeometrie.js';

	/**
	 * `class` ERSETZT die Standardbreite w-full — sonst entschiede die
	 * Stylesheet-Reihenfolge über die Breite, nicht der Aufruf.
	 *
	 * ACHTUNG, keine `//`-Kommentare in den Objekttyp schreiben: Der JSDoc-Parser
	 * bricht dort ab. Genau das ist hier passiert — `aria-label` und `onchange`
	 * standen HINTER einem solchen Kommentar und waren deshalb nie Teil des Typs,
	 * also war jeder Aufruf, der sie übergibt, ein Fehler (14 Stück in 9 Dateien).
	 *
	 * @type {{
	 *   value?: any,
	 *   options?: Array<{ value: any, label: string, disabled?: boolean }>,
	 *   id?: string,
	 *   disabled?: boolean,
	 *   placeholder?: string,
	 *   class?: string,
	 *   'aria-label'?: string,
	 *   onchange?: (wert: any) => void
	 * }}
	 */
	let {
		value = $bindable(),
		options = [],
		id = undefined,
		disabled = false,
		placeholder = 'Bitte wählen',
		class: className = 'w-full',
		onchange = undefined,
		...rest
	} = $props();

	let offen = $state(false);
	let aktiv = $state(-1);
	/** @type {HTMLButtonElement | undefined} */
	let ausloeser = $state();
	/** @type {HTMLDivElement | undefined} */
	let liste = $state();
	let box = $state({ left: 0, top: 0, breite: 0 });
	let suchpuffer = '';
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let suchTimer;

	let gewaehlt = $derived(options.find((o) => o.value === value));
	let beschriftung = $derived(gewaehlt?.label ?? '');

	function messen() {
		if (ausloeser) box = berechneBox(ausloeser, options.length);
	}

	function oeffnen() {
		if (disabled) return;
		messen();
		offen = true;
		aktiv = Math.max(
			0,
			options.findIndex((o) => o.value === value)
		);
	}

	function schliessen(fokusZurueck = true) {
		offen = false;
		aktiv = -1;
		if (fokusZurueck) ausloeser?.focus();
	}

	/** @param {number} i */
	function waehlen(i) {
		const o = options[i];
		if (!o || o.disabled) return;
		value = o.value;
		onchange?.(o.value);
		schliessen();
	}

	/** @param {number} richtung */
	function wandern(richtung) {
		if (!options.length) return;
		let i = aktiv;
		for (let n = 0; n < options.length; n++) {
			i = (i + richtung + options.length) % options.length;
			if (!options[i].disabled) break;
		}
		aktiv = i;
	}

	/** Tippen springt zum ersten passenden Eintrag — wie beim nativen select.
	 * @param {string} zeichen */
	function tippsprung(zeichen) {
		suchpuffer += zeichen.toLowerCase();
		clearTimeout(suchTimer);
		suchTimer = setTimeout(() => (suchpuffer = ''), 600);
		const treffer = options.findIndex((o) => o.label.toLowerCase().startsWith(suchpuffer));
		if (treffer >= 0) aktiv = treffer;
	}

	/** @param {KeyboardEvent} e */
	function taste(e) {
		if (!offen) {
			if (['Enter', ' ', 'ArrowDown', 'ArrowUp'].includes(e.key)) {
				e.preventDefault();
				oeffnen();
			}
			return;
		}
		const sprung = { ArrowDown: 1, ArrowUp: -1 }[e.key];
		if (sprung) {
			e.preventDefault();
			wandern(sprung);
		} else if (e.key === 'Escape') {
			e.preventDefault();
			schliessen();
		} else if (e.key === 'Home' || e.key === 'End') {
			e.preventDefault();
			aktiv = e.key === 'Home' ? 0 : options.length - 1;
		} else if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			waehlen(aktiv);
		} else if (e.key === 'Tab') {
			schliessen(false);
		} else if (e.key.length === 1) {
			tippsprung(e.key);
		}
	}

	$effect(() => {
		if (!offen) return;
		const neu = () => messen();
		window.addEventListener('scroll', neu, true);
		window.addEventListener('resize', neu);
		return () => {
			window.removeEventListener('scroll', neu, true);
			window.removeEventListener('resize', neu);
		};
	});

	// Der aktive Eintrag muss sichtbar bleiben, sonst wandert die Auswahl blind.
	$effect(() => {
		if (!offen || aktiv < 0 || !liste) return;
		liste.querySelector(`[data-i="${aktiv}"]`)?.scrollIntoView({ block: 'nearest' });
	});
</script>

<svelte:window
	onpointerdown={(e) => {
		if (!offen) return;
		const z = /** @type {Node} */ (e.target);
		if (!ausloeser?.contains(z) && !liste?.contains(z)) schliessen(false);
	}}
/>

<button
	bind:this={ausloeser}
	type="button"
	{id}
	{disabled}
	role="combobox"
	aria-expanded={offen}
	aria-haspopup="listbox"
	aria-controls={offen && id ? `${id}-liste` : undefined}
	onclick={() => (offen ? schliessen() : oeffnen())}
	onkeydown={taste}
	class="flex h-9 cursor-pointer items-center gap-2 rounded-xl border bg-surface-container-lowest px-3 text-left text-sm text-on-surface transition-colors focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-40
		{offen ? 'border-primary ring-1 ring-primary' : 'border-outline-variant'} {className}"
	{...rest}
>
	<!-- truncate statt hartem Abschneiden: Der native Select kappte mitten im
	     Zeichen, hier endet ein zu langer Wert sichtbar mit Auslassungspunkten. -->
	<span class="min-w-0 flex-1 truncate {beschriftung ? '' : 'text-on-surface-variant'}">
		{beschriftung || placeholder}
	</span>
	<ChevronDown
		class="h-4 w-4 shrink-0 text-on-surface-variant transition-transform {offen
			? 'rotate-180'
			: ''}"
		aria-hidden="true"
	/>
</button>

{#if offen}
	<SelectListe
		{options}
		{value}
		{aktiv}
		{box}
		{id}
		onwaehlen={waehlen}
		onaktiv={(i) => (aktiv = i)}
		onelement={(el) => (liste = el)}
	/>
{/if}
