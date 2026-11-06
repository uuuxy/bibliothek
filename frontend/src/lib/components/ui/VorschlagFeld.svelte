<!-- @component VorschlagFeld — ein Textfeld, das beim Tippen Werte aus dem Bestand vorschlägt.

     Material 3 nennt das ein Menü am Textfeld mit Filter: „As someone types, the list of menu
     options filters to show relevant results. This helps people quickly find the right option
     from a long list.“ Getippt werden darf auch, was die Liste nicht kennt („Some variations can
     accept user-entered input“, material-components-android, Exposed dropdown menu).

     Kein <datalist>, weil dessen Liste der Browser zeichnet: Systemschrift, keine Höchsthöhe,
     eine lange Liste setzt Chrome neben das Feld über die Seite. Kein Feld oder Select, weil
     Feld an der 200-Zeilen-Marke steht und Select nur aus der Liste wählt. -->
<script>
	import { ChevronDown } from '@lucide/svelte';
	import Feld from './Feld.svelte';
	import SelectListe from './SelectListe.svelte';
	import { berechneBox } from './selectGeometrie.js';
	import { passendeVorschlaege, vorschlagTaste } from './vorschlagRegeln.js';

	/**
	 * vorschlaege: die Werte zur Wahl; `zusatz` steht rechts im Eintrag (etwa „21 Titel“).
	 * onwaehlen: meldet eine Wahl aus der Liste (ChipFeld macht daraus sofort einen Chip).
	 * onkeydown bekommt nur Tasten, die die Liste nicht verbraucht hat. Alles Weitere (etikett,
	 * hint, placeholder, required …) geht an Feld.
	 * @type {{
	 *   value?: string | null,
	 *   vorschlaege?: Array<{ wert: string, zusatz?: string }>,
	 *   element?: HTMLInputElement,
	 *   onwaehlen?: (wert: string) => void,
	 *   oninput?: (e: Event & { currentTarget: EventTarget & HTMLInputElement }) => void,
	 *   onkeydown?: (e: KeyboardEvent & { currentTarget: EventTarget & HTMLInputElement }) => void,
	 *   onblur?: (e: FocusEvent & { currentTarget: EventTarget & HTMLInputElement }) => void
	 * } & Omit<import('svelte').ComponentProps<typeof Feld>, 'value' | 'element' | 'nachlaufend' | 'oninput' | 'onkeydown' | 'onblur'>}
	 */
	let {
		value = $bindable(),
		vorschlaege = [],
		element = $bindable(),
		id = undefined,
		disabled = false,
		onwaehlen = undefined,
		oninput = undefined,
		onkeydown = undefined,
		onblur = undefined,
		...rest
	} = $props();

	// Eindeutige Kennungen für aria-controls und aria-activedescendant, auch ohne `id`.
	const eigen = $props.id();
	const kennung = $derived(id ?? eigen);
	/** @param {number} i */
	const zeilenKennung = (i) => `${kennung}-option-${i}`;

	/** @type {HTMLDivElement | undefined} */
	let liste = $state();
	// 'getippt' zeigt die Liste nur, solange Text im Feld steht; 'ganz' auch ohne (der Pfeil im
	// Feld, Pfeil abwärts). Leert der Aufrufer das Feld, etwa nach einem Chip, geht sie so zu.
	let offen = $state(/** @type {false | 'getippt' | 'ganz'} */ (false));
	let aktiv = $state(-1);
	let box = $state({ left: 0, top: 0, breite: 0 });

	const treffer = $derived(passendeVorschlaege(vorschlaege, value ?? ''));
	const sichtbar = $derived(
		!disabled &&
			treffer.length > 0 &&
			(offen === 'ganz' || (offen === 'getippt' && !!value?.trim()))
	);

	// Neue Treffer heben die Markierung auf; sonst zeigte sie auf eine andere Zeile.
	$effect(() => {
		void treffer;
		aktiv = -1;
	});

	$effect(() => {
		if (!sichtbar) return;
		const messen = () => {
			if (element) box = berechneBox(element, treffer.length);
		};
		messen();
		window.addEventListener('scroll', messen, true);
		window.addEventListener('resize', messen);
		return () => {
			window.removeEventListener('scroll', messen, true);
			window.removeEventListener('resize', messen);
		};
	});

	function schliessen() {
		offen = false;
		aktiv = -1;
	}

	/** @param {number} i */
	function waehlen(i) {
		const v = treffer[i];
		if (!v) return;
		value = v.wert;
		schliessen();
		element?.focus();
		onwaehlen?.(v.wert);
	}

	/** @param {Event & { currentTarget: EventTarget & HTMLInputElement }} e */
	function getippt(e) {
		oninput?.(e);
		offen = e.currentTarget.value.trim() ? 'getippt' : false;
		aktiv = -1;
	}

	/** @param {KeyboardEvent & { currentTarget: EventTarget & HTMLInputElement }} e */
	function taste(e) {
		if (e.defaultPrevented) return;
		const befehl = vorschlagTaste(e.key, sichtbar, aktiv, treffer.length);
		if (befehl.verhindern) e.preventDefault();
		if (befehl.tat === 'oeffnen') {
			offen = 'ganz';
			aktiv = 0;
		} else if (befehl.tat === 'wandern') {
			// Läuft um; ohne Markierung beginnt abwärts die erste Zeile, aufwärts die letzte.
			const richtung = befehl.richtung ?? 1;
			const von = aktiv >= 0 ? aktiv : richtung > 0 ? -1 : 0;
			aktiv = (von + richtung + treffer.length) % treffer.length;
		} else if (befehl.tat === 'waehlen') waehlen(aktiv);
		else if (befehl.tat === 'schliessen') schliessen();
		if (!e.defaultPrevented) onkeydown?.(e);
	}

	function umschalten() {
		if (sichtbar) return schliessen();
		offen = 'ganz';
		aktiv = -1;
		element?.focus();
	}
</script>

<svelte:window
	onpointerdown={(e) => {
		if (!sichtbar) return;
		const z = /** @type {Node} */ (e.target);
		if (!element?.parentElement?.contains(z) && !liste?.contains(z)) schliessen();
	}}
/>

<Feld
	bind:value
	bind:element
	{id}
	{disabled}
	role="combobox"
	aria-autocomplete="list"
	aria-expanded={sichtbar}
	aria-controls={sichtbar ? `${kennung}-liste` : undefined}
	aria-activedescendant={sichtbar && aktiv >= 0 ? zeilenKennung(aktiv) : undefined}
	autocomplete="off"
	oninput={getippt}
	onkeydown={taste}
	onblur={(e) => {
		schliessen();
		onblur?.(e);
	}}
	{...rest}
>
	{#snippet nachlaufend()}
		<!-- M3, Text fields: „A dropdown arrow indicates that a text field has a nested selection
		     component.“ Er zeigt die Liste auch ohne Tippen; der Fokus bleibt dabei im Feld. -->
		{#if vorschlaege.length && !disabled}
			<button
				type="button"
				tabindex="-1"
				aria-label={sichtbar ? 'Vorschläge schließen' : 'Vorschläge zeigen'}
				data-tip={sichtbar ? 'Vorschläge schließen' : 'Vorschläge zeigen'}
				onmousedown={(e) => e.preventDefault()}
				onclick={umschalten}
				class="icon-btn text-on-surface-variant"
			>
				<ChevronDown
					class="h-4 w-4 transition-transform {sichtbar ? 'rotate-180' : ''}"
					aria-hidden="true"
				/>
			</button>
		{/if}
	{/snippet}
</Feld>

{#if sichtbar}
	<SelectListe
		options={treffer.map((v) => ({ value: v.wert, label: v.wert, zusatz: v.zusatz }))}
		value={value ?? ''}
		{aktiv}
		{box}
		{kennung}
		{zeilenKennung}
		onwaehlen={waehlen}
		onaktiv={(i) => (aktiv = i)}
		onelement={(el) => (liste = el)}
	/>
{/if}
