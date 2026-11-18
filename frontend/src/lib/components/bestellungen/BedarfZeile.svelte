<!-- @component BedarfZeile — eine Zeile des Bestellbedarfs. Gehört der Titel zu einem Buch
     mit mehreren Auflagen, ist die Zeile das Buch: Titel und ISBN der neuesten Auflage (sie
     wird bestellt), die Zahlen gelten für das Buch, und die dritte Zeile nennt den Bestand je
     Auflage. Material 3, Lists:
     „Limit supporting text to one to three lines". -->
<script>
	import Icon from '../ui/Icon.svelte';
	import CoverVorschau from '../ui/CoverVorschau.svelte';
	import BuchCover from '../ui/BuchCover.svelte';
	import Menue from '../ui/Menue.svelte';
	import { auflagenAufschluesselung } from '../../utils/auflagenText.js';

	/** @type {{ r: any, onAddToCart: (r: any, menge: number) => void, onNeueAuflage?: (r: any) => void }} */
	let { r, onAddToCart, onNeueAuflage } = $props();

	// Material 3, Menus: „Use menus in situations that need extra actions, like: Overflow menus" —
	// das Plus bleibt die eine sichtbare Aktion der Zeile, alles Weitere steht dahinter.
	const eintraege = [{ id: 'neue-auflage', text: 'Neue Auflage bestellen …', icon: 'library_add' }];
</script>

<div class="group flex items-center gap-3 rounded-xl border border-transparent px-3 py-2">
	<!-- Cover IN der Zeile, zugleich Auslöser der Großansicht (so sieht CoverVorschau es
	     über `children` vor). Frühere Gegengründe gemessen widerlegt: `loading="lazy"`
	     erspart die 247 Requests, 5.724 von 8.706 Titeln ohne Exemplar haben ein Cover. -->
	<CoverVorschau isbn={r.isbn || ''} coverUrl={r.cover_url || ''} titel={r.titel}>
		<BuchCover coverUrl={r.cover_url || ''} isbn={r.isbn || ''} titel={r.titel} dekorativ />
	</CoverVorschau>

	<div class="min-w-0 flex-1">
		<h4 class="text-on-surface text-title-small truncate leading-snug">{r.titel}</h4>
		<p class="text-xs text-on-surface-variant truncate">
			{#if r.isbn}<span class="font-mono">{r.isbn}</span>{/if}
			{#if r.verlag}<span class="mx-1.5">·</span>{r.verlag}{/if}
			{#if r.signatur}<span class="mx-1.5">·</span>{r.signatur}{/if}
		</p>
		{#if r.auflagen?.length > 1}
			<p class="truncate text-xs text-on-surface-variant">
				{auflagenAufschluesselung(r.auflagen)}
			</p>
		{/if}
	</div>

	<!-- Material 3, Lists: „Use trailing text for supplemental details, like a price, count, or
	     date". Bestand gegen Bedarf; was fehlt, legt das Plus in den Warenkorb. -->
	<div
		class="text-right shrink-0 leading-tight text-body-medium-emphasized tabular-nums text-on-surface-variant"
		title="Bestand, gemessen am Bedarf im nächsten Schuljahr"
	>
		{r.gesamt_bestand} von {r.bedarf}
	</div>

	<button
		onclick={() => onAddToCart(r, r.fehlen)}
		aria-label="{r.fehlen} {r.fehlen === 1
			? 'Exemplar'
			: 'Exemplare'} von {r.titel} zur Bestellung hinzufügen"
		data-tip="{r.fehlen} zur Bestellung hinzufügen"
		class="icon-btn shrink-0 w-9 h-9 border border-outline-variant text-on-surface-variant"
	>
		<Icon name="add" />
	</button>
	{#if onNeueAuflage}
		<Menue etikett="Weitere Aktionen zu {r.titel}" {eintraege} onwahl={() => onNeueAuflage?.(r)} />
	{/if}
</div>
