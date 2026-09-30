<script>
	import { apiGet, apiPost, apiDelete } from './apiFetch.js';
	import Tabelle from './components/ui/Tabelle.svelte';
	import Ladekreis from './components/ui/Ladekreis.svelte';
	import { onMount } from 'svelte';
	import { toastStore } from './stores/toastStore.svelte.js';
	import Feld from './components/ui/Feld.svelte';
	import Select from './components/ui/Select.svelte';
	import { erzeugeKlassenVorschlaege } from './components/students/klassenVorschlaege.svelte.js';
	import { Trash2 } from '@lucide/svelte';

	/** @type {{klasse: string, lehrer_email: string}[]} */
	let mappingRows = $state([]);
	let mappingLoading = $state(false);
	let newMappingKlasse = $state('');
	let newMappingEmail = $state('');
	let mappingSaving = $state(false);
	// Auswählen statt tippen (docs/OFFEN.md 5.18, 30.09.2026): Ein Tippfehler legte hier eine
	// Klasse an, die es an der Schule nicht gibt, und ihre Mahnliste ging an niemanden.
	const klassenListe = erzeugeKlassenVorschlaege();

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
		await Promise.all([fetchMapping(), klassenListe.lade()]);
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
		<p class="mt-1 max-w-2xl text-sm text-on-surface-variant">
			Von Hand eingetragen, die LUSD liefert es nicht; die Versetzung rückt jede Zuordnung eine
			Stufe hoch, außer vor Klasse 7 und vor der Oberstufe.
		</p>
	</div>

	{#if mappingLoading}
		<div class="py-8 flex justify-center">
			<Ladekreis size="lg" />
		</div>
	{:else if mappingRows.length === 0}
		<p class="text-sm text-slate-500 py-4">Noch keine Mappings vorhanden.</p>
	{:else}
		<Tabelle beschriftung="Klassenleitungen und ihre E-Mail-Adressen">
			<thead>
				<tr>
					<th>Klasse</th>
					<th>Lehrer-E-Mail</th>
					<th class="text-right">Aktion</th>
				</tr>
			</thead>
			<tbody>
				{#each mappingRows as row, _i (_i)}
					<tr>
						<td class="font-semibold">{row.klasse}</td>
						<td>{row.lehrer_email}</td>
						<td class="text-right">
							<button
								onclick={() => deleteMapping(row.klasse)}
								class="p-2 text-slate-400 hover:text-rose-600 rounded-lg transition-colors cursor-pointer"
								title="Mapping löschen"
							>
								<Trash2 class="w-5 h-5" aria-hidden="true" />
							</button>
						</td>
					</tr>
				{/each}
			</tbody>
		</Tabelle>
	{/if}

	<!-- Neuen Eintrag hinzufügen: flacher Eingabeblock ohne Box -->
	<div class="flex flex-col md:flex-row items-end gap-4">
		<div class="w-full md:w-32">
			<div class="grid gap-y-1.5">
				<label for="routing-klasse" class="text-sm font-medium text-on-surface-variant"
					>Klasse</label
				>
				<Select
					id="routing-klasse"
					bind:value={newMappingKlasse}
					options={klassenListe.liste.map((k) => ({ value: k, label: k }))}
					placeholder={klassenListe.liste.length ? 'Klasse wählen' : 'Keine Klassen'}
				/>
			</div>
		</div>
		<div class="flex-1 w-full">
			<Feld
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
