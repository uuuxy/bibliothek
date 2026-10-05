<!-- @component BuchVorschlagFeld — Textfeld, das beim Tippen Titel aus dem Katalog vorschlägt.

     Ein Vorschlag füllt nur das Feld: Gespeichert wird Text, und wer kein Buch aus dem
     Katalog meint, schreibt weiter. Deshalb ein Textfeld mit Menü und kein Auswahlfeld.
     Gesucht wird wie im Portal (portalSuche.svelte.js), die Menüfläche ist die von Select. -->
<script>
	import Feld from '../ui/Feld.svelte';
	import SelectListe from '../ui/SelectListe.svelte';
	import BuchCover from '../ui/BuchCover.svelte';
	import { berechneBox } from '../ui/selectGeometrie.js';
	import { naechsterIndex } from '../ui/selectTastatur.js';
	import { erzeugePortalSuche } from './portalSuche.svelte.js';

	/** @type {{ value?: string, label: string, placeholder?: string, maxlength?: number, element?: HTMLInputElement }} */
	let {
		value = $bindable(''),
		label,
		placeholder = '',
		maxlength = undefined,
		element = $bindable()
	} = $props();

	const kennung = $props.id();
	/** @param {number} i */
	const zeilenKennung = (i) => `${kennung}-option-${i}`;
	const suche = erzeugePortalSuche();

	let offen = $state(false);
	let gemerkt = $state(-1);
	/** @type {HTMLDivElement | undefined} */
	let liste = $state();
	let box = $state({ left: 0, top: 0, breite: 0 });

	/** @typedef {{ value: string, label: string, neben: string, cover_url?: string, isbn?: string, disabled?: boolean }} Vorschlag */
	const optionen = $derived(
		/** @type {Vorschlag[]} */ (
			suche.treffer.map((t) => ({
				value: t.id,
				label: t.titel,
				neben: [t.autor, t.isbn].filter(Boolean).join(' · '),
				cover_url: t.cover_url,
				isbn: t.isbn
			}))
		)
	);
	const sichtbar = $derived(offen && optionen.length > 0);
	// Neue Treffer können kürzer sein als die Liste, in der die Markierung stand.
	const aktiv = $derived(gemerkt < optionen.length ? gemerkt : -1);

	function messen() {
		if (element) box = berechneBox(element, optionen.length);
	}

	$effect(() => {
		if (!sichtbar) return;
		messen();
		window.addEventListener('scroll', messen, true);
		window.addEventListener('resize', messen);
		return () => {
			window.removeEventListener('scroll', messen, true);
			window.removeEventListener('resize', messen);
		};
	});

	/** Der Wert kommt aus dem Ereignis: Die Bindung des Feldes schreibt am selben Ereignis.
	 * @param {Event} e */
	function eingabe(e) {
		suche.text = /** @type {HTMLInputElement} */ (e.currentTarget).value;
		offen = true;
		gemerkt = -1;
	}

	function schliessen() {
		offen = false;
		gemerkt = -1;
	}

	/** @param {number} i */
	function waehlen(i) {
		const o = optionen[i];
		if (!o) return;
		value = o.label;
		// Ohne Suchtext endet die Suche, und der gewählte Titel schlägt sich nicht selbst vor.
		suche.text = '';
		schliessen();
		element?.focus();
	}

	/** Pfeile wandern, Enter wählt die Markierung, Tab geht weiter ins nächste Feld.
	 * @param {KeyboardEvent} e */
	function taste(e) {
		if (!sichtbar) return;
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			gemerkt = naechsterIndex(optionen, aktiv, 1);
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			gemerkt = naechsterIndex(optionen, Math.max(aktiv, 0), -1);
		} else if (e.key === 'Enter' && aktiv >= 0) {
			e.preventDefault();
			waehlen(aktiv);
		} else if (e.key === 'Escape') {
			e.preventDefault();
			schliessen();
		} else if (e.key === 'Tab') {
			schliessen();
		}
	}
</script>

<svelte:window
	onpointerdown={(e) => {
		if (!offen) return;
		const z = /** @type {Node} */ (e.target);
		if (!element?.contains(z) && !liste?.contains(z)) schliessen();
	}}
/>

<Feld
	bind:value
	bind:element
	{label}
	{placeholder}
	{maxlength}
	autocomplete="off"
	role="combobox"
	aria-autocomplete="list"
	aria-expanded={sichtbar}
	aria-controls={sichtbar ? `${kennung}-liste` : undefined}
	aria-activedescendant={sichtbar && aktiv >= 0 ? zeilenKennung(aktiv) : undefined}
	oninput={eingabe}
	onkeydown={taste}
/>

{#if sichtbar}
	<SelectListe
		options={optionen}
		value={undefined}
		{aktiv}
		{box}
		{kennung}
		{zeilenKennung}
		onwaehlen={waehlen}
		onaktiv={(i) => (gemerkt = i)}
		onelement={(el) => (liste = el)}
		{zeile}
	/>
{/if}

<!-- M3 Lists: das Bild an der führenden Kante, darunter der Nebentext in on-surface-variant. -->
{#snippet zeile(/** @type {any} */ o)}
	<BuchCover coverUrl={o.cover_url} isbn={o.isbn} titel={o.label} dekorativ nurGespeichert />
	<span class="min-w-0 flex-1">
		<span class="block truncate">{o.label}</span>
		<span class="block truncate text-xs text-on-surface-variant">{o.neben}</span>
	</span>
{/snippet}
