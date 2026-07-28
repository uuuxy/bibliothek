<script>
	import { apiFetch, apiClient } from '../../../../lib/apiFetch.js';
	import { onMount } from 'svelte';
	import Button from '../../../../lib/components/ui/Button.svelte';

	/**
	 * Klassenlehrer-Mapping — in sich geschlossener Einstellungs-Block.
	 * Verwaltet eigenes Laden/Anlegen/Löschen; meldet Erfolge/Fehler über showToast.
	 * @type {{ showToast: (msg: string, type?: string) => void }}
	 */
	let { showToast } = $props();

	let mappingRows = $state(
		/** @type {{klasse:string, lehrer_email:string, erstellt_am?:string}[]} */ ([])
	);
	let mappingLoading = $state(false);
	let newMappingKlasse = $state('');
	let newMappingEmail = $state('');
	let mappingSaving = $state(false);

	async function fetchMapping() {
		mappingLoading = true;
		try {
			const res = await apiClient.get('/api/klassen-mapping');
			if (res.ok) mappingRows = await res.json();
		} catch {
			/* ignore */
		} finally {
			mappingLoading = false;
		}
	}

	async function upsertMapping() {
		if (!newMappingKlasse.trim() || !newMappingEmail.trim()) return;
		mappingSaving = true;
		try {
			const res = await apiClient.post('/api/klassen-mapping', {
				klasse: newMappingKlasse.trim(),
				lehrer_email: newMappingEmail.trim()
			});
			if (res.ok) {
				newMappingKlasse = '';
				newMappingEmail = '';
				await fetchMapping();
				showToast('Mapping gespeichert.');
			} else {
				showToast((await res.text()) || 'Fehler beim Speichern', 'error');
			}
		} catch {
			showToast('Netzwerkfehler', 'error');
		} finally {
			mappingSaving = false;
		}
	}

	/**
	 * @param {string} klasse
	 */
	async function deleteMapping(klasse) {
		try {
			const res = await apiFetch(`/api/klassen-mapping/${encodeURIComponent(klasse)}`, {
				method: 'DELETE'
			});
			if (res.ok || res.status === 204) {
				await fetchMapping();
				showToast(`Mapping für ${klasse} gelöscht.`);
			} else {
				showToast('Fehler beim Löschen', 'error');
			}
		} catch {
			showToast('Netzwerkfehler', 'error');
		}
	}

	onMount(fetchMapping);
</script>

<div class="py-6 border-b border-slate-200 space-y-4">
	<div>
		<h3 class="text-lg font-semibold text-slate-900">Klassenlehrer-Mapping</h3>
		<p class="text-sm text-slate-500 mt-0.5">
			Weist jeder Klasse die E-Mail-Adresse der Klassenlehrerin / des Klassenlehrers zu. Diese
			Adresse wird im Mahnwesen vorausgefüllt.
		</p>
	</div>

	{#if mappingLoading}
		<p class="text-sm text-slate-400">Lade Mapping…</p>
	{:else if mappingRows.length === 0}
		<p class="text-sm text-slate-400 italic">Noch keine Einträge vorhanden.</p>
	{:else}
		<div class="overflow-hidden rounded-xl border border-slate-200">
			<table class="w-full text-sm">
				<thead class="bg-slate-50 text-slate-500">
					<tr>
						<th class="text-left px-4 py-2.5 font-medium text-xs">Klasse</th>
						<th class="text-left px-4 py-2.5 font-medium text-xs">E-Mail Klassenlehrer</th>
						<th class="px-4 py-2.5 w-12"></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each mappingRows as row (row.klasse)}
						<tr class="hover:bg-slate-50 transition-colors">
							<td class="px-4 py-2.5 font-semibold text-slate-700">{row.klasse}</td>
							<td class="px-4 py-2.5 text-slate-600">{row.lehrer_email}</td>
							<td class="px-4 py-2.5 text-right">
								<button
									onclick={() => deleteMapping(row.klasse)}
									class="p-1.5 rounded-lg text-rose-400 hover:bg-rose-50 hover:text-rose-600 transition-colors"
									title="Löschen"
								>
									<svg
										xmlns="http://www.w3.org/2000/svg"
										class="h-4 w-4"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="2"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
										/>
									</svg>
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<!-- Add new mapping -->
	<div class="flex gap-3 items-end pt-2 border-t border-slate-100">
		<div class="w-28">
			<label for="new-mapping-klasse" class="block text-xs font-medium text-slate-400 mb-1"
				>Klasse</label
			>
			<input
				id="new-mapping-klasse"
				type="text"
				bind:value={newMappingKlasse}
				placeholder="z. B. 8b"
				class="w-full px-3 py-2 rounded-lg border border-slate-300 text-sm focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none"
			/>
		</div>
		<div class="flex-1">
			<label for="new-mapping-email" class="block text-xs font-medium text-slate-400 mb-1"
				>E-Mail</label
			>
			<input
				id="new-mapping-email"
				type="email"
				bind:value={newMappingEmail}
				placeholder="klassenlehrer@schule.de"
				class="w-full px-3 py-2 rounded-lg border border-slate-300 text-sm focus:border-blue-400 focus:ring-2 focus:ring-blue-100 focus:outline-none"
			/>
		</div>
		<Button
			onclick={upsertMapping}
			disabled={mappingSaving || !newMappingKlasse.trim() || !newMappingEmail.trim()}
			class="shrink-0"
		>
			{#if mappingSaving}
				<div
					class="w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin"
				></div>
			{:else}
				Speichern
			{/if}
		</Button>
	</div>
</div>
