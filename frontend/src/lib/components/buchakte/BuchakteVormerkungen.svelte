<script>
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import Tabelle from '../ui/Tabelle.svelte';
	import { loeschenBestaetigen } from '../../stores/bestaetigung.svelte.js';
	import { melde } from '../../../inventur/lib/store.svelte.js';
	import { formatDatum } from '../../utils/format.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import BuchakteVormerken from './BuchakteVormerken.svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { Clock, Trash } from '@lucide/svelte';

	/** @type {{ vormerkungen: any[], titel: any }} */
	let { vormerkungen = $bindable(), titel } = $props();

	// Vormerkungen, deren Löschen läuft. Ein zweiter Auftrag vor der Antwort schickte eine zweite
	// Anfrage, und deren „Vormerkung nicht gefunden" stand nach dem Erfolg der ersten da.
	const loescht = new SvelteSet();

	/** @param {string} id */
	async function loescheVormerkung(id) {
		if (loescht.has(id)) return;
		if (!(await loeschenBestaetigen('Vormerkung löschen?'))) return;
		loescht.add(id);
		try {
			const res = await apiFetch(`/api/vormerkungen/${id}`, { method: 'DELETE' });
			if (res.ok) {
				vormerkungen = vormerkungen.filter((v) => v.id !== id);
				melde('Vormerkung gelöscht', 'success');
			} else {
				melde(await leseApiFehler(res, 'Fehler beim Löschen'), 'error');
			}
		} catch {
			melde('Netzwerkfehler', 'error');
		} finally {
			loescht.delete(id);
		}
	}
</script>

<!-- Ohne eigene Überschrift: Der Reiter nennt den Inhalt, wie bei den anderen Reitern der Akte. -->
<div class="space-y-6 pt-4">
	<BuchakteVormerken bind:vormerkungen {titel} />

	{#if vormerkungen.length === 0}
		<div
			class="py-12 flex flex-col items-center text-on-surface-variant gap-3 border-2 border-dashed border-outline-variant rounded-xl bg-surface/50"
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
						<td class="font-medium">{formatDatum(v.erstellt_am)}</td>
						<td class="font-semibold text-primary">
							{v.schueler_name || 'Unbekannt'}
						</td>
						<td>
							{v.notiz || '—'}
						</td>
						<td class="text-right">
							<!-- Gesperrt mit Ladekreis, solange die Anfrage läuft, wie in der Leserakte. -->
							<button
								onclick={() => loescheVormerkung(v.id)}
								disabled={loescht.has(v.id)}
								class="icon-btn text-error disabled:cursor-not-allowed disabled:text-on-surface/38"
								data-tip="Vormerkung löschen"
								aria-label="Vormerkung löschen"
							>
								{#if loescht.has(v.id)}
									<Ladekreis size="sm" farbe="aktuell" />
								{:else}
									<Trash class="w-4 h-4" aria-hidden="true" />
								{/if}
							</button>
						</td>
					</tr>
				{/each}
			</tbody>
		</Tabelle>
	{/if}
</div>
