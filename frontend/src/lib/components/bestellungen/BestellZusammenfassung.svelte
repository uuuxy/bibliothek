<!-- @component Die Zusammenfassung der Bestellung: Lieferant, Summe je Topf, Gesamt und der eine
     Knopf, der die Mail an den Händler schickt. Wie die Zusammenfassung im Warenkorb des Google
     Store steht sie neben der Liste und bleibt beim Scrollen stehen (Bestellwesen.svelte); was
     sich bedienen lässt, steht in der Liste. -->
<script>
	import { bestellStore } from '../../stores/bestellStore.svelte.js';
	import Select from '../ui/Select.svelte';
	import Kaestchen from '../ui/Kaestchen.svelte';
	import Button from '../ui/Button.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { formatEuro, anzahlText } from '../../utils/format.js';

	const anzahlBestellungen = $derived(bestellStore.gruppen.length);
	const exemplare = $derived(anzahlText(bestellStore.totalQty, 'Exemplar', 'Exemplare'));
</script>

<div>
	<h2 class="pb-4 border-b border-outline-variant text-lg text-on-surface tracking-tight">
		Zusammenfassung
	</h2>
	<div class="pt-5 space-y-5">
		<div class="space-y-1.5">
			<label for="supplier" class="block text-sm font-medium text-on-surface-variant"
				>Lieferant</label
			>
			<!-- Wie in den Bestellberichten: Kam die Liste nicht, wäre „Kein Lieferant angelegt" eine
			     falsche Auskunft. -->
			{#if bestellStore.nichtGeladen.lieferanten && bestellStore.suppliers.length === 0}
				<LadeFehler knapp titel="Lieferanten nicht geladen" />
			{:else}
				<Select
					id="supplier"
					bind:value={bestellStore.selectedSupplierId}
					options={bestellStore.suppliers.map((s) => ({
						value: s.id,
						label: s.name,
						zusatz: s.customerNumber
					}))}
					placeholder="Kein Lieferant angelegt"
				/>
			{/if}
		</div>

		{#if bestellStore.cart.length}
			<!-- Je Topf eine Bestellung mit eigener Summe (Warenkorb.svelte). -->
			<ul class="space-y-3 border-b border-outline-variant pb-3">
				{#each bestellStore.gruppen as gruppe (gruppe.mittel)}
					<li class="flex items-end justify-between gap-3">
						<div class="min-w-0">
							<p class="text-body-small-emphasized text-on-surface">
								{gruppe.label}
								<span class="font-normal text-on-surface-variant">· {gruppe.traeger}</span>
							</p>
							<p class="text-xs text-on-surface-variant tabular-nums">
								{anzahlText(gruppe.menge, 'Exemplar', 'Exemplare')}
							</p>
						</div>
						{#if bestellStore.preiseErfassen}
							<span class="text-sm text-on-surface tabular-nums">{formatEuro(gruppe.summe)}</span>
						{/if}
					</li>
				{/each}
			</ul>

			<!-- Ohne Preiserfassung stuende hier dauerhaft 0,00 €: eine Summe, die wie ein
			     Betrag aussieht und keiner ist. Dann lieber die Menge, die feststeht. -->
			<div class="flex items-center justify-between">
				{#if bestellStore.preiseErfassen}
					<span class="text-body-medium-emphasized text-on-surface-variant">Gesamt</span>
					<span class="text-xl font-bold text-on-surface tabular-nums"
						>{formatEuro(bestellStore.total)}</span
					>
				{:else}
					<span class="text-body-medium-emphasized text-on-surface-variant">Exemplare</span>
					<span class="text-xl font-bold text-on-surface tabular-nums">{bestellStore.totalQty}</span
					>
				{/if}
			</div>
			<Kaestchen bind:checked={bestellStore.attachBarcodes} etikett="Barcodes mitschicken" />
			<!-- Eigener Block: Kästchen und Knopf sind beide inline; ab 600 px stand der Knopf
			     sonst in der Zeile des Kästchens, sobald beide hineinpassten. -->
			<div class="space-y-2">
				<Button
					size="lg"
					onclick={() => bestellStore.submitOrder()}
					disabled={bestellStore.submitting || !bestellStore.selectedSupplier}
					class="w-full sm:w-auto"
				>
					{#if bestellStore.submitting}
						<Ladekreis size="sm" farbe="aktuell" />
						Wird gesendet …
					{:else if anzahlBestellungen > 1}
						{anzahlBestellungen} Bestellungen auslösen · {exemplare}
					{:else}
						Bestellung auslösen · {exemplare}
					{/if}
				</Button>
				{#if anzahlBestellungen > 1}
					<!-- Kein Hinweis, der erschrickt, sondern die Ansage, was gleich passiert: Der
					     Händler bekommt zwei Mails und stellt zwei Rechnungen — so will es die
					     getrennte Finanzierung von Land und Schulträger. -->
					<p class="text-label-small text-on-surface-variant">
						Lernmittel und Bücherei gehen als getrennte Bestellungen an denselben Händler
					</p>
				{/if}
				{#if !bestellStore.selectedSupplier}
					<p class="text-label-small text-warning">Bitte zuerst einen Lieferanten wählen</p>
				{/if}
			</div>
		{/if}
	</div>
</div>
