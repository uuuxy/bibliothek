<!-- @component FolieLmfHeute — die Folie am Tag der Bücherausgabe oder des Büchertauschs: links
     groß, was heute ist, rechts die Stunden mit den Klassen wie auf einer Abfahrtstafel.
     Gezeichnet im Raster des Fernsehers, 864 × 450 innerhalb des Rands (Google TV, Layouts:
     „Always design at MDPI resolution at 960px * 540px“); Monitor.svelte vergrößert es auf den
     Bildschirm. Die Zeilen sind eine Liste im segmentierten Stil (M3 Lists: „Use segmented gaps
     and filled list items to define a list group“). -->
<script module>
	// „Cookie12Sided“ aus Googles Formenbibliothek (androidx MaterialShapes: RoundedPolygon.star
	// mit zwölf Ecken, Innenradius 0,8), als Kurve angenähert. Eine Form als Schmuck auf einer
	// Fläche ohne Bedienung, M3 Shape: „Emphasize aesthetic moments with shape“.
	const KEKS = `polygon(${Array.from({ length: 240 }, (_, i) => {
		const w = (i / 240) * 2 * Math.PI;
		const r = 45 + 5 * Math.cos(12 * w);
		return `${(50 + r * Math.cos(w)).toFixed(2)}% ${(50 + r * Math.sin(w)).toFixed(2)}%`;
	}).join(', ')})`;

	/** Segmentierte Liste: außen 16, innen 4 (M3 Lists, Specs: „4dp inner, 16dp outer“).
	 *  @param {number} i @param {number} n */
	function ecken(i, n) {
		if (n === 1) return 'rounded-2xl';
		if (i === 0) return 'rounded-t-2xl rounded-b-sm';
		if (i === n - 1) return 'rounded-t-sm rounded-b-2xl';
		return 'rounded-sm';
	}
</script>

<script>
	import Icon from '../components/ui/Icon.svelte';
	import { langesDatum } from '../utils/datum.js';
	import { lmfTitel } from './monitorTakt.svelte.js';

	/** @type {{ heute: import('./monitorTakt.svelte.js').LmfHeute }} */
	let { heute } = $props();

	const titel = $derived(lmfTitel(heute.art));
	const mitHinweis = $derived(heute.termine.some((t) => t.vermerk !== '' && t.klassen.length > 0));
	// Mehr als sechs Zeilen passen nur in der kleineren Schrift auf die Folie.
	const dicht = $derived(heute.termine.length > 6);
</script>

<div class="flex size-full gap-4">
	<section
		class="flex w-98 shrink-0 flex-col justify-between rounded-3xl bg-primary-container p-8 text-on-primary-container"
	>
		<!-- Die Form dreht sich, das Buch darin bleibt gerade. -->
		<div class="keks size-40 bg-primary text-on-primary" style:clip-path={KEKS}>
			<div class="gerade flex size-full items-center justify-center">
				<Icon name="menu_book" class="[--icon-groesse:80px]" />
			</div>
		</div>
		<div>
			<h2 class="text-display-medium-emphasized">
				<span class="block">Heute</span>
				<span class="block">{titel}</span>
			</h2>
			<p class="mt-3 text-lg">{langesDatum(heute.datum)}</p>
		</div>
	</section>

	<section class="flex min-w-0 flex-1 flex-col justify-center">
		<!-- Die Köpfe der Spalten in label-large: So passt „Stunde“ über den Avatar der Zeile. -->
		<div class="flex items-center gap-4 pb-2 pl-4 pr-6 text-label-large text-on-surface-variant">
			<span class="w-10 shrink-0 text-center">Stunde</span>
			<span class="flex-1">Klasse</span>
			{#if mitHinweis}<span>Hinweis</span>{/if}
		</div>
		<ol class="flex flex-col gap-0.5">
			{#each heute.termine as t, i (i)}
				<li
					class="flex items-center gap-4 bg-surface-container-high pl-4 pr-6 {dicht
						? 'py-0'
						: 'py-2'} {ecken(i, heute.termine.length)}"
				>
					<!-- Die Stunde als Avatar der Zeile, nach Googles Token (material-web, md.comp.list:
					     leading-avatar 40px, corner-full, primary-container, Schrift title-medium). -->
					<span
						class="flex shrink-0 items-center justify-center rounded-full bg-primary-container text-title-medium text-on-primary-container {dicht
							? 'size-8'
							: 'size-10'}">{t.stunde}</span
					>
					{#if t.klassen.length > 0}
						<span
							class="min-w-0 flex-1 truncate text-on-surface {dicht
								? 'text-headline-small-emphasized'
								: 'text-display-small-emphasized'}">{t.klassen.join(', ')}</span
						>
						{#if t.vermerk}
							<span class="max-w-1/2 truncate text-lg text-on-surface-variant">{t.vermerk}</span>
						{/if}
					{:else}
						<span
							class="min-w-0 flex-1 truncate text-on-surface-variant {dicht
								? 'text-xl'
								: 'text-4xl'}">{t.vermerk}</span
						>
					{/if}
				</li>
			{/each}
		</ol>
	</section>
</div>

<style>
	.keks {
		animation: drehen 40s linear infinite;
	}
	.gerade {
		animation: drehen 40s linear infinite reverse;
	}
	@keyframes drehen {
		to {
			transform: rotate(360deg);
		}
	}
</style>
