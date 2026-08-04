<!-- @component Select — Material-3-Auswahlfeld („exposed dropdown menu").

     Ersetzt das native <select>. Warum überhaupt: Das Browser-Widget folgt keiner
     Designsprache — es bringt Systemoptik, Systemschrift und einen Systempfeil mit,
     und es SCHNEIDET lange Werte hart ab. Bei Lieferantennamen war genau das der
     gemeldete Fall („Cornelsen ist abgeschnitten").

     Zwei Dinge, die hier bewusst gelöst sind:

     1. POSITION FIXED statt absolute. Die Liste öffnet sich regelmäßig in einem
        Container mit overflow-y-auto (Formularspalten, Tabellen, Dialoge). Absolut
        positioniert würde sie dort abgeschnitten — dieselbe Falle, die in CoverPeek
        schon dokumentiert ist. Die Koordinaten kommen deshalb aus
        getBoundingClientRect und werden beim Scrollen und bei Größenänderung neu
        berechnet.

     2. TASTATURBEDIENUNG. Ein natives select kann jeder ohne Maus bedienen; ein
        nachgebautes muss das auch. Pfeiltasten wandern, Pos1/Ende springen, Enter
        und Leertaste wählen, Escape schließt und gibt den Fokus zurück, Tippen
        springt zum ersten passenden Eintrag. Ohne das wäre der Ersatz ein
        Rückschritt, egal wie er aussieht. -->
<script>
	import { ChevronDown, Check } from '@lucide/svelte';

	/**
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
		class: className = '',
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
		if (!ausloeser) return;
		const r = ausloeser.getBoundingClientRect();
		// Nach unten, wenn Platz ist; sonst nach oben. 240 px ist die Höhengrenze der
		// Liste (max-h-60), darunter würde sie aus dem Fenster ragen.
		const untenPlatz = window.innerHeight - r.bottom;
		const nachOben = untenPlatz < 240 && r.top > untenPlatz;
		box = {
			left: r.left,
			top: nachOben ? Math.max(8, r.top - Math.min(240, options.length * 40 + 8) - 4) : r.bottom + 4,
			breite: r.width
		};
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

	/** @param {KeyboardEvent} e */
	function taste(e) {
		if (!offen) {
			if (['Enter', ' ', 'ArrowDown', 'ArrowUp'].includes(e.key)) {
				e.preventDefault();
				oeffnen();
			}
			return;
		}
		switch (e.key) {
			case 'Escape':
				e.preventDefault();
				schliessen();
				return;
			case 'ArrowDown':
				e.preventDefault();
				wandern(1);
				return;
			case 'ArrowUp':
				e.preventDefault();
				wandern(-1);
				return;
			case 'Home':
				e.preventDefault();
				aktiv = 0;
				return;
			case 'End':
				e.preventDefault();
				aktiv = options.length - 1;
				return;
			case 'Enter':
			case ' ':
				e.preventDefault();
				waehlen(aktiv);
				return;
			case 'Tab':
				schliessen(false);
				return;
		}
		// Tippen springt zum ersten passenden Eintrag — wie beim nativen select.
		if (e.key.length === 1) {
			suchpuffer += e.key.toLowerCase();
			clearTimeout(suchTimer);
			suchTimer = setTimeout(() => (suchpuffer = ''), 600);
			const treffer = options.findIndex((o) => o.label.toLowerCase().startsWith(suchpuffer));
			if (treffer >= 0) aktiv = treffer;
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
	class="h-9 w-full flex items-center gap-2 px-3 rounded-md border bg-white text-sm text-left transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/30 {offen
		? 'border-blue-600'
		: 'border-slate-300'} {className}"
	{...rest}
>
	<!-- truncate statt hartem Abschneiden: Der native Select kappte mitten im
	     Zeichen, hier endet ein zu langer Wert sichtbar mit Auslassungspunkten. -->
	<span class="flex-1 min-w-0 truncate {beschriftung ? 'text-slate-900' : 'text-slate-400'}">
		{beschriftung || placeholder}
	</span>
	<ChevronDown
		class="h-4 w-4 shrink-0 text-slate-500 transition-transform {offen ? 'rotate-180' : ''}"
		aria-hidden="true"
	/>
</button>

{#if offen}
	<div
		bind:this={liste}
		id={id ? `${id}-liste` : undefined}
		role="listbox"
		tabindex="-1"
		style="position:fixed; left:{box.left}px; top:{box.top}px; width:{box.breite}px; z-index:60;"
		class="max-h-60 overflow-y-auto rounded-md border border-slate-200 bg-white shadow-xl py-1"
	>
		{#each options as o, i (o.value)}
			<div
				data-i={i}
				role="option"
				aria-selected={o.value === value}
				aria-disabled={o.disabled || undefined}
				tabindex="-1"
				onclick={() => waehlen(i)}
				onkeydown={() => {}}
				onpointerenter={() => (aktiv = i)}
				class="flex items-center gap-2 px-3 py-2 text-sm cursor-pointer {o.disabled
					? 'opacity-40 cursor-not-allowed'
					: ''} {i === aktiv ? 'bg-slate-100' : ''} {o.value === value
					? 'text-blue-700 font-semibold'
					: 'text-slate-800'}"
			>
				<span class="flex-1 min-w-0 truncate">{o.label}</span>
				{#if o.value === value}
					<Check class="h-4 w-4 shrink-0" aria-hidden="true" />
				{/if}
			</div>
		{/each}
		{#if !options.length}
			<div class="px-3 py-2 text-sm text-slate-400">Keine Auswahl verfügbar</div>
		{/if}
	</div>
{/if}
