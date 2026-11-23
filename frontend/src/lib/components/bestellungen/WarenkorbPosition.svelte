<!-- @component Eine Position im Warenkorb — Cover, Titel, Menge, Preis, Topf-Wechsel.

     Eigene Datei, weil der Warenkorb nach Topf gruppiert: Warenkorb trägt die Gruppen und
     den Absenden-Knopf, diese Datei die Zeile. Beides zusammen läge über der
     200-Zeilen-Marke. -->
<script>
	import { bestellStore } from '../../stores/bestellStore.svelte.js';
	import Feld from '../ui/Feld.svelte';
	import BuchCover from '../ui/BuchCover.svelte';
	import StatusChip from '../ui/StatusChip.svelte';
	import { MITTEL, andereMittel } from './mittel.js';
	import Icon from '../ui/Icon.svelte';
	import Button from '../ui/Button.svelte';

	/** @type {{ position: import('../../stores/bestellStore.svelte.js').CartItem }} */
	let { position } = $props();

	/** Wohin die Position wechseln kann — jeder Knopf nennt sein ZIEL, nicht den Zustand. */
	const ziele = $derived(andereMittel(position.mittel));
</script>

<div class="space-y-2.5 rounded-xl border border-outline-variant bg-surface-container-lowest p-3">
	<div class="flex items-start gap-2.5">
		<BuchCover
			coverUrl={position.cover_url}
			isbn={position.isbn}
			titel={position.titel}
			klasse="shrink-0"
			dekorativ
		/>
		<div class="min-w-0 flex-1">
			<h4 class="truncate text-title-small leading-snug text-on-surface">
				{position.titel}
			</h4>
			<p class="truncate font-mono text-xs text-on-surface-variant">{position.isbn || '–'}</p>
			{#if position.generate_barcodes}
				<div class="mt-1">
					<StatusChip
						icon="label"
						text={position.menge === 1 ? '1 Barcode' : `${position.menge} Barcodes`}
					/>
				</div>
			{/if}
		</div>
		<Button
			variant="symbol"
			size="sm"
			loeschen
			onclick={() => bestellStore.removeFromCart(position)}
			aria-label="Entfernen"
			data-tip="Entfernen"
			class="shrink-0"
		>
			<Icon name="close" />
		</Button>
	</div>

	<div class="flex items-center justify-between gap-2 pl-10">
		<div
			class="flex items-center rounded-sm border border-outline-variant bg-surface-container-lowest"
		>
			<!-- Unter 1 geht die Menge nicht; entfernt wird die Position über das Kreuz. Der Knopf
			     ist dann gesperrt, statt ohne Wirkung zu bleiben. -->
			<Button
				variant="symbol"
				size="sm"
				aria-label="Menge verringern"
				data-tip="Menge verringern"
				disabled={position.menge <= 1}
				onclick={() => (position.menge = Math.max(1, position.menge - 1))}
			>
				<Icon name="remove" />
			</Button>
			<span
				class="min-w-6 px-2 text-center text-body-medium-emphasized text-on-surface tabular-nums"
				>{position.menge}</span
			>
			<Button
				variant="symbol"
				size="sm"
				aria-label="Menge erhöhen"
				data-tip="Menge erhöhen"
				onclick={() => (position.menge += 1)}
			>
				<Icon name="add" />
			</Button>
		</div>
		{#if bestellStore.preiseErfassen}
			<div class="flex items-center gap-1.5">
				<!-- Der Vorschlag bleibt als solcher erkennbar, solange er unverändert ist: Er ist
				     der DNB-Ladenpreis bei Erscheinen, nicht der Preis, den die Schule zahlt. Wer
				     ihn überschreibt, verliert das Abzeichen. -->
				{#if position.preis_vorschlag > 0 && Number(position.preis) === position.preis_vorschlag}
					<StatusChip
						ton="warten"
						text="DNB"
						tip="Ladenpreis aus dem DNB-Datensatz, bitte gegen den Schulpreis prüfen"
					/>
				{/if}
				<Feld
					type="number"
					step="0.01"
					bind:value={position.preis}
					aria-label="Preis"
					feld="w-20 text-right"
				/>
				<span class="text-body-medium-emphasized text-on-surface-variant">€</span>
			</div>
		{/if}
	</div>

	<!-- Der Topf ist eine Entscheidung je Bestellung; der Titel schlägt ihn nur vor. Ein
	     falsch gekennzeichnetes Schulbuch wandert hier in die Lernmittel-Bestellung, eine
	     Spende in die Bestellung ohne Topf, ohne dass jemand erst den Titel bearbeiten muss.
	     Jeder Knopf nennt sein Ziel. -->
	<div class="flex flex-wrap gap-x-1 pl-10">
		{#each ziele as ziel (ziel)}
			<Button
				variant="ghost"
				size="sm"
				onclick={() => bestellStore.verschiebe(position, ziel)}
				aria-label="{position.titel} in die Bestellung „{MITTEL[ziel].label}“ verschieben"
			>
				→ {MITTEL[ziel].label} · {MITTEL[ziel].traeger}
			</Button>
		{/each}
	</div>
</div>
