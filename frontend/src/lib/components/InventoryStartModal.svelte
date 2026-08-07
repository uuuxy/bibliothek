<script>
	import { slide } from 'svelte/transition';
	import Select from './ui/Select.svelte';

	const KLASSEN = [
		{ value: '', label: 'Alle Klassen' },
		...[5, 6, 7, 8, 9, 10, 11, 12, 13].map((g) => ({ value: g, label: `Klasse ${g}` }))
	];

	/**
	 * @type {{
	 *   dialogEl: HTMLDialogElement | undefined,
	 *   state: any,
	 *   onClose: () => void,
	 *   onStart: () => void
	 * }}
	 */
	let { dialogEl = $bindable(), state, onClose, onStart } = $props();
</script>

<dialog
	bind:this={dialogEl}
	class="backdrop:bg-slate-900/50 backdrop:backdrop-blur-sm bg-transparent p-0 w-full max-w-md m-auto border-0 rounded-2xl shadow-2xl overflow-hidden"
>
	<div class="bg-white w-full">
		<div class="p-6 border-b border-slate-100">
			<h2 class="text-xl font-bold text-slate-900">Inventur-Scope wählen</h2>
			<p class="text-sm text-slate-500 mt-1">Welcher Teil der Bibliothek soll geprüft werden?</p>
		</div>
		<div class="p-6 space-y-6">
			{#if state.errorMessage}
				<div class="p-3 bg-red-50 text-red-700 text-sm font-medium rounded-lg">
					{state.errorMessage}
				</div>
			{/if}

			<div class="space-y-3">
				<label
					class="flex items-start space-x-3 p-3 border rounded-xl cursor-pointer transition-colors {state.scopeType ===
					'global'
						? 'border-blue-500 bg-blue-50'
						: 'border-slate-200 hover:bg-slate-50'}"
				>
					<div class="flex items-center h-5 mt-0.5">
						<input
							type="radio"
							bind:group={state.scopeType}
							value="global"
							class="w-4 h-4 text-blue-600 border-slate-300 focus:ring-blue-600"
						/>
					</div>
					<div class="flex-1">
						<div class="font-bold text-slate-900">Komplette Bibliothek</div>
						<div class="text-xs text-slate-500">Prüfe den gesamten Bestand ab.</div>
					</div>
				</label>

				<label
					class="flex items-start space-x-3 p-3 border rounded-xl cursor-pointer transition-colors {state.scopeType ===
					'signature'
						? 'border-blue-500 bg-blue-50'
						: 'border-slate-200 hover:bg-slate-50'}"
				>
					<div class="flex items-center h-5 mt-0.5">
						<input
							type="radio"
							bind:group={state.scopeType}
							value="signature"
							class="w-4 h-4 text-blue-600 border-slate-300 focus:ring-blue-600"
						/>
					</div>
					<div class="flex-1">
						<div class="font-bold text-slate-900">Nur bestimmte Signatur</div>
						<div class="text-xs text-slate-500">Grenze die Inventur auf eine Kategorie ein.</div>
					</div>
				</label>

				<label
					class="flex items-start space-x-3 p-3 border rounded-xl cursor-pointer transition-colors {state.scopeType ===
					'filter'
						? 'border-blue-500 bg-blue-50'
						: 'border-slate-200 hover:bg-slate-50'}"
				>
					<div class="flex items-center h-5 mt-0.5">
						<input
							type="radio"
							bind:group={state.scopeType}
							value="filter"
							class="w-4 h-4 text-blue-600 border-slate-300 focus:ring-blue-600"
						/>
					</div>
					<div class="flex-1">
						<div class="font-bold text-slate-900">Nach Fach / Klasse</div>
						<div class="text-xs text-slate-500">
							Gezielte Teil-Inventur, z. B. „Mathe, Klasse 5“.
						</div>
					</div>
				</label>
			</div>

			{#if state.scopeType === 'signature'}
				<div transition:slide class="space-y-2">
					<label for="inv-signatur" class="block text-sm font-medium text-slate-700"
						>Signatur auswählen</label
					>
					<!-- Freie Eingabe MIT Vorschlagsliste: Die Signatur wird als Präfix gelesen,
					     "BIB Deu" erfasst also auch "BIB Deu 5 KRÜ". Eine reine Auswahlliste
					     könnte nur exakt vorhandene Werte anbieten und damit kein ganzes Regal. -->
					<input
						id="inv-signatur"
						list="inv-signatur-vorschlaege"
						bind:value={state.selectedSignatur}
						placeholder="z. B. BIB Deu"
						class="w-full bg-slate-50 border border-slate-200 rounded-xl px-4 py-3 text-slate-900 focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none"
					/>
					<datalist id="inv-signatur-vorschlaege">
						{#each state.signaturen as sig (sig.signatur)}
							<option value={sig.signatur}>{sig.signatur} — {sig.exemplare} Exemplare</option>
						{/each}
					</datalist>
					<p class="text-xs text-slate-500">
						Erfasst alles, was mit dieser Signatur beginnt — „BIB Deu“ also auch „BIB Deu 5 KRÜ“.
					</p>
				</div>
			{/if}

			{#if state.scopeType === 'filter'}
				<div transition:slide class="grid grid-cols-2 gap-3">
					<div class="space-y-2">
						<label for="inventur-fach" class="block text-sm font-medium text-slate-700">Fach</label>
						<Select
							id="inventur-fach"
							bind:value={state.selectedFach}
							options={[
								{ value: '', label: 'Alle Fächer' },
								...state.faecher.map((/** @type {string} */ f) => ({ value: f, label: f }))
							]}
						/>
					</div>
					<div class="space-y-2">
						<label for="inventur-klasse" class="block text-sm font-medium text-slate-700"
							>Klasse</label
						>
						<Select id="inventur-klasse" bind:value={state.selectedGrade} options={KLASSEN} />
					</div>
				</div>
			{/if}
		</div>
		<div class="p-4 bg-slate-50 border-t border-slate-100 flex justify-end space-x-3">
			<button
				onclick={onClose}
				class="px-5 py-2.5 text-sm font-semibold text-slate-600 hover:bg-slate-200 rounded-xl transition-colors"
				>Abbrechen</button
			>
			<button
				onclick={onStart}
				class="px-5 py-2.5 text-sm font-semibold text-white bg-blue-600 hover:bg-blue-700 rounded-xl shadow-sm transition-colors"
				>Inventur Starten</button
			>
		</div>
	</div>
</dialog>
