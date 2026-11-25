<script>
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import Tabelle from '../ui/Tabelle.svelte';
	import { loeschenBestaetigen } from '../../stores/bestaetigung.svelte.js';
	import { melde } from '../../../inventur/lib/store.svelte.js';
	import { vormerkungStand } from '../../utils/vormerkungStand.js';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import SuchZustand from '../ui/SuchZustand.svelte';
	import BuchakteVormerken from './BuchakteVormerken.svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import Icon from '../ui/Icon.svelte';
	import Button from '../ui/Button.svelte';

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
		<SuchZustand symbol="schedule" titel="Keine Vormerkungen" />
	{:else}
		<Tabelle beschriftung="Vormerkungen" class="whitespace-nowrap">
			<thead>
				<tr>
					<th>Stand</th>
					<th>Schüler</th>
					<th>Notiz</th>
					<th class="text-right">Aktion</th>
				</tr>
			</thead>
			<tbody>
				{#each vormerkungen as v, _i (_i)}
					<tr>
						<!-- Wie in der Leserakte: Eine abholbereite Vormerkung nennt ihre Abholfrist. -->
						<td class="font-medium">{vormerkungStand(v)}</td>
						<td class="text-primary text-body-medium-emphasized">
							{v.schueler_name || 'Unbekannt'}
						</td>
						<td>
							{v.notiz || '–'}
						</td>
						<td class="text-right">
							<!-- Gesperrt mit Ladekreis, solange die Anfrage läuft, wie in der Leserakte. -->
							<Button
								variant="symbol"
								size="sm"
								loeschen
								onclick={() => loescheVormerkung(v.id)}
								disabled={loescht.has(v.id)}
								data-tip="Vormerkung löschen"
								aria-label="Vormerkung löschen"
							>
								{#if loescht.has(v.id)}
									<Ladekreis size="sm" farbe="aktuell" />
								{:else}
									<Icon name="delete" />
								{/if}
							</Button>
						</td>
					</tr>
				{/each}
			</tbody>
		</Tabelle>
	{/if}
</div>
