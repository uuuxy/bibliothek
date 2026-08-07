<script>
	import { appState } from '../../inventur/lib/store.svelte.js';
	import { apiFetch, apiClient } from '../apiFetch.js';
	import BookExemplarStatusEditor from './BookExemplarStatusEditor.svelte';
	import Button from './ui/Button.svelte';
	import { Pencil, Plus, Printer, Trash2 } from '@lucide/svelte';

	/**
	 * Einzelne Exemplar-Karte. Verwaltet ihren eigenen Bearbeitungsmodus
	 * (Barcode/Status) lokal; Auswahl & Löschen laufen über Callbacks zum Eltern-Tab.
	 * @type {{ ex: any, selected: boolean, onToggleSelect: () => void, onDelete: () => void }}
	 */
	let { ex, selected, onToggleSelect, onDelete } = $props();

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

<!-- Auswahl per Tastatur, nicht nur per Maus. Die Pruefung auf currentTarget ist hier
     wichtig: In der Karte liegt das Barcode-Eingabefeld, dessen Enter das Speichern
     ausloest — ohne die Pruefung wuerde derselbe Tastendruck zusaetzlich die Auswahl
     umschalten. -->
<div
	class="bg-white rounded-xl border p-4 shadow-sm transition-colors cursor-pointer {selected
		? 'border-blue-500 bg-blue-50/30 ring-1 ring-blue-500'
		: 'border-slate-200 hover:border-slate-300'}"
	role="button"
	tabindex="0"
	aria-pressed={selected}
	onclick={() => {
		if (appState.adminAuthenticated) onToggleSelect();
	}}
	onkeydown={(e) => {
		if (e.target !== e.currentTarget) return;
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			if (appState.adminAuthenticated) onToggleSelect();
		}
	}}
>
	<div class="flex items-start justify-between mb-3">
		{#if editingBarcode}
			<div class="flex-1 mr-2 relative">
				<!-- svelte-ignore a11y_autofocus -->
				<!-- Bewusst behalten: Das Feld erscheint erst auf Klick und ersetzt an dieser
				     Stelle den Barcode. Wer es oeffnet, will sofort tippen oder scannen. -->
				<input
					type="text"
					bind:value={editBarcodeValue}
					autofocus
					onfocus={(e) => e.currentTarget.select()}
					class="w-full px-2 py-1 text-xs font-mono border {barcodeError
						? 'border-rose-500 bg-rose-50 text-rose-700'
						: 'border-blue-300'} rounded focus:outline-none focus:ring-2 focus:ring-blue-500/30"
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
				{#if appState.adminAuthenticated}
					<input
						type="checkbox"
						checked={selected}
						class="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500 cursor-pointer pointer-events-none"
					/>
				{/if}
				<div class="flex items-center gap-2">
					<span
						class="text-xs font-bold {ex.barcode_id.startsWith('AUTO-') ||
						ex.barcode_id.startsWith('SYS-')
							? 'text-amber-700 bg-amber-50 border-amber-100'
							: 'text-blue-700 bg-blue-50 border-blue-100'} border px-2 py-0.5 rounded font-mono"
					>
						{ex.barcode_id}
					</span>
					{#if appState.adminAuthenticated}
						{#if ex.barcode_id.startsWith('B-')}
							<a
								href={`/api/print/etikett/${ex.id}`}
								target="_blank"
								title="Ersatz-Etikett drucken"
								class="text-slate-400 hover:text-blue-600 transition-colors cursor-pointer flex items-center gap-1"
								onclick={(e) => e.stopPropagation()}
							>
								<Printer class="w-3.5 h-3.5" aria-hidden="true" />
							</a>
						{/if}
						{#if ex.barcode_id.startsWith('AUTO-') || ex.barcode_id.startsWith('SYS-')}
							<button
								class="text-xs px-2 py-1 bg-amber-100 hover:bg-amber-200 text-amber-800 font-semibold rounded shadow-sm transition-colors cursor-pointer flex items-center gap-1"
								onclick={(e) => {
									e.stopPropagation();
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
								class="text-slate-400 hover:text-blue-600 transition-colors cursor-pointer flex items-center gap-1"
								onclick={(e) => {
									e.stopPropagation();
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
			<span
				class="text-label-small font-bold px-2 py-0.5 rounded-full {!ex.ist_ausleihbar
					? 'bg-rose-50 text-rose-700 border border-rose-100'
					: !ex.ist_verfuegbar
						? 'bg-amber-50 text-amber-700 border border-amber-100'
						: 'bg-emerald-50 text-emerald-700 border border-emerald-100'}"
			>
				{!ex.ist_ausleihbar ? 'Gesperrt' : !ex.ist_verfuegbar ? 'Ausgeliehen' : 'Verfügbar'}
			</span>
			{#if !editingStatus}
				<button
					title="Status ändern"
					aria-label="Status ändern"
					class="text-slate-400 hover:text-blue-600 transition-colors cursor-pointer"
					onclick={(e) => {
						e.stopPropagation();
						editingStatus = true;
					}}
				>
					<Pencil class="w-3.5 h-3.5" aria-hidden="true" />
				</button>
				<button
					title="Exemplar löschen"
					aria-label="Exemplar löschen"
					class="text-slate-400 hover:text-rose-600 transition-colors cursor-pointer"
					onclick={(e) => {
						e.stopPropagation();
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
	{:else if ex.zustand_notiz}
		<p class="text-xs text-slate-500">
			<span class="font-semibold text-slate-400">Zustand:</span>
			{ex.zustand_notiz}
		</p>
	{/if}
</div>
