<!-- @component LabelBarcodeSchritt — Schritt 2 des Druck-Centers: WELCHE Barcodes
     aufs Blatt kommen. Zwei Wege, die einander ausschließen — vorhandene Exemplare
     abhaken oder eine Reihe neuer Nummern erzeugen.

     Der Platzhalter im else-Zweig ist Absicht: Ohne ihn spränge die Schrittfolge von
     1 auf 3 und sähe aus wie ein übersprungener Schritt. -->
<script>
	import { labelStore } from '../../stores/labels.svelte.js';
</script>

{#if labelStore.selectedTitle}
	<div class="py-5 space-y-4 border-b border-slate-200">
		<h3 class="text-xs text-blue-600 font-medium">2. Barcodes generieren</h3>

		<!-- Selection mode -->
		<div class="flex bg-slate-100 p-0.5 rounded-xl border border-slate-200/40 text-xs">
			<button
				onclick={() => (labelStore.generationMode = 'existing')}
				class="flex-1 text-center py-1.5 rounded-lg font-bold transition-all cursor-pointer {labelStore.generationMode ===
				'existing'
					? 'bg-white text-slate-800 shadow-xs'
					: 'text-slate-500 hover:text-slate-700'}">Vorhandene Exemplare</button
			>
			<button
				onclick={() => (labelStore.generationMode = 'new')}
				class="flex-1 text-center py-1.5 rounded-lg font-bold transition-all cursor-pointer {labelStore.generationMode ===
				'new'
					? 'bg-white text-slate-800 shadow-xs'
					: 'text-slate-500 hover:text-slate-700'}">Neue Barcodes</button
			>
		</div>

		{#if labelStore.generationMode === 'existing'}
			<div class="space-y-2">
				<span class="text-xs font-medium text-slate-500 block"
					>Exemplare auswählen ({labelStore.existingCopies.length} gefunden)</span
				>
				{#if labelStore.loadingCopies}
					<div class="flex items-center justify-center py-4">
						<div
							class="w-5 h-5 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"
						></div>
					</div>
				{:else if labelStore.existingCopies.length === 0}
					<p class="text-label-small text-slate-500">
						Keine physischen Exemplare in der Datenbank vorhanden.
					</p>
				{:else}
					<div
						class="max-h-40 overflow-y-auto border border-slate-100 rounded-xl divide-y divide-slate-50 p-2 space-y-1 bg-slate-50/50"
					>
						{#each labelStore.existingCopies as copy, _i (_i)}
							<label
								class="flex items-center space-x-3 text-xs text-slate-700 cursor-pointer p-1.5 hover:bg-slate-50 rounded-lg"
							>
								<input
									type="checkbox"
									bind:checked={copy.checked}
									class="accent-blue-600 w-4 h-4 rounded border-slate-200 bg-white"
								/>
								<span class="font-bold text-slate-800">{copy.barcode_id}</span>
								<span class="text-label-small text-slate-500 font-sans"
									>({copy.zustand_notiz || 'Neuwertig'})</span
								>
							</label>
						{/each}
					</div>
				{/if}
			</div>
		{:else}
			<!-- Generating new sequential labels -->
			<div class="grid grid-cols-2 gap-3">
				<div class="space-y-1.5">
					<span class="text-xs font-medium text-slate-500 block">Menge</span>
					<input
						type="number"
						min="1"
						max="100"
						bind:value={labelStore.newQuantity}
						class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-sm text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500"
					/>
				</div>
				<div class="space-y-1.5">
					<span class="text-xs font-medium text-slate-500 block">Start-Ziffer (B-)</span>
					<input
						type="number"
						min="1"
						bind:value={labelStore.newStartNum}
						class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-sm text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500"
					/>
				</div>
			</div>
		{/if}
	</div>
{:else}
	<!-- Platzhalter, damit die Schrittfolge nicht von 1 auf 3 springt (wirkt sonst
	     wie ein übersprungener Schritt). Wird aktiv, sobald ein Titel gewählt ist. -->
	<div class="py-5 space-y-2 border-b border-slate-200 opacity-60">
		<h3 class="text-xs text-slate-400 font-medium">2. Barcodes generieren</h3>
		<p class="text-label-small text-slate-400">Zuerst oben einen Titel oder Klassensatz wählen.</p>
	</div>
{/if}
