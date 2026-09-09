<script>
	import { apiFetch, apiClient } from './apiFetch.js';
	import Tabelle from './components/ui/Tabelle.svelte';
	import { loeschenBestaetigen } from './stores/bestaetigung.svelte.js';
	import { showToast } from '../inventur/lib/store.svelte.js';
	import { authStore } from './stores/authStore.svelte.js';
	import { schuelerRechte } from './schuelerRechte.js';
	import { sucheSchuelerFuerVormerkung } from './vormerkungSchuelersuche.js';
	import Button from './components/ui/Button.svelte';
	import Feld from './components/ui/Feld.svelte';
	import { Clock, Trash2 } from '@lucide/svelte';

	/** @type {{ vormerkungen: any[], book: any }} */
	let { vormerkungen = $bindable(), book } = $props();

	let isAdding = $state(false);
	let searchVal = $state('');
	let searchResults = $state.raw(/** @type {any[]} */ ([]));
	let isSearching = $state(false);
	let notiz = $state('');
	// Die Schülersuche läuft über die Schülerdatei (GET /api/schueler?q=), also gilt deren
	// Recht — dasselbe, das der Server an der Route verlangt (schuelerRechte.js).
	const rechte = $derived(schuelerRechte(authStore.currentUser));

	async function deleteVormerkung(id) {
		if (!(await loeschenBestaetigen('Vormerkung löschen?'))) return;
		try {
			const res = await apiFetch(`/api/vormerkungen/${id}`, { method: 'DELETE' });
			if (res.ok) {
				vormerkungen = vormerkungen.filter((v) => v.id !== id);
				showToast('Vormerkung gelöscht', 'success');
			} else {
				showToast((await res.json().catch(() => ({}))).error || 'Fehler beim Löschen', 'error');
			}
		} catch {
			showToast('Netzwerkfehler', 'error');
		}
	}

	async function searchStudent() {
		if (!searchVal.trim()) {
			searchResults = [];
			return;
		}
		isSearching = true;
		try {
			searchResults = await sucheSchuelerFuerVormerkung(searchVal);
		} catch {
			searchResults = [];
		} finally {
			isSearching = false;
		}
	}

	async function addVormerkung(studentId) {
		try {
			const res = await apiClient.post('/api/vormerkungen', {
				titel_id: book.id,
				schueler_id: studentId,
				notiz
			});
			if (res.ok) {
				showToast('Erfolgreich vorgemerkt', 'success');
				isAdding = false;
				searchVal = '';
				searchResults = [];
				notiz = '';
				// Reload list
				const listRes = await apiFetch(`/api/vormerkungen?titel_id=${book.id}`);
				// Angelegt ist angelegt — schweigt die Liste, legt jemand sie ein zweites Mal an.
				if (listRes.ok) vormerkungen = await listRes.json();
				else showToast('Vormerkung angelegt — die Liste konnte nicht neu geladen werden.', 'error');
			} else {
				const err = await res.json().catch(() => ({}));
				showToast(err.error || 'Fehler beim Hinzufügen', 'error');
			}
		} catch {
			showToast('Netzwerkfehler', 'error');
		}
	}
</script>

<div class="space-y-6 pt-4">
	<div class="flex items-center justify-between">
		<h3 class="text-lg font-bold text-slate-800">Warteliste / Vormerkungen</h3>
		<Button onclick={() => (isAdding = !isAdding)}>
			{isAdding ? 'Abbrechen' : '+ Schüler vormerken'}
		</Button>
	</div>

	{#if isAdding}
		<div class="p-5 bg-slate-50 border border-slate-200 rounded-2xl space-y-4 animate-fade-in">
			{#if !rechte.einsehen}
				<!-- Sichtbar statt still: Ein fehlendes Suchfeld sähe wie ein Fehler aus. -->
				<p class="text-sm text-slate-500">
					Die Schülersuche braucht das Recht „Schülerdatei anzeigen“ (view_students).
				</p>
			{:else}
				<!-- Flex statt Raster: Die Suche trägt neben sich einen Knopf, die Notiz nicht. -->
				<div class="flex flex-col sm:flex-row items-end gap-4">
					<div class="flex flex-1 w-full items-end gap-2">
						<Feld
							id="student-search-input"
							label="Schüler suchen (Name oder Barcode)"
							bind:value={searchVal}
							onkeydown={(e) => e.key === 'Enter' && searchStudent()}
							placeholder="z.B. Max Mustermann"
							class="flex-1"
						/>
						<Button variant="secondary" onclick={searchStudent} disabled={isSearching}>
							{isSearching ? '...' : 'Suchen'}
						</Button>
					</div>
					<Feld
						id="notiz-input"
						label="Interne Notiz (optional)"
						bind:value={notiz}
						placeholder="z.B. Braucht es dringend für Referat"
						class="flex-1 w-full"
					/>
				</div>

				{#if searchResults.length > 0}
					<div class="mt-4 border border-slate-200 rounded-xl overflow-hidden bg-white">
						{#each searchResults as r, _i (_i)}
							<div
								class="flex items-center justify-between p-3 border-b border-slate-100 last:border-0 hover:bg-slate-50 transition-colors"
							>
								<div>
									<p class="font-semibold text-slate-800 text-sm">{r.title}</p>
									<p class="text-xs text-slate-500">{r.subtitle}</p>
								</div>
								<Button
									variant="secondary"
									size="sm"
									onclick={() => addVormerkung(r.id)}
									class="border-blue-100 bg-blue-50 text-blue-700 hover:bg-blue-100"
								>
									Auswählen
								</Button>
							</div>
						{/each}
					</div>
				{:else if searchVal && !isSearching && searchResults.length === 0}
					<p class="text-sm text-slate-500 mt-2">Keine Schüler gefunden.</p>
				{/if}
			{/if}
		</div>
	{/if}

	{#if vormerkungen.length === 0}
		<div
			class="py-12 flex flex-col items-center text-slate-400 gap-3 border-2 border-dashed border-slate-200 rounded-2xl bg-slate-50/50"
		>
			<Clock class="w-10 h-10" aria-hidden="true" />
			<p class="font-medium text-sm">Keine ausstehenden Vormerkungen für diesen Titel.</p>
		</div>
	{:else}
		<Tabelle beschriftung="Vormerkungen" class="whitespace-nowrap">
			<thead>
				<tr>
					<th>Wartet seit</th>
					<th>Schüler</th>
					<th>Notiz</th>
					<th class="text-right">Aktion</th>
				</tr>
			</thead>
			<tbody>
				{#each vormerkungen as v, _i (_i)}
					<tr>
						<td class="font-medium">
							{new Date(v.erstellt_am).toLocaleDateString('de-DE', {
								day: '2-digit',
								month: '2-digit',
								year: 'numeric'
							})}
						</td>
						<td class="font-semibold text-blue-600">
							{v.schueler_name || 'Unbekannt'}
						</td>
						<td>
							{v.notiz || '—'}
						</td>
						<td class="text-right">
							<button
								onclick={() => deleteVormerkung(v.id)}
								class="text-rose-600 hover:text-rose-700 font-semibold p-2 hover:bg-rose-50 rounded-lg transition-colors cursor-pointer"
								title="Vormerkung löschen"
								aria-label="Vormerkung löschen"
							>
								<Trash2 class="w-4 h-4" aria-hidden="true" />
							</button>
						</td>
					</tr>
				{/each}
			</tbody>
		</Tabelle>
	{/if}
</div>
