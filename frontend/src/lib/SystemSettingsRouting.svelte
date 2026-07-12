<script>
	import { apiGet, apiPost, apiDelete } from './apiFetch.js';
	import { onMount } from 'svelte';
	import { toastStore } from './stores/toastStore.svelte.js';
	import SettingField from './components/settings/SettingField.svelte';

	/** @type {{klasse: string, lehrer_email: string}[]} */
	let mappingRows = $state([]);
	let mappingLoading = $state(false);
	let newMappingKlasse = $state('');
	let newMappingEmail = $state('');
	let mappingSaving = $state(false);

	async function fetchMapping() {
		mappingLoading = true;
		try {
			mappingRows = (await apiGet('/api/klassen-mapping')) || [];
		} catch {
			/* ignore */
		} finally {
			mappingLoading = false;
		}
	}

	onMount(async () => {
		await fetchMapping();
	});

	async function upsertMapping() {
		if (!newMappingKlasse.trim() || !newMappingEmail.trim()) return;
		mappingSaving = true;
		try {
			await apiPost('/api/klassen-mapping', {
				klasse: newMappingKlasse.trim(),
				lehrer_email: newMappingEmail.trim()
			});
			newMappingKlasse = '';
			newMappingEmail = '';
			await fetchMapping();
			toastStore.addToast('Mapping gespeichert.', 'success');
		} catch {
			// Toast already shown by apiPost
		} finally {
			mappingSaving = false;
		}
	}

	/** @param {string} klasse */
	async function deleteMapping(klasse) {
		try {
			await apiDelete(`/api/klassen-mapping/${encodeURIComponent(klasse)}`);
			await fetchMapping();
			toastStore.addToast(`Mapping für ${klasse} gelöscht.`, 'success');
		} catch {
			// Toast already shown by apiDelete
		}
	}
</script>

<!-- Flach & edge-to-edge: keine umschließende Box, flaches Listen-Layout (divide-y) -->
<div class="max-w-3xl space-y-8">
	<div>
		<h3 class="text-base font-bold text-slate-900">E-Mail Routing für Mahnungen</h3>
		<p class="text-xs text-slate-500 mt-1 leading-relaxed max-w-2xl">
			Ordnet jeder Klasse eine E-Mail-Adresse zu. Wird im Mahnwesen als Empfänger für
			Benachrichtigungen vorausgefüllt.
		</p>
	</div>

	{#if mappingLoading}
		<div class="py-8 flex justify-center">
			<div
				class="w-8 h-8 border-4 border-slate-400 border-t-transparent rounded-full animate-spin"
			></div>
		</div>
	{:else if mappingRows.length === 0}
		<p class="text-sm text-slate-500 py-4">Noch keine Mappings vorhanden.</p>
	{:else}
		<table class="w-full text-sm border-b border-slate-200">
			<thead>
				<tr
					class="border-b border-slate-200 text-xs font-bold text-slate-500 uppercase tracking-wider"
				>
					<th class="text-left py-3">Klasse</th>
					<th class="text-left py-3">Lehrer-E-Mail</th>
					<th class="py-3 text-right">Aktion</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-slate-200">
				{#each mappingRows as row, _i (_i)}
					<tr class="hover:bg-slate-50/60 transition-colors">
						<td class="py-3 font-semibold text-slate-800">{row.klasse}</td>
						<td class="py-3 text-slate-600">{row.lehrer_email}</td>
						<td class="py-3 text-right">
							<button
								onclick={() => deleteMapping(row.klasse)}
								class="p-2 text-slate-400 hover:text-rose-600 rounded-lg transition-colors cursor-pointer"
								title="Mapping löschen"
							>
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"
									><path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
									></path></svg
								>
							</button>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}

	<!-- Neuen Eintrag hinzufügen: flacher Eingabeblock ohne Box -->
	<div class="flex flex-col md:flex-row items-end gap-4">
		<div class="w-full md:w-32">
			<SettingField
				bind:value={newMappingKlasse}
				label="Klasse"
				type="text"
				placeholder="z.B. 7a"
			/>
		</div>
		<div class="flex-1 w-full">
			<SettingField
				bind:value={newMappingEmail}
				label="E-Mail"
				type="email"
				placeholder="lehrkraft@schule.de"
			/>
		</div>
		<button
			onclick={upsertMapping}
			disabled={mappingSaving || !newMappingKlasse.trim() || !newMappingEmail.trim()}
			class="w-full md:w-auto px-6 py-2.5 bg-slate-900 hover:bg-slate-800 text-white font-bold text-sm rounded-full transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed whitespace-nowrap shadow-sm"
		>
			{mappingSaving ? 'Lädt…' : 'Hinzufügen'}
		</button>
	</div>
</div>
