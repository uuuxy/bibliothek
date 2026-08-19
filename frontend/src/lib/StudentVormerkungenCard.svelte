<script>
	import { apiFetch } from './apiFetch.js';
	import { showToast } from '../inventur/lib/store.svelte.js';
	import { Clock, Trash2 } from '@lucide/svelte';

	/** @type {{ vormerkungen: any[] }} */
	let { vormerkungen = $bindable() } = $props();

	async function deleteVormerkung(id) {
		if (!confirm('Vormerkung wirklich löschen?')) return;
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
	<div class="flex items-center justify-between pb-3 border-b border-slate-100 mb-6">
		<h3 class="text-base font-medium text-slate-500">
			Vorgemerkte Bücher ({vormerkungen?.length || 0})
		</h3>
	</div>

	{#if !vormerkungen || vormerkungen.length === 0}
		<div class="py-16 flex flex-col items-center justify-center text-slate-500 space-y-3">
			<Clock class="h-12 w-12 text-slate-400" aria-hidden="true" />
			<span class="text-sm font-semibold text-slate-400">Aktuell keine Bücher vorgemerkt.</span>
		</div>
	{:else}
		<div class="space-y-4">
			{#each vormerkungen as v, _i (_i)}
				<div class="border-b border-slate-200 py-4 flex items-start justify-between">
					<div class="flex flex-col gap-1">
						<h4 class="font-bold text-slate-800">{v.titel_name || 'Unbekannter Titel'}</h4>
						<div class="flex items-center gap-2 text-xs font-semibold text-slate-500">
							<!-- Ein Span, Text je Status: 'abholbereit' zeigt die Abholfrist,
							     alles andere ('wartend' und Unbekanntes) die Wartezeit. Bewusst
							     dasselbe Farb-Token wie zuvor — neue Paletten-Farben verbietet die
							     Farb-Ratsche. Das Status-Konsistenz-Gate erzwingt, dass jeder von
							     der DB erlaubte Nicht-Default-Status hier vorkommt. -->
							<span class="px-2 py-0.5 rounded-md bg-blue-50 text-blue-700">
								{v.status === 'abholbereit'
									? `Abholbereit${v.bereitgestellt_bis ? ' bis: ' + new Date(v.bereitgestellt_bis).toLocaleDateString('de-DE') : ''}`
									: `Wartet seit: ${new Date(v.erstellt_am).toLocaleDateString('de-DE')}`}
							</span>
						</div>
						{#if v.notiz}
							<p class="text-sm text-slate-600 mt-1 italic">Notiz: {v.notiz}</p>
						{/if}
					</div>
					<button
						onclick={() => deleteVormerkung(v.id)}
						class="text-rose-600 hover:text-rose-700 hover:bg-rose-50 p-2 rounded-lg transition-colors cursor-pointer"
						title="Vormerkung löschen"
					>
						<Trash2 class="w-5 h-5" aria-hidden="true" />
					</button>
				</div>
			{/each}
		</div>
	{/if}
</div>
