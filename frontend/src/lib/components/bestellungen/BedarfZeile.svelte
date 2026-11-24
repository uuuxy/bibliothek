<script module>
	// Zeile und Spaltenkopf (Bestellbedarf.svelte) teilen ihre Maße wie die Positionen darüber
	// (WarenkorbPosition): schmal Bestand und Knopf in einer zweiten Zeile unter dem Titel, ab
	// 768 px Breite der Liste in einer Zeile.
	export const STELLEN =
		'order-2 flex basis-full items-center justify-between gap-3 pl-10 @3xl:order-1 @3xl:basis-auto @3xl:justify-end @3xl:pl-0';
	export const SPALTEN = { bestand: '@3xl:w-28', knopf: '@3xl:w-44' };
</script>

<!-- @component BedarfZeile — eine Zeile des Bestellbedarfs. Gehört der Titel zu einem Buch
     mit mehreren Auflagen, ist die Zeile das Buch: Titel und ISBN der neuesten Auflage (sie
     wird bestellt), die Zahlen gelten für das Buch, und die dritte Zeile nennt den Bestand je
     Auflage. Material 3, Lists:
     „Limit supporting text to one to three lines". -->
<script>
	import Icon from '../ui/Icon.svelte';
	import Button from '../ui/Button.svelte';
	import CoverVorschau from '../ui/CoverVorschau.svelte';
	import BuchCover from '../ui/BuchCover.svelte';
	import Menue from '../ui/Menue.svelte';
	import { auflagenAufschluesselung } from '../../utils/auflagenText.js';

	/** @type {{ r: any, onAddToCart: (r: any, menge: number) => void, onNeueAuflage?: (r: any) => void }} */
	let { r, onAddToCart, onNeueAuflage } = $props();

	// Material 3, Menus: „Use menus in situations that need extra actions, like: Overflow menus" —
	// „hinzufügen" bleibt die eine sichtbare Aktion der Zeile, alles Weitere steht dahinter.
	const eintraege = [{ id: 'neue-auflage', text: 'Neue Auflage bestellen …', icon: 'library_add' }];
</script>

<li class="flex flex-wrap items-center gap-x-3 gap-y-2 py-3 @3xl:flex-nowrap">
	<!-- Cover IN der Zeile, zugleich Auslöser der Großansicht (so sieht CoverVorschau es
	     über `children` vor). Frühere Gegengründe gemessen widerlegt: `loading="lazy"`
	     erspart die 247 Requests, 5.724 von 8.706 Titeln ohne Exemplar haben ein Cover. -->
	<span class="w-7 shrink-0 self-start @3xl:self-center">
		<CoverVorschau isbn={r.isbn || ''} coverUrl={r.cover_url || ''} titel={r.titel}>
			<BuchCover coverUrl={r.cover_url || ''} isbn={r.isbn || ''} titel={r.titel} dekorativ />
		</CoverVorschau>
	</span>

	<div class="min-w-0 flex-1">
		<h4 class="text-on-surface text-title-small leading-snug">{r.titel}</h4>
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

	<div class={STELLEN}>
		<!-- Material 3, Lists: „Use trailing text for supplemental details, like a price, count, or
		     date". Bestand gegen Bedarf. -->
		<div
			class="{SPALTEN.bestand} text-right shrink-0 leading-tight text-body-medium-emphasized tabular-nums text-on-surface-variant"
			data-tip="Bestand, gemessen am Bedarf im nächsten Schuljahr"
		>
			<span class="sr-only">Bestand, gemessen am Bedarf im nächsten Schuljahr:</span>
			{r.gesamt_bestand} von {r.bedarf}
		</div>
		<!-- Ein Knopf mit der Zahl statt eines Plus: Was er tut, steht da, ohne zu zeigen. M3 Buttons:
		     „The outlined style is ideal for medium-emphasis buttons which contain actions that are
		     important, but aren't the primary action"; das Symbol „on the leading side". Der Name
		     beginnt mit dem sichtbaren Text, der Titel dahinter unterscheidet die Zeilen. -->
		<div class="{SPALTEN.knopf} flex justify-end">
			<Button variant="secondary" data-hinzufuegen onclick={() => onAddToCart(r, r.fehlen)}>
				<Icon name="add" />
				{r.fehlen} hinzufügen<span class="sr-only">: {r.titel}</span>
			</Button>
		</div>
	</div>

	{#if onNeueAuflage}
		<span class="order-1 shrink-0 @3xl:order-2">
			<Menue
				etikett="Weitere Aktionen zu {r.titel}"
				{eintraege}
				onwahl={() => onNeueAuflage?.(r)}
			/>
		</span>
	{/if}
</li>
