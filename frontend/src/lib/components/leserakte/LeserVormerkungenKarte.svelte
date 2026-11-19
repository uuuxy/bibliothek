<script>
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import { loeschenBestaetigen } from '../../stores/bestaetigung.svelte.js';
	import { melde } from '../../../inventur/lib/store.svelte.js';
	import Icon from '../ui/Icon.svelte';
	import Button from '../ui/Button.svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { anzahlText, formatDatum } from '../../utils/format.js';

	/** @type {{ vormerkungen: any[] }} */
	let { vormerkungen = $bindable() } = $props();

	// Vormerkungen, deren Löschen läuft. Ein zweiter Auftrag vor der Antwort schickte eine zweite
	// Anfrage, und deren „Vormerkung nicht gefunden" stand nach dem Erfolg der ersten da.
	const loescht = new SvelteSet();

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

<div class="w-full h-full pt-2">
	<div class="flex items-center justify-between pb-3 border-b border-outline-variant mb-6">
		<h3 class="text-title-medium text-on-surface-variant">
			{anzahlText(vormerkungen?.length, 'vorgemerktes Buch', 'vorgemerkte Bücher')}
		</h3>
	</div>

	<div class="space-y-4">
		<!-- Die Akte lädt an der Theke auf ein Signal der Leitung neu: An der Stelle in der Liste
			     rückte der Fokus auf den Löschknopf der nächsten Vormerkung. -->
		{#each vormerkungen as v (v.id)}
			<div class="border-b border-outline-variant py-4 flex items-start justify-between">
				<div class="flex flex-col gap-1">
					<h4 class="text-on-surface text-title-medium">{v.titel}</h4>
					<div class="flex items-center gap-2 text-body-small-emphasized text-on-surface-variant">
						<!-- Ein Span, Text je Status: 'abholbereit' zeigt die Abholfrist,
							     alles andere ('wartend' und Unbekanntes) die Wartezeit. Das Status-Konsistenz-Gate erzwingt, dass jeder von
							     der DB erlaubte Nicht-Default-Status hier vorkommt. -->
						<span class="px-2 py-0.5 rounded-md bg-primary-container text-on-primary-container">
							{v.status === 'abholbereit'
								? `Abholbereit${v.bereitgestellt_bis ? ' bis: ' + formatDatum(v.bereitgestellt_bis) : ''}`
								: `Wartet seit: ${formatDatum(v.erstellt_am)}`}
						</span>
					</div>
					{#if v.notiz}
						<p class="text-sm text-on-surface-variant mt-1 italic">Notiz: {v.notiz}</p>
					{/if}
				</div>
				<!-- Gesperrt mit Ladekreis, solange die Anfrage läuft, wie „Verlängern" daneben. -->
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
			</div>
		{/each}
	</div>
</div>
