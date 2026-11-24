<!-- @component Der Warenkorb — nach Topf gruppiert.

     Eine Bestellung = ein Topf (Migration 109): Lernmittel bezahlt das Land im Rahmen der
     Lernmittelfreiheit, die Schülerbücherei der Schulträger. Der Händler gewährt darauf
     verschiedene Nachlässe, und die Rechnungen gehen getrennte Wege — er kann eine
     gemischte Bestellung weder richtig rabattieren noch richtig abrechnen. Deshalb zeigt
     der Warenkorb seine Positionen in Abschnitten mit eigener Summe, und „Bestellung
     auslösen" (BestellZusammenfassung) erzeugt je Abschnitt eine Bestellung an denselben
     Händler. -->
<script>
	import { bestellStore } from '../../stores/bestellStore.svelte.js';
	import Icon from '../ui/Icon.svelte';
	import WarenkorbPosition, { STELLEN, SPALTEN } from './WarenkorbPosition.svelte';
	import { formatEuro, anzahlText } from '../../utils/format.js';
</script>

<div class="space-y-3">
	<div class="flex items-center justify-between">
		<span class="text-xs font-medium text-on-surface-variant">Warenkorb</span>
		{#if bestellStore.cart.length}
			<span
				class="text-label-medium text-on-surface-variant bg-surface-container rounded-full px-2 py-0.5 tabular-nums"
				>{anzahlText(bestellStore.totalQty, 'Exemplar', 'Exemplare')}</span
			>
		{/if}
	</div>

	{#if !bestellStore.cart.length}
		<div
			class="flex flex-col items-center gap-1.5 py-10 px-4 border border-dashed border-outline-variant rounded-xl text-center text-sm text-on-surface-variant"
		>
			<Icon name="shopping_cart" groesse={24} />
			<p>Noch nichts ausgewählt</p>
			<!-- Svelte nimmt den Leerraum am Anfang eines Blocks weg; das geschützte Leerzeichen bleibt,
			     wie vor dem Punkt im Kopf der Töpfe. -->
			<p>
				Titel oben suchen{#if bestellStore.recommendations.length}&nbsp;oder unten aus dem
					Bestellbedarf hinzufügen{/if}
			</p>
		</div>
	{:else}
		{#each bestellStore.gruppen as gruppe (gruppe.mittel)}
			<!-- Abschnittskopf: der Topf und seine Summe. Bei nur EINEM Abschnitt steht er
			     trotzdem da — er ist der Vermerk, den der Händler auf der Bestellung liest. -->
			<section aria-label="Bestellung {gruppe.label}">
				<div class="flex items-baseline justify-between gap-2 border-b border-outline-variant pb-1">
					<h3 class="text-label-medium text-on-surface">
						{gruppe.label}
						<span class="font-normal text-on-surface-variant">· {gruppe.traeger}</span>
					</h3>
					<span class="text-xs text-on-surface-variant tabular-nums">
						{anzahlText(
							gruppe.menge,
							'Exemplar',
							'Exemplare'
						)}{#if bestellStore.preiseErfassen}&nbsp;·
							{formatEuro(gruppe.summe)}{/if}
					</span>
				</div>
				<!-- Spaltenkopf nur, wenn die Zeile Spalten hat; schmal steht alles unter dem Titel.
				     Die Felder tragen ihren Namen selbst (aria-label), der Kopf ist nur zum Sehen. -->
				<div
					aria-hidden="true"
					class="hidden items-center gap-x-3 pt-2 text-xs font-medium text-on-surface-variant select-none @3xl:flex"
				>
					<span class="w-7 shrink-0"></span>
					<span class="flex-1">Titel</span>
					<span class={STELLEN}>
						<span class={SPALTEN.topf}>Topf</span>
						<span class={SPALTEN.menge}>Menge</span>
						{#if bestellStore.preiseErfassen}
							<span class="{SPALTEN.preis} pr-4 text-right">Preis</span>
						{/if}
					</span>
					<span class="order-2 w-8 shrink-0"></span>
				</div>
				<ul class="divide-y divide-outline-variant">
					{#each gruppe.items as item (item.id)}
						<WarenkorbPosition position={item} />
					{/each}
				</ul>
			</section>
		{/each}
	{/if}
</div>
