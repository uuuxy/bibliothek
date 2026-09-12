<!-- @component StudentBescheideCard — die Schadensersatz-Bescheide eines Schülers in
     seiner Akte (#597, Etappe 1).

     Bis zum 12.09.2026 stand der Bescheid nur in der Arbeitsliste des Mahnwesens. Wer die
     Akte öffnete, weil Eltern in der Bibliothek nachfragen („wir haben da einen Brief
     bekommen"), sah die Forderung als Gebühr — aber weder ihre Referenznummer noch die
     Frist noch, ob der Fall schon bei der Aufsicht liegt. Genau danach wird gefragt.

     Der Zustand kommt aus bescheidStatus.js, derselben Stelle wie in der Arbeitsliste.
     Nachdruck öffnet dasselbe PDF mit derselben Nummer (nie eine neue). -->
<script>
	import { bescheidStatus } from './bescheidStatus.js';
	import StatusChip from './components/ui/StatusChip.svelte';
	import Button from './components/ui/Button.svelte';
	import { Printer } from '@lucide/svelte';

	/** @type {{ bescheide: any[] }} */
	let { bescheide = [] } = $props();

	/** @param {number} n */
	const euro = (n) =>
		Number(n ?? 0).toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) +
		' €';
	/** @param {string} iso */
	const datum = (iso) => new Date(iso).toLocaleDateString('de-DE');

	/** @param {any} b */
	function nachdruck(b) {
		// Neues Fenster wie im Mahnwesen: Das PDF gehört in den Betrachter des Browsers.
		window.open(`/api/bescheide/${b.id}/pdf`, '_blank');
	}
</script>

<!-- Ohne Bescheid zeigt die Karte nichts: Der Bescheid ist der seltene Fall, und eine
     leere Karte „Schadensersatz-Bescheide" in jeder Akte läse sich wie eine offene
     Forderung. Die Entscheidung gehört hierher und nicht in die Akte — sonst muss jeder
     neue Einbauort sie noch einmal treffen. -->
{#if bescheide.length > 0}
	<div class="w-full h-full pt-2">
		<div class="flex items-center justify-between pb-3 border-b border-outline-variant mb-6">
			<h3 class="text-base font-medium text-on-surface-variant">
				Schadensersatz-Bescheide ({bescheide.length})
			</h3>
		</div>

		<div class="space-y-4">
			{#each bescheide as b (b.id)}
				{@const stand = bescheidStatus(b, datum)}
				<div class="border-b border-outline-variant py-4 flex items-start justify-between gap-4">
					<div class="flex flex-col gap-1">
						<h4 class="font-mono font-bold text-on-surface">{b.referenznummer}</h4>
						<span class="text-sm text-on-surface-variant">
							{b.anzahl_positionen}
							{b.anzahl_positionen === 1 ? 'Position' : 'Positionen'} · {euro(b.gesamtbetrag)} · vom {datum(
								b.brief_datum
							)}
						</span>
						<span class="text-sm text-on-surface-variant">Frist: {datum(b.frist_bis)}</span>
						<div class="mt-1">
							<StatusChip
								ton={stand.ton}
								icon={stand.icon}
								text={stand.text}
								tip={stand.tip}
								detail={stand.detail}
							/>
						</div>
					</div>
					<Button variant="secondary" onclick={() => nachdruck(b)}>
						<Printer class="h-4 w-4" aria-hidden="true" />
						Nachdruck
					</Button>
				</div>
			{/each}
		</div>
	</div>
{/if}
