<script>
	import { labelStore } from '../../stores/labels.svelte.js';
	import { printQueue } from '../../stores/printQueue.svelte.js';
</script>

<div class="lg:col-span-5 space-y-6 text-left">
	{#if (printQueue.copies?.length ?? 0) > 0}
		<div class="p-4 border-l-2 border-blue-300 bg-blue-50/50 space-y-4 text-left animate-fade-in">
			<div class="flex items-start gap-2.5">
				<span class="text-lg">🖨️</span>
				<div>
					<h3 class="text-xs font-bold text-blue-800 uppercase tracking-wider">
						Aktiver Druckauftrag
					</h3>
					<p class="text-xs text-blue-700 font-medium leading-relaxed mt-1">
						Es werden {printQueue.copies?.length ?? 0} Etiketten aus der freigegebenen Lieferung geladen.
					</p>
				</div>
			</div>
			<button
				onclick={labelStore.resetPendingCopies}
				class="w-full py-2 bg-white hover:bg-slate-50 border border-slate-200 text-slate-700 font-bold rounded-xl text-xs transition-colors cursor-pointer"
			>
				Auswahl zurücksetzen / Anderes Buch wählen
			</button>
		</div>
	{:else}
		<!-- Step 1: Selection -->
		<div class="py-5 space-y-4 border-b border-gray-200">
			<h3 class="text-[10px] uppercase tracking-wider text-blue-600 font-bold">
				1. Titel / Klassensatz wählen
			</h3>

			<!-- Tab selector for search vs class set -->
			<div class="space-y-3">
				<!-- Autocomplete search -->
				<div class="space-y-1.5">
					<span class="text-[10px] uppercase font-bold text-slate-450 block"
						>Buchtitel im Katalog suchen</span
					>
					<div class="relative">
						<input
							type="text"
							bind:value={labelStore.searchVal}
							oninput={labelStore.handleSearchInput}
							placeholder="Titel, Autor oder ISBN eingeben..."
							class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500"
						/>
						{#if labelStore.isSearching}
							<div class="absolute right-3 top-1/2 -translate-y-1/2">
								<div
									class="w-3.5 h-3.5 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"
								></div>
							</div>
						{/if}
					</div>

					{#if labelStore.searchResults.length > 0}
						<div class="relative">
							<div
								class="absolute left-0 right-0 mt-1 bg-white border border-slate-100 rounded-xl shadow-xl z-20 max-h-48 overflow-y-auto divide-y divide-slate-50"
							>
								{#each labelStore.searchResults as r, _i (_i)}
									<button
										onclick={() => labelStore.selectBookTitle(r)}
										class="w-full text-left px-3.5 py-2.5 hover:bg-slate-50 transition-colors flex flex-col gap-0.5 cursor-pointer"
									>
										<span class="text-xs font-bold text-slate-900">{r.titel}</span>
										<span class="text-[10px] text-slate-450"
											>{r.autor || 'Unbekannt'} · {r.verlag || 'Kein Verlag'}</span
										>
									</button>
								{/each}
							</div>
						</div>
					{/if}
				</div>

				<!-- Divider -->
				<div class="relative flex py-1 items-center">
					<div class="grow border-t border-slate-100"></div>
					<span class="shrink mx-3 text-[9px] uppercase tracking-wider text-slate-400 font-bold"
						>ODER</span
					>
					<div class="grow border-t border-slate-100"></div>
				</div>

				<!-- Class Selection -->
				<div class="grid grid-cols-2 gap-3">
					<div class="space-y-1.5">
						<span class="text-[10px] uppercase font-bold text-slate-450 block"
							>Aus Klasse laden</span
						>
						<select
							bind:value={labelStore.selectedClass}
							onchange={labelStore.handleClassChange}
							class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-550"
						>
							<option value="">-- Klasse wählen --</option>
							{#each labelStore.classGroups as group, _i (_i)}
								<option value={group.className}>{group.className}</option>
							{/each}
						</select>
					</div>

					<div class="space-y-1.5">
						<span class="text-[10px] uppercase font-bold text-slate-450 block">Buch aus Klasse</span
						>
						<select
							disabled={!labelStore.selectedClass}
							onchange={(e) => {
								const bookId = /** @type {any} */ (e.target).value;
								const book = labelStore.classBooks.find(
									(/** @type {any} */ b) => String(b.id) === bookId
								);
								if (book) {
									labelStore.selectBookTitle({
										id: String(book.id),
										titel: book.title,
										autor: book.author
									});
								}
							}}
							class="w-full bg-slate-50 border border-slate-200 disabled:opacity-50 disabled:cursor-not-allowed rounded-xl px-3 py-2 text-xs text-slate-700 focus:outline-none"
						>
							<option value="">-- Buch wählen --</option>
							{#each labelStore.classBooks as book, _i (_i)}
								<option value={String(book.id)}>{book.title}</option>
							{/each}
						</select>
					</div>
				</div>
			</div>
		</div>

		<!-- Step 2: Barcodes & Mode -->
		{#if labelStore.selectedTitle}
			<div class="py-5 space-y-4 border-b border-gray-200">
				<h3 class="text-[10px] uppercase tracking-wider text-blue-600 font-bold">
					2. Barcodes generieren
				</h3>

				<!-- Selection mode -->
				<div class="flex bg-slate-100 p-0.5 rounded-lg border border-slate-200/40 text-xs">
					<button
						onclick={() => (labelStore.generationMode = 'existing')}
						class="flex-1 text-center py-1 rounded-md font-bold transition-all cursor-pointer {labelStore.generationMode ===
						'existing'
							? 'bg-white text-slate-800 shadow-xs'
							: 'text-slate-500 hover:text-slate-700'}">Vorhandene Exemplare</button
					>
					<button
						onclick={() => (labelStore.generationMode = 'new')}
						class="flex-1 text-center py-1 rounded-md font-bold transition-all cursor-pointer {labelStore.generationMode ===
						'new'
							? 'bg-white text-slate-800 shadow-xs'
							: 'text-slate-500 hover:text-slate-700'}">Neue Barcodes</button
					>
				</div>

				{#if labelStore.generationMode === 'existing'}
					<div class="space-y-2">
						<span class="text-[10px] uppercase font-bold text-slate-450 block"
							>Exemplare auswählen ({labelStore.existingCopies.length} gefunden)</span
						>
						{#if labelStore.loadingCopies}
							<div class="flex items-center justify-center py-4">
								<div
									class="w-5 h-5 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"
								></div>
							</div>
						{:else if labelStore.existingCopies.length === 0}
							<p class="text-[11px] text-slate-450">
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
										<span class="text-[10px] text-slate-450 font-sans"
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
							<span class="text-[10px] uppercase font-bold text-slate-450 block">Menge</span>
							<input
								type="number"
								min="1"
								max="100"
								bind:value={labelStore.newQuantity}
								class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-550"
							/>
						</div>
						<div class="space-y-1.5">
							<span class="text-[10px] uppercase font-bold text-slate-450 block"
								>Start-Ziffer (B-)</span
							>
							<input
								type="number"
								min="1"
								bind:value={labelStore.newStartNum}
								class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-550"
							/>
						</div>
					</div>
				{/if}
			</div>
		{/if}
	{/if}

	<!-- Step 3: Print Layout settings -->
	<div class="py-5 space-y-4 border-b border-gray-200">
		<h3 class="text-[10px] uppercase tracking-wider text-blue-600 font-bold">3. Layout-Optionen</h3>

		<div class="space-y-3.5">
			<div class="space-y-1.5">
				<span class="text-[10px] uppercase font-bold text-slate-450 block">Etikettenformat</span>
				<select
					bind:value={labelStore.formatId}
					class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-550"
				>
					<option value="zweckform_l4760">Zweckform L4760 (3x7, 21 Etiketten)</option>
					<option value="avery_3475">Avery 3475 (3x8, 24 Etiketten)</option>
					<option value="standard_52">Kleine Barcodes (4x13, 52 Etiketten)</option>
				</select>
			</div>

			<div class="space-y-1.5">
				<span class="text-[10px] uppercase font-bold text-slate-450 block"
					>Startposition auf dem A4-Bogen</span
				>
				<div class="flex items-center gap-2">
					<input
						type="number"
						min="1"
						max={labelStore.maxPositions}
						bind:value={labelStore.startPosition}
						class="w-24 bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-550"
					/>
					<span class="text-[10px] text-slate-400">max. {labelStore.maxPositions}</span>
				</div>
				<p class="text-[10px] text-slate-400 mt-1">
					Für angebrochene Bögen: Gibt an, auf welchem Feld der Druck starten soll.
				</p>
			</div>

			<div class="space-y-1.5">
				<span class="text-[10px] uppercase font-bold text-slate-450 block">Barcode-Ausgabe</span>
				<select
					bind:value={labelStore.barcodeType}
					class="w-full bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-xs text-slate-700 focus:outline-none"
				>
					<option value="code39">Code39 (1D Standard)</option>
					<option value="qr">QR-Code (2D)</option>
				</select>
			</div>

			<label class="flex items-center space-x-3 text-xs text-slate-705 cursor-pointer select-none">
				<input
					type="checkbox"
					bind:checked={labelStore.labelBorder}
					class="accent-blue-600 w-4 h-4 rounded border-slate-200 bg-white"
				/>
				<span>Hilfsrahmen / Begrenzungslinien auf Etikett zeichnen</span>
			</label>
		</div>
	</div>
</div>
