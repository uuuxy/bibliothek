<script>
	import { apiGet } from '../../apiFetch.js';
	import Tabelle from '../ui/Tabelle.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
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
	// Deep-Link /medienkatalog/buch/{id} und der Zurück-Knopf funktionieren.
	/** @param {string} titelId */
	function oeffneBuch(titelId) {
		appState.activeBookId = titelId;
		uiStore.activeTab = 'book_detail';
	}
</script>

{#if !signatur}
	<div class="text-sm text-on-surface-variant p-6 text-center">
		Wähle links eine Signatur, um das Regal zu sehen.
	</div>
{:else if laedt}
	<div class="flex justify-center p-6"><Ladekreis label="Regal lädt" /></div>
{:else if buecher.length === 0}
	<div class="text-sm text-on-surface-variant p-6 text-center">
		Unter „{signatur}“ steht kein Buch.
	</div>
{:else}
	<div class="space-y-3">
		<div class="flex items-baseline justify-between gap-3 flex-wrap">
			<h2 class="font-bold text-on-surface">
				{signatur}
				<span class="font-normal text-on-surface-variant text-sm">· {buecher.length} Titel</span>
			</h2>
			<p class="text-xs text-on-surface-variant">
				In Regalreihenfolge — so läufst du das Regal ab.
			</p>
		</div>

		{#if gekappt}
			<p class="text-xs bg-warning-container text-on-warning-container rounded-lg px-3 py-2">
				Es werden nur die ersten {buecher.length} Titel angezeigt. Grenze die Signatur weiter ein, um
				den Rest zu sehen.
			</p>
		{/if}

		<div class="overflow-x-auto">
			<Tabelle beschriftung="Signaturen im Regal">
				<thead>
					<tr>
						<th>Signatur</th>
						<th>Titel</th>
						<th>Autor</th>
						<th class="text-right">Exemplare</th>
						<th class="text-right">verliehen</th>
					</tr>
				</thead>
				<tbody>
					{#each buecher as buch (buch.titel_id)}
						<!-- Der Titel öffnet die Akte, nicht die Zeile: Eine Zeile mit onclick ist nur
						     mit der Maus erreichbar. -->
						<tr>
							<td class="font-mono whitespace-nowrap">{buch.signatur}</td>
							<td>
								<button
									type="button"
									onclick={() => oeffneBuch(buch.titel_id)}
									class="cursor-pointer text-left hover:text-primary hover:underline focus-visible:outline-2 focus-visible:outline-primary"
								>
									{buch.titel}
								</button>
							</td>
							<td>{buch.autor || '—'}</td>
							<td class="text-right">{buch.exemplare}</td>
							<td class="text-right">{buch.verliehen}</td>
						</tr>
					{/each}
				</tbody>
			</Tabelle>
		</div>
	</div>
{/if}
