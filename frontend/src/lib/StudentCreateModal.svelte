<script>
	import Modal from './Modal.svelte';
	import { apiFetch, apiClient } from './apiFetch.js';

	let { open = false, readerGroups = [], onclose, onsuccess } = $props();

	let newVorname = $state('');
	let newNachname = $state('');
	let newKlasse = $state('');
	let customKlasseInput = $state(false);
	let newBarcode = $state('');
	let newGeburtsdatum = $state('');
	let createError = $state('');
	let duplicateConflict = $state(false);
	let isSaving = $state(false);

	// Watch for open state changes to reset form
	$effect(() => {
		if (open) {
			newVorname = '';
			newNachname = '';
			newKlasse = '';
			newBarcode = '';
			newGeburtsdatum = '';
			createError = '';
			duplicateConflict = false;
			customKlasseInput = false;
		}
	});

	async function createStudent() {
		createError = '';
		duplicateConflict = false;
		if (!newVorname.trim() || !newNachname.trim() || !newKlasse.trim()) {
			createError = 'Vorname, Nachname und Klasse sind Pflichtfelder.';
			return;
		}
		isSaving = true;
		try {
			const res = await apiClient.post('/api/schueler', {
				vorname: newVorname.trim(),
				nachname: newNachname.trim(),
				klasse: newKlasse.trim(),
				barcode_id: newBarcode.trim(),
				geburtsdatum: newGeburtsdatum.trim() || null
			});
			if (res.ok) {
				onsuccess?.();
			} else {
				if (res.status === 409) {
					duplicateConflict = true;
				} else {
					const errText = await res.text();
					try {
						const errObj = JSON.parse(errText);
						createError = errObj.error || 'Fehler beim Anlegen des Schülers.';
					} catch {
						createError = errText || 'Fehler beim Anlegen des Schülers.';
					}
				}
			}
		} catch (err) {
			createError = 'Netzwerkfehler beim Anlegen des Schülers.';
			console.error(err);
		} finally {
			isSaving = false;
		}
	}

	function handleClose() {
		onclose?.();
	}
</script>

<Modal {open} onclose={handleClose} size="md">
	{#snippet header()}
		<h3 class="text-sm font-bold text-slate-800">Neuen Schüler anlegen</h3>
	{/snippet}
		<div class="p-6 space-y-4">
			{#if duplicateConflict}
				<div
					class="p-4 bg-amber-50 border border-amber-200 rounded-xl flex items-start gap-3 text-sm font-semibold text-amber-800"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-5 w-5 text-amber-500 shrink-0 mt-0.5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2.5"
						><path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
						/></svg
					>
					<div>
						<p>
							Achtung: Ein Schüler mit diesem Namen und Geburtsdatum existiert bereits im System.
						</p>
						<p class="text-xs font-normal mt-1 opacity-80">
							Bitte überprüfe die Daten, um Duplikate zu vermeiden. Wurde der Schüler eventuell
							bereits angelegt oder importiert?
						</p>
					</div>
				</div>
			{/if}

			{#if createError}
				<div
					class="p-3 bg-rose-50 border border-rose-100 rounded-xl text-xs font-semibold text-rose-600"
				>
					{createError}
				</div>
			{/if}

			<label class="block text-xs font-bold uppercase tracking-wider text-slate-400"
				>Vorname *
				<input
					type="text"
					bind:value={newVorname}
					placeholder="z.B. Max"
					class="mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-sans"
				/>
			</label>

			<label class="block text-xs font-bold uppercase tracking-wider text-slate-400"
				>Nachname *
				<input
					type="text"
					bind:value={newNachname}
					placeholder="z.B. Mustermann"
					class="mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-sans"
				/>
			</label>

			<label class="block text-xs font-bold uppercase tracking-wider text-slate-400"
				>Geburtsdatum
				<input
					type="date"
					bind:value={newGeburtsdatum}
					class="mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-sans"
				/>
			</label>

			<label class="block text-xs font-bold uppercase tracking-wider text-slate-400"
				>Klasse *
				<div class="mt-1.5 flex gap-2">
					{#if !customKlasseInput}
						<select
							bind:value={newKlasse}
							onchange={(e) => {
								const sel = /** @type {HTMLSelectElement} */ (e.target);
								if (sel && sel.value === '__custom__') {
									customKlasseInput = true;
									newKlasse = '';
								}
							}}
							class="w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all cursor-pointer font-sans"
						>
							<option value="">-- Lesergruppe / Klasse auswählen --</option>
							{#each readerGroups as g, _i (_i)}
								<option value={g.kuerzel}>{g.kuerzel} ({g.bezeichnung})</option>
							{/each}
							<option value="__custom__">Manuell eingeben...</option>
						</select>
					{:else}
						<div class="relative w-full">
							<input
								type="text"
								bind:value={newKlasse}
								placeholder="z.B. 10b"
								class="w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-sans"
							/>
							<button
								type="button"
								onclick={() => {
									customKlasseInput = false;
									newKlasse = '';
								}}
								class="absolute right-2.5 top-1/2 -translate-y-1/2 text-xs font-semibold text-blue-600 hover:text-blue-750 transition-colors bg-transparent border-none cursor-pointer"
								>Auswahl</button
							>
						</div>
					{/if}
				</div>
			</label>

			<label class="block text-xs font-bold uppercase tracking-wider text-slate-400"
				>Barcode-ID (optional)
				<input
					type="text"
					bind:value={newBarcode}
					placeholder="Wird automatisch generiert, wenn leer"
					class="mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-slate-800 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all"
				/>
			</label>

			<div class="flex justify-end gap-3 pt-2 border-t border-slate-100">
				<button
					onclick={handleClose}
					disabled={isSaving}
					class="rounded-xl bg-slate-100 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-200 disabled:opacity-60 transition-colors cursor-pointer font-sans"
					>Abbrechen</button
				>
				<button
					onclick={createStudent}
					disabled={isSaving}
					class="rounded-xl bg-blue-600 px-4 py-2 text-sm font-bold text-white hover:bg-blue-750 disabled:opacity-60 transition-colors cursor-pointer font-sans"
				>
					{isSaving ? 'Speichern...' : 'Speichern'}
				</button>
			</div>
		</div>
</Modal>
