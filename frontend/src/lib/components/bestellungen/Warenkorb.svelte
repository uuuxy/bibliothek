<!-- @component Der Warenkorb — nach Topf gruppiert, ein Absenden-Knopf für alle.

     Eine Bestellung = ein Topf (Migration 109): Lernmittel bezahlt das Land im Rahmen der
     Lernmittelfreiheit, die Schülerbücherei der Schulträger. Der Händler gewährt darauf
     verschiedene Nachlässe, und die Rechnungen gehen getrennte Wege — er kann eine
     gemischte Bestellung weder richtig rabattieren noch richtig abrechnen. Deshalb zeigt
     der Warenkorb seine Positionen in zwei Abschnitten mit eigener Summe, und
     „Bestellung auslösen" erzeugt je Abschnitt eine Bestellung an denselben Händler. -->
<script>
	import { bestellStore } from '../../stores/bestellStore.svelte.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import Button from '../ui/Button.svelte';
	import Kaestchen from '../ui/Kaestchen.svelte';
	import WarenkorbPosition from './WarenkorbPosition.svelte';
	import { formatEuro } from '../../utils/format.js';

	const anzahlBestellungen = $derived(bestellStore.gruppen.length);
</script>

<div class="space-y-3">
	<div class="flex items-center justify-between">
		<span class="text-xs font-medium text-on-surface-variant">Warenkorb</span>
		{#if bestellStore.cart.length}
			<span
				class="text-xs font-bold text-on-surface-variant bg-surface-container rounded-full px-2 py-0.5 tabular-nums"
				>{bestellStore.totalQty} Expl.</span
			>
		{/if}
	</div>

	{#if !bestellStore.cart.length}
		<div
			class="py-10 px-4 border border-dashed border-outline-variant rounded-xl text-center text-sm text-on-surface-variant"
		>
			<div class="text-2xl mb-1.5">🛒</div>
			Noch nichts ausgewählt.<br />
			Tippe links bei einem Titel auf <span class="font-bold">+</span> oder suche oben.
		</div>
	{:else}
		{#each bestellStore.gruppen as gruppe (gruppe.mittel)}
			<!-- Abschnittskopf: der Topf und seine Summe. Bei nur EINEM Abschnitt steht er
			     trotzdem da — er ist der Vermerk, den der Händler auf der Bestellung liest. -->
			<section aria-label="Bestellung {gruppe.label}" class="space-y-2">
				<div class="flex items-baseline justify-between gap-2 border-b border-outline-variant pb-1">
					<h3 class="text-xs font-semibold text-on-surface">
						{gruppe.label}
						<span class="font-normal text-on-surface-variant">({gruppe.traeger})</span>
					</h3>
					<span class="text-xs text-on-surface-variant tabular-nums">
						{gruppe.menge} Expl.{#if bestellStore.preiseErfassen}
							· {formatEuro(gruppe.summe)}{/if}
					</span>
				</div>
				{#each gruppe.items as item (item.id)}
					<WarenkorbPosition position={item} />
				{/each}
			</section>
		{/each}

		<!-- Footer: Summe + CTA -->
		<div class="pt-3 mt-1 border-t border-outline-variant space-y-3">
			<!-- Ohne Preiserfassung stuende hier dauerhaft 0,00 €: eine Summe, die wie ein
			     Betrag aussieht und keiner ist. Dann lieber die Menge, die feststeht. -->
			<div class="flex items-center justify-between">
				{#if bestellStore.preiseErfassen}
					<span class="text-sm font-semibold text-on-surface-variant">Gesamt</span>
					<span class="text-xl font-bold text-on-surface tabular-nums"
						>{formatEuro(bestellStore.total)}</span
					>
				{:else}
					<span class="text-sm font-semibold text-on-surface-variant">Exemplare</span>
					<span class="text-xl font-bold text-on-surface tabular-nums">{bestellStore.totalQty}</span
					>
				{/if}
			</div>
			<Kaestchen bind:checked={bestellStore.attachBarcodes} etikett="Barcodes mitschicken" />
			<Button
				size="lg"
				onclick={() => bestellStore.submitOrder()}
				disabled={bestellStore.submitting || !bestellStore.selectedSupplier}
				class="w-full"
			>
				{#if bestellStore.submitting}
					<Ladekreis size="sm" farbe="aktuell" />
					Wird gesendet …
				{:else if anzahlBestellungen > 1}
					{anzahlBestellungen} Bestellungen auslösen · {bestellStore.totalQty} Expl.
				{:else}
					Bestellung auslösen · {bestellStore.totalQty} Expl.
				{/if}
			</Button>
			{#if anzahlBestellungen > 1}
				<!-- Kein Hinweis, der erschrickt, sondern die Ansage, was gleich passiert: Der
				     Händler bekommt zwei Mails und stellt zwei Rechnungen — so will es die
				     getrennte Finanzierung von Land und Schulträger. -->
				<p class="text-label-small text-center text-on-surface-variant">
					Lernmittel und Bücherei gehen als getrennte Bestellungen an denselben Händler.
				</p>
			{/if}
			{#if !bestellStore.selectedSupplier}
				<p class="text-label-small text-center text-warning font-medium">
					Bitte zuerst einen Lieferanten wählen.
				</p>
			{/if}
		</div>
	{/if}
</div>
