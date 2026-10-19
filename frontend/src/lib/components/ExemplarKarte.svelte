<script>
	import ExemplarBarcodeBearbeiten from './ExemplarBarcodeBearbeiten.svelte';
	import ExemplarStatusBearbeiten from './ExemplarStatusBearbeiten.svelte';
	import ExemplarZustand from './ExemplarZustand.svelte';
	import { exemplarStatus } from './exemplarStatus.js';
	import StatusChip from './ui/StatusChip.svelte';
	import Kaestchen from './ui/Kaestchen.svelte';
	import { Pencil, Plus, Printer, Trash } from '@lucide/svelte';

	/**
	 * Einzelne Exemplar-Karte. Verwaltet ihren eigenen Bearbeitungsmodus
	 * (Barcode/Status) lokal; Auswahl, Löschen und Etikett laufen über Callbacks zum Eltern-Tab.
	 * @type {{ ex: any, gewaehlt: boolean, darfBearbeiten?: boolean, onToggleSelect: () => void, onDelete: () => void, onEtikett?: () => void }}
	 */
	let { ex, gewaehlt, darfBearbeiten = false, onToggleSelect, onDelete, onEtikett } = $props();

	let barcodeBearbeiten = $state(false);

	let zustandBearbeiten = $state(false);
	const status = $derived(exemplarStatus(ex));
	// Platzhalter (AUTO-, SYS-) bekommen statt des Etikett-Knopfs „Barcode scannen".
	const platzhalter = $derived(/^(AUTO|SYS)-/.test(ex.barcode_id));
</script>

<!-- Die Karte ist kein Knopf: In ihr liegen Barcode-Feld, Stift und Papierkorb, und ein
     Bedienelement im Bedienelement lässt sich nicht bedienen. Ausgewählt wird über das
     Kästchen. Sie trägt einen Rahmen und keinen Schatten. -->
<div
	class="rounded-xl border bg-surface-container-lowest p-4 transition-colors {gewaehlt
		? 'border-primary bg-primary-container/30 ring-1 ring-primary'
		: 'border-outline-variant hover:border-outline'}"
>
	<div class="flex items-start justify-between mb-3">
		{#if barcodeBearbeiten}
			<div class="flex-1 mr-2">
				<!-- Ein Platzhalter (AUTO-, SYS-) öffnet das Feld leer für den Scanner. -->
				<ExemplarBarcodeBearbeiten
					{ex}
					vorgabe={platzhalter ? '' : ex.barcode_id}
					onDone={() => (barcodeBearbeiten = false)}
				/>
			</div>
		{:else}
			<div class="flex items-center gap-3">
				{#if darfBearbeiten}
					<Kaestchen
						checked={gewaehlt}
						onchange={onToggleSelect}
						aria-label="Exemplar {ex.barcode_id} auswählen"
					/>
				{/if}
				<!-- Chip-Form (siehe StatusChip), die Warnfarbe heißt Platzhalternummer; nowrap,
				     weil „Barcode scannen" sonst wortweise umbrach. -->
				<div class="flex flex-wrap items-center gap-2">
					<span
						class="rounded-md px-2 py-0.5 font-mono text-xs font-bold whitespace-nowrap {platzhalter
							? 'bg-warning-container text-on-warning-container'
							: 'bg-primary-container text-on-primary-container'}"
					>
						{ex.barcode_id}
					</span>
					{#if darfBearbeiten}
						{#if onEtikett && !platzhalter}
							<button
								class="text-on-surface-variant hover:text-primary transition-colors cursor-pointer"
								aria-label="Etikett für {ex.barcode_id} im Druck-Center drucken"
								data-tip="Etikett im Druck-Center drucken"
								onclick={onEtikett}
							>
								<Printer class="w-3.5 h-3.5" aria-hidden="true" />
							</button>
						{/if}
						{#if platzhalter}
							<button
								class="text-xs px-2 py-1 bg-warning-container text-on-warning-container font-semibold rounded-lg cursor-pointer flex items-center gap-1 whitespace-nowrap"
								onclick={() => (barcodeBearbeiten = true)}
							>
								<Plus class="w-3.5 h-3.5" aria-hidden="true" />
								Barcode scannen
							</button>
						{:else}
							<button
								title="Barcode zuweisen/ändern"
								aria-label="Barcode zuweisen oder ändern"
								class="text-on-surface-variant hover:text-primary transition-colors cursor-pointer flex items-center gap-1"
								onclick={() => (barcodeBearbeiten = true)}
							>
								<Pencil class="w-3.5 h-3.5" aria-hidden="true" />
							</button>
						{/if}
					{/if}
				</div>
			</div>
		{/if}
		<div class="flex items-center gap-2">
			<StatusChip {...status} />
			{#if !zustandBearbeiten}
				<button
					title="Status ändern"
					aria-label="Status ändern"
					class="text-on-surface-variant hover:text-primary transition-colors cursor-pointer"
					onclick={() => {
						zustandBearbeiten = true;
					}}
				>
					<Pencil class="w-3.5 h-3.5" aria-hidden="true" />
				</button>
				<!-- Ein ausgesondertes Exemplar ist schon ausgebucht; zurück kommt es über den Status. -->
				{#if !ex.ist_ausgesondert}
					<button
						title="Exemplar löschen"
						aria-label="Exemplar löschen"
						class="text-on-surface-variant hover:text-error transition-colors cursor-pointer"
						onclick={onDelete}
					>
						<Trash class="w-3.5 h-3.5" aria-hidden="true" />
					</button>
				{/if}
			{/if}
		</div>
	</div>
	{#if zustandBearbeiten}
		<ExemplarStatusBearbeiten {ex} onDone={() => (zustandBearbeiten = false)} />
	{:else}
		<ExemplarZustand {ex} />
	{/if}
</div>
