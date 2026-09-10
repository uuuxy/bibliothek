<!-- @component BescheideTabelle — die Arbeitsliste der Schadensersatz-Bescheide.

     Sie steht im vierten Register des Mahnwesens, weil der Bescheid die letzte Stufe
     derselben Eskalation ist: Alle → Akut fällig → Eskaliert → Bescheide. Eine eigene
     Seite hätte dieselbe Schülerliste ein zweites Mal.

     Die Zeile trägt ihren Zustand als Chip (M3: trailing icon/metadata am Listeneintrag)
     und genau die Aktionen, die dazu passen: Nachdruck immer, Übergabe erst nach
     Fristablauf. -->
<script>
	import { bescheideStore } from '../../stores/bescheide.svelte.js';
	import Tabelle from '../ui/Tabelle.svelte';
	import StatusChip from '../ui/StatusChip.svelte';
	import Button from '../ui/Button.svelte';
	import { AlertTriangle, CheckCircle2, Printer } from '@lucide/svelte';

	/** @type {{ darfSchreiben: boolean }} */
	let { darfSchreiben } = $props();

	/** @param {number} n */
	const euro = (n) =>
		Number(n).toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) +
		' €';
	/** @param {string} iso */
	const datum = (iso) => new Date(iso).toLocaleDateString('de-DE');

	/** @param {any} b */
	function nachdruck(b) {
		// Neues Fenster statt fetch: Das PDF soll im Betrachter des Browsers aufgehen,
		// wie bei den übrigen Briefen des Mahnwesens.
		window.open(`/api/bescheide/${b.id}/pdf`, '_blank');
	}
</script>

{#if bescheideStore.liste.length === 0}
	<div class="rounded-2xl border border-outline-variant py-10 text-center">
		<p class="font-semibold text-on-surface">Keine Schadensersatz-Bescheide.</p>
		<p class="mt-1 text-sm text-on-surface-variant">
			Ein Bescheid entsteht aus der Liste der Überfälligen: einen Schüler markieren, dann
			„Schadensersatz-Bescheid".
		</p>
	</div>
{:else}
	<div class="w-full pb-6">
		<div class="w-full overflow-x-auto">
			<Tabelle beschriftung="Schadensersatz-Bescheide" class="whitespace-nowrap">
				<thead class="font-medium">
					<tr>
						<th>Referenz-Nr.</th>
						<th>Schüler/in</th>
						<th class="text-right">Betrag</th>
						<th>Frist</th>
						<th>Status</th>
						<th class="text-right">Aktionen</th>
					</tr>
				</thead>
				<tbody>
					{#each bescheideStore.liste as b (b.id)}
						<tr>
							<!-- Optik gehört in die Zelle, nicht an sie (Tabellen-Ratsche): <td> trägt
							     nur Breite, Ausrichtung, Umbruch und Ziffernform. -->
							<td><span class="font-mono text-sm">{b.referenznummer}</span></td>
							<td>
								<span class="font-semibold text-on-surface">
									{b.schueler_name || '(Angaben getilgt)'}
								</span>
								{#if b.klasse}
									<span class="text-on-surface-variant"> · {b.klasse}</span>
								{/if}
								<span class="block text-sm text-on-surface-variant">
									{b.anzahl_positionen}
									{b.anzahl_positionen === 1 ? 'Position' : 'Positionen'} · vom {datum(
										b.brief_datum
									)}
								</span>
							</td>
							<td class="text-right tabular-nums">{euro(b.gesamtbetrag)}</td>
							<td class="tabular-nums">{datum(b.frist_bis)}</td>
							<td>
								{#if b.rueckgabe_nach_uebergabe}
									<StatusChip
										ton="warten"
										icon={AlertTriangle}
										text="Rückgabe nach Übergabe"
										tip="Das Buch kam zurück, nachdem der Fall abgegeben war — die Aufsicht ist zu informieren."
									/>
								{:else if b.status === 'uebergeben'}
									<StatusChip ton="neutral" text="übergeben" detail={datum(b.uebergeben_am)} />
								{:else if b.status === 'erledigt'}
									<StatusChip ton="erfolg" icon={CheckCircle2} text="erledigt" />
								{:else if b.frist_abgelaufen}
									<StatusChip
										ton="fehler"
										icon={AlertTriangle}
										text="Frist abgelaufen"
										tip="Original und Buchungsbeleg gehen jetzt an die Aufsicht."
									/>
								{:else}
									<StatusChip ton="warten" text="offen" />
								{/if}
							</td>
							<td class="space-x-2 text-right whitespace-nowrap">
								<Button variant="secondary" onclick={() => nachdruck(b)}>
									<Printer class="h-4 w-4" aria-hidden="true" />
									Nachdruck
								</Button>
								{#if darfSchreiben && b.status === 'offen' && b.frist_abgelaufen}
									<Button onclick={() => bescheideStore.uebergebe(b.id)}>Übergeben</Button>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</Tabelle>
		</div>
	</div>
{/if}
