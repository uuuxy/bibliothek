<script>
	import { apiFetch, apiClient } from '../apiFetch.js';
	import BookExemplarStatusEditor from './BookExemplarStatusEditor.svelte';
	import BookExemplarZustand from './BookExemplarZustand.svelte';
	import StatusChip from './ui/StatusChip.svelte';
	import Button from './ui/Button.svelte';
	import Feld from './ui/Feld.svelte';
	import Kaestchen from './ui/Kaestchen.svelte';
	import { Pencil, Plus, Printer, Trash2 } from '@lucide/svelte';

	/**
	 * Einzelne Exemplar-Karte. Verwaltet ihren eigenen Bearbeitungsmodus
	 * (Barcode/Status) lokal; Auswahl & Löschen laufen über Callbacks zum Eltern-Tab.
	 * @type {{ ex: any, selected: boolean, darfBearbeiten?: boolean, onToggleSelect: () => void, onDelete: () => void }}
	 */
	let { ex, selected, darfBearbeiten = false, onToggleSelect, onDelete } = $props();

	let editingBarcode = $state(false);
	let editBarcodeValue = $state('');
	let barcodeError = $state('');

	let editingStatus = $state(false);

	async function saveBarcode() {
		if (!editBarcodeValue.trim()) return;
		if (editBarcodeValue.trim() === ex.barcode_id) {
			editingBarcode = false;
			return;
		}
		barcodeError = '';
		try {
			const res = await apiClient.put(`/api/buecher/exemplare/${ex.id}/barcode`, {
				barcode: editBarcodeValue.trim()
			});
			if (res.ok) {
				ex.barcode_id = editBarcodeValue.trim();
				editingBarcode = false;
			} else {
				const errorData = await res.json().catch(() => ({}));
				barcodeError = errorData.error || 'Fehler beim Speichern';
			}
		} catch {
			barcodeError = 'Netzwerkfehler';
		}
	}

	async function generateInternalId() {
		try {
			const res = await apiFetch('/api/barcode/next');
			if (res.ok) {
				const data = await res.json();
				editBarcodeValue = data.next_barcode;
			} else {
				barcodeError = 'Fehler beim Generieren der ID';
			}
		} catch {
			barcodeError = 'Netzwerkfehler';
		}
	}
</script>

<!-- Die Karte ist KEIN Knopf. Bis zum 09.09.2026 trug sie role="button" mit
     aria-pressed — und darin lagen Barcode-Feld, Drucken-Link, Stift und Papierkorb:
     Bedienelemente im Bedienelement (nested-interactive, 50 Verstöße im
     Medienkatalog). Ausgewählt wird über das Kästchen, das ohnehin da war und bis
     dahin nur Anzeige war (pointer-events-none). -->
<!-- Rahmen XOR Erhebung: `shadow-sm` ist weg, der Rahmen dafür sichtbarer. -->
<div
	class="rounded-xl border bg-surface-container-lowest p-4 transition-colors {selected
		? 'border-primary bg-primary-container/30 ring-1 ring-primary'
		: 'border-outline-variant hover:border-outline'}"
>
	<div class="flex items-start justify-between mb-3">
		{#if editingBarcode}
			<div class="flex-1 mr-2 relative">
				<!-- autofocus bewusst: Das Feld erscheint erst auf Klick und ersetzt an dieser
				     Stelle den Barcode. Wer es oeffnet, will sofort tippen oder scannen. -->
				<Feld
					bind:value={editBarcodeValue}
					aria-label="Barcode des Exemplars"
					autofocus
					onfocus={(e) => e.currentTarget.select()}
					ungueltig={!!barcodeError}
					feld="font-mono"
					onkeydown={(e) => {
						if (e.key === 'Enter') saveBarcode();
						if (e.key === 'Escape') {
							editingBarcode = false;
							barcodeError = '';
						}
					}}
				/>
				<div class="mt-1 flex gap-2">
					<Button
						variant="secondary"
						size="sm"
						onclick={generateInternalId}
						class="text-label-small"
					>
						Interne ID generieren
					</Button>
					<Button size="sm" onclick={saveBarcode} class="text-label-small">Speichern</Button>
				</div>
				{#if barcodeError}
					<p
						class="text-label-small text-rose-600 mt-1 absolute -bottom-4 left-0 truncate w-full"
						title={barcodeError}
					>
						{barcodeError}
					</p>
				{/if}
			</div>
		{:else}
			<div class="flex items-center gap-3">
				{#if darfBearbeiten}
					<Kaestchen
						checked={selected}
						onchange={onToggleSelect}
						aria-label="Exemplar {ex.barcode_id} auswählen"
					/>
				{/if}
				<!-- Chip-Form (siehe StatusChip), amber heißt Platzhalternummer; nowrap,
				     weil „Barcode scannen" sonst wortweise umbrach. -->
				<div class="flex flex-wrap items-center gap-2">
					<span
						class="rounded-md px-2 py-0.5 font-mono text-xs font-bold whitespace-nowrap {ex.barcode_id.startsWith(
							'AUTO-'
						) || ex.barcode_id.startsWith('SYS-')
							? 'bg-amber-100 text-amber-700'
							: 'bg-primary-container text-on-primary-container'}"
					>
						{ex.barcode_id}
					</span>
					{#if darfBearbeiten}
						{#if ex.barcode_id.startsWith('B-')}
							<a
								href={`/api/print/etikett/${ex.id}`}
								target="_blank"
								title="Ersatz-Etikett drucken"
								class="text-on-surface-variant hover:text-primary transition-colors cursor-pointer flex items-center gap-1"
							>
								<Printer class="w-3.5 h-3.5" aria-hidden="true" />
							</a>
						{/if}
						{#if ex.barcode_id.startsWith('AUTO-') || ex.barcode_id.startsWith('SYS-')}
							<button
								class="text-xs px-2 py-1 bg-amber-100 hover:bg-amber-200 text-amber-700 font-semibold rounded-lg transition-colors cursor-pointer flex items-center gap-1 whitespace-nowrap"
								onclick={() => {
									editingBarcode = true;
									editBarcodeValue = ''; // Leer lassen für den Scanner
									barcodeError = '';
								}}
							>
								<Plus class="w-3.5 h-3.5" aria-hidden="true" />
								Barcode scannen
							</button>
						{:else}
							<button
								title="Barcode zuweisen/ändern"
								aria-label="Barcode zuweisen oder ändern"
								class="text-on-surface-variant hover:text-primary transition-colors cursor-pointer flex items-center gap-1"
								onclick={() => {
									editingBarcode = true;
									editBarcodeValue = ex.barcode_id;
									barcodeError = '';
								}}
							>
								<Pencil class="w-3.5 h-3.5" aria-hidden="true" />
							</button>
						{/if}
					{/if}
				</div>
			</div>
		{/if}
		<div class="flex items-center gap-2">
			<StatusChip
				ton={!ex.ist_ausleihbar ? 'fehler' : !ex.ist_verfuegbar ? 'warten' : 'erfolg'}
				text={!ex.ist_ausleihbar ? 'Gesperrt' : !ex.ist_verfuegbar ? 'Ausgeliehen' : 'Verfügbar'}
			/>
			{#if !editingStatus}
				<button
					title="Status ändern"
					aria-label="Status ändern"
					class="text-on-surface-variant hover:text-primary transition-colors cursor-pointer"
					onclick={() => {
						editingStatus = true;
					}}
				>
					<Pencil class="w-3.5 h-3.5" aria-hidden="true" />
				</button>
				<button
					title="Exemplar löschen"
					aria-label="Exemplar löschen"
					class="text-on-surface-variant hover:text-error transition-colors cursor-pointer"
					onclick={() => {
						onDelete();
					}}
				>
					<Trash2 class="w-3.5 h-3.5" aria-hidden="true" />
				</button>
			{/if}
		</div>
	</div>
	{#if editingStatus}
		<BookExemplarStatusEditor {ex} onDone={() => (editingStatus = false)} />
	{:else}
		<BookExemplarZustand {ex} />
	{/if}
</div>
