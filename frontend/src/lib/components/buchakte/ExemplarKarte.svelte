<script>
	import ExemplarBarcodeBearbeiten from './ExemplarBarcodeBearbeiten.svelte';
	import ExemplarStatusBearbeiten from './ExemplarStatusBearbeiten.svelte';
	import ExemplarZustand from './ExemplarZustand.svelte';
	import { exemplarStatus } from './exemplarStatus.js';
	import StatusChip from '../ui/StatusChip.svelte';
	import Kaestchen from '../ui/Kaestchen.svelte';
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
	<!-- Zwei Aktionsbereiche, der Barcode und der Status, je mit seinen Symbolknöpfen. Ist die
	     Karte zu schmal für beide, steht der Status darunter. Abstand 16 px, auch zwischen den
	     Zeilen: So berühren sich die Trefferflächen von 48 px nicht. -->
	<div class="mb-3 flex flex-wrap items-center justify-between gap-4">
		{#if barcodeBearbeiten}
			<div class="min-w-0 flex-1">
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
						{#if platzhalter}
							<button
								class="text-xs px-2 py-1 bg-warning-container text-on-warning-container font-semibold rounded-lg cursor-pointer flex items-center gap-1 whitespace-nowrap"
								onclick={() => (barcodeBearbeiten = true)}
							>
								<Plus class="w-3.5 h-3.5" aria-hidden="true" />
								Barcode scannen
							</button>
						{:else}
							<div class="flex items-center gap-4">
								{#if onEtikett}
									<button
										class="icon-btn text-on-surface-variant hover:text-primary"
										aria-label="Etikett für {ex.barcode_id} im Druck-Center drucken"
										data-tip="Etikett im Druck-Center drucken"
										onclick={onEtikett}
									>
										<Printer class="h-4 w-4" aria-hidden="true" />
									</button>
								{/if}
								<button
									class="icon-btn text-on-surface-variant hover:text-primary"
									aria-label="Barcode zuweisen oder ändern"
									data-tip="Barcode zuweisen oder ändern"
									onclick={() => (barcodeBearbeiten = true)}
								>
									<Pencil class="h-4 w-4" aria-hidden="true" />
								</button>
							</div>
						{/if}
					{/if}
				</div>
			</div>
		{/if}
		<div class="flex items-center gap-2">
			<StatusChip {...status} />
			{#if !zustandBearbeiten}
				<div class="flex items-center gap-4">
					<button
						aria-label="Status ändern"
						data-tip="Status ändern"
						class="icon-btn text-on-surface-variant hover:text-primary"
						onclick={() => {
							zustandBearbeiten = true;
						}}
					>
						<Pencil class="h-4 w-4" aria-hidden="true" />
					</button>
					<!-- Ein ausgesondertes Exemplar ist schon ausgebucht; zurück kommt es über den Status. -->
					{#if !ex.ist_ausgesondert}
						<button
							aria-label="Exemplar löschen"
							data-tip="Exemplar löschen"
							class="icon-btn text-on-surface-variant hover:text-error"
							onclick={onDelete}
						>
							<Trash class="h-4 w-4" aria-hidden="true" />
						</button>
					{/if}
				</div>
			{/if}
		</div>
	</div>
	{#if zustandBearbeiten}
		<ExemplarStatusBearbeiten {ex} onDone={() => (zustandBearbeiten = false)} />
	{:else}
		<ExemplarZustand {ex} />
	{/if}
</div>
