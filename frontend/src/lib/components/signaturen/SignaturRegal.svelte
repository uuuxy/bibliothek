<script>
	import { apiGet } from '../../apiFetch.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import { appState } from '../../../inventur/lib/store.svelte.js';

	/** @type {{ signatur: string }} */
	let { signatur } = $props();

	let buecher = $state(/** @type {any[]} */ ([]));
	let gekappt = $state(false);
	let laedt = $state(false);

	// Neu laden, sobald eine andere Signatur gewählt wird. Ohne das Zurücksetzen von
	// buecher/gekappt bliebe beim Wechsel kurz das Regal der vorigen Signatur stehen.
	$effect(() => {
		const gewaehlt = signatur;
		if (!gewaehlt) {
			buecher = [];
			gekappt = false;
			return;
		}
		let abgebrochen = false;
		laedt = true;
		buecher = [];
		gekappt = false;
		apiGet(`/api/signaturen/buecher?signatur=${encodeURIComponent(gewaehlt)}`)
			.then((daten) => {
				if (abgebrochen) return;
				buecher = daten?.buecher ?? [];
				gekappt = daten?.gekappt ?? false;
			})
			.catch(() => {
				// apiGet hat die Servermeldung bereits als Toast gezeigt.
			})
			.finally(() => {
				if (!abgebrochen) laedt = false;
			});
		return () => {
			abgebrochen = true;
		};
	});

	// Gleicher Weg wie im Medienkatalog: über appState.activeBookId, damit der
	// Deep-Link /katalog/buch/{id} und der Zurück-Knopf funktionieren.
	/** @param {string} titelId */
	function oeffneBuch(titelId) {
		appState.activeBookId = titelId;
		uiStore.activeTab = 'book_detail';
	}
</script>

{#if !signatur}
	<div class="text-sm text-slate-500 p-6 text-center">
		Wähle links eine Signatur, um das Regal zu sehen.
	</div>
{:else if laedt}
	<div class="text-sm text-slate-500 p-6 text-center">Wird geladen …</div>
{:else if buecher.length === 0}
	<div class="text-sm text-slate-500 p-6 text-center">
		Unter „{signatur}“ steht kein Buch.
	</div>
{:else}
	<div class="space-y-3">
		<div class="flex items-baseline justify-between gap-3 flex-wrap">
			<h2 class="font-bold text-slate-900">
				{signatur}
				<span class="font-normal text-slate-500 text-sm">· {buecher.length} Titel</span>
			</h2>
			<p class="text-xs text-slate-500">In Regalreihenfolge — so läufst du das Regal ab.</p>
		</div>

		{#if gekappt}
			<p
				class="text-xs bg-amber-50 border border-amber-200 text-amber-800 rounded-lg px-3 py-2"
			>
				Es werden nur die ersten {buecher.length} Titel angezeigt. Grenze die Signatur weiter ein,
				um den Rest zu sehen.
			</p>
		{/if}

		<div class="overflow-x-auto">
			<table class="w-full text-sm">
				<thead>
					<tr class="text-left text-xs uppercase tracking-wide text-slate-500">
						<th class="py-2 pr-3 font-medium">Signatur</th>
						<th class="py-2 pr-3 font-medium">Titel</th>
						<th class="py-2 pr-3 font-medium">Autor</th>
						<th class="py-2 pr-3 font-medium text-right">Exemplare</th>
						<th class="py-2 font-medium text-right">verliehen</th>
					</tr>
				</thead>
				<tbody>
					{#each buecher as buch (buch.titel_id)}
						<tr
							class="border-t border-slate-100 hover:bg-slate-50 cursor-pointer"
							onclick={() => oeffneBuch(buch.titel_id)}
						>
							<td class="py-2 pr-3 font-mono whitespace-nowrap text-slate-900"
								>{buch.signatur}</td
							>
							<td class="py-2 pr-3 text-slate-900">{buch.titel}</td>
							<td class="py-2 pr-3 text-slate-600">{buch.autor || '—'}</td>
							<td class="py-2 pr-3 text-right text-slate-700">{buch.exemplare}</td>
							<td class="py-2 text-right text-slate-700">{buch.verliehen}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</div>
{/if}
