<script>
	import { apiFetch } from './apiFetch.js';
	import { loeschenBestaetigen } from './stores/bestaetigung.svelte.js';
	import { showToast } from '../inventur/lib/store.svelte.js';
	import { Trash } from '@lucide/svelte';

	/** @type {{ vormerkungen: any[] }} */
	let { vormerkungen = $bindable() } = $props();

	async function deleteVormerkung(id) {
		if (!(await loeschenBestaetigen('Vormerkung löschen?'))) return;
		try {
			const res = await apiFetch(`/api/vormerkungen/${id}`, { method: 'DELETE' });
			if (res.ok) {
				vormerkungen = vormerkungen.filter((v) => v.id !== id);
				showToast('Vormerkung gelöscht', 'success');
			} else {
				const err = await res.json().catch(() => ({}));
				showToast(err.error || 'Fehler beim Löschen', 'error');
			}
		} catch {
			showToast('Netzwerkfehler', 'error');
		}
	}
</script>

<div class="w-full h-full pt-2">
	<div class="flex items-center justify-between pb-3 border-b border-outline-variant mb-6">
		<h3 class="text-base font-medium text-on-surface-variant">
			Vorgemerkte Bücher ({vormerkungen?.length || 0})
		</h3>
	</div>

	{#if !vormerkungen || vormerkungen.length === 0}
		<div class="py-16 text-center text-base text-on-surface-variant">
			Aktuell keine Bücher vorgemerkt.
		</div>
	{:else}
		<div class="space-y-4">
			{#each vormerkungen as v, _i (_i)}
				<div class="border-b border-outline-variant py-4 flex items-start justify-between">
					<div class="flex flex-col gap-1">
						<h4 class="font-bold text-on-surface">{v.titel_name || 'Unbekannter Titel'}</h4>
						<div class="flex items-center gap-2 text-xs font-semibold text-on-surface-variant">
							<!-- Ein Span, Text je Status: 'abholbereit' zeigt die Abholfrist,
							     alles andere ('wartend' und Unbekanntes) die Wartezeit. Das Status-Konsistenz-Gate erzwingt, dass jeder von
							     der DB erlaubte Nicht-Default-Status hier vorkommt. -->
							<span class="px-2 py-0.5 rounded-md bg-primary-container text-on-primary-container">
								{v.status === 'abholbereit'
									? `Abholbereit${v.bereitgestellt_bis ? ' bis: ' + new Date(v.bereitgestellt_bis).toLocaleDateString('de-DE') : ''}`
									: `Wartet seit: ${new Date(v.erstellt_am).toLocaleDateString('de-DE')}`}
							</span>
						</div>
						{#if v.notiz}
							<p class="text-sm text-on-surface-variant mt-1 italic">Notiz: {v.notiz}</p>
						{/if}
					</div>
					<button
						onclick={() => deleteVormerkung(v.id)}
						class="icon-btn text-error"
						title="Vormerkung löschen"
						aria-label="Vormerkung löschen"
					>
						<Trash class="w-5 h-5" aria-hidden="true" />
					</button>
				</div>
			{/each}
		</div>
	{/if}
</div>
