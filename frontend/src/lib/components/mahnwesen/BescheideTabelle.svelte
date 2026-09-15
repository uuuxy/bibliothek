<!-- @component BescheideTabelle — der Reiter „Schadensersatz": wer schuldet Geld, und
     was ist der nächste Schritt.

     Die ersten drei Reiter des Mahnwesens fragen „Wer hat Bücher zu spät?". Dieser
     fragt nach dem Geld: Sobald ein Verlust oder Schaden gemeldet ist, endet die
     Ausleihe (ReportDamage), das Kind fällt aus der Mahnliste — und stand bis zum
     15.09.2026 nirgends mehr, bis jemand zufällig den Bescheid fand. Jetzt hat die
     Stufe „Forderung offen, noch kein Brief" ihre eigene Zeile, oben, weil sie der
     erste Schritt ist; darunter die Briefe, abgelaufene Fristen zuerst (Server).

     Jede Zeile trägt ihren Zustand als Chip mit dem nächsten Schritt im Tipp und
     genau die Aktionen, die dazu passen: Bescheid erstellen, Nachdruck, Übergeben. -->
<script>
	import { bescheideStore } from '../../stores/bescheide.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import Tabelle from '../ui/Tabelle.svelte';
	import StatusChip from '../ui/StatusChip.svelte';
	import Button from '../ui/Button.svelte';
	import { bescheidStatus } from '../../bescheidStatus.js';
	import { FileText, Printer } from '@lucide/svelte';

	/** @type {{ darfSchreiben: boolean, onBescheid: (schuelerId: string) => void }} */
	let { darfSchreiben, onBescheid } = $props();

	/** @param {number} n */
	const euro = (n) =>
		Number(n).toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) +
		' €';
	/** @param {string} iso */
	const datum = (iso) => new Date(iso).toLocaleDateString('de-DE');

	/** Öffnet die Akte — dort wird bezahlt, storniert und gemeldet. @param {string} id */
	function oeffneAkte(id) {
		uiStore.requestedStudentId = id;
		uiStore.activeTab = 'students_dir';
	}

	/** @param {any} b */
	function nachdruck(b) {
		// Neues Fenster statt fetch: Das PDF soll im Betrachter des Browsers aufgehen,
		// wie bei den übrigen Briefen des Mahnwesens.
		window.open(`/api/bescheide/${b.id}/pdf`, '_blank');
	}

	const leer = $derived(
		bescheideStore.ausstehend.length === 0 && bescheideStore.zeilen.length === 0
	);
</script>

{#snippet schueler(id, name, klasse)}
	<button
		type="button"
		onclick={() => oeffneAkte(id)}
		class="font-semibold text-on-surface text-left hover:underline cursor-pointer rounded focus-visible:outline-2 focus-visible:outline-primary"
		aria-label="Akte von {name} öffnen"
	>
		{name || '(Angaben getilgt)'}
	</button>
	{#if klasse}
		<span class="text-on-surface-variant"> · {klasse}</span>
	{/if}
{/snippet}

{#if leer}
	<div class="rounded-2xl border border-outline-variant py-10 text-center">
		<p class="font-semibold text-on-surface">Keine offenen Schadensersatz-Fälle.</p>
		<p class="mt-1 text-sm text-on-surface-variant">
			Ein Fall beginnt in der Schülerakte mit „Verlust/Schaden melden" an der Ausleihzeile. Er steht
			dann hier, bis der Bescheid erstellt, die Frist abgelaufen und der Fall übergeben ist.
		</p>
	</div>
{:else}
	<div class="w-full pb-6">
		<div class="w-full overflow-x-auto">
			<Tabelle beschriftung="Schadensersatz: Forderungen und Bescheide" class="whitespace-nowrap">
				<thead class="font-medium">
					<tr>
						<th>Referenz-Nr.</th>
						<th>Schüler/in</th>
						<th class="text-right">Betrag</th>
						<th>Frist</th>
						<th>Stand</th>
						<th class="text-right">Nächster Schritt</th>
					</tr>
				</thead>
				<tbody>
					{#each bescheideStore.ausstehend as z (z.schueler_id)}
						<tr>
							<td><span class="text-sm text-on-surface-variant">noch keine</span></td>
							<td>
								{@render schueler(z.schueler_id, z.schueler_name, z.klasse)}
								<span class="block text-sm text-on-surface-variant">
									{z.anzahl}
									{z.anzahl === 1 ? 'Forderung' : 'Forderungen'} · seit {datum(z.seit)}
								</span>
							</td>
							<td class="text-right tabular-nums">{euro(z.summe)}</td>
							<td><span class="text-on-surface-variant">–</span></td>
							<td>
								{#if z.lernmittel}
									<StatusChip
										ton="warten"
										icon={FileText}
										text="Bescheid noch nicht erstellt"
										tip="Verlust oder Schaden ist gemeldet. Der Brief mit Referenznummer und Frist fehlt noch."
									/>
								{:else}
									<StatusChip
										ton="neutral"
										text="Bücherei-Buch, kein Bescheid"
										tip="Für Bestände der Schülerbücherei gibt es keinen Bescheid des Landes; bezahlt wird in der Akte."
									/>
								{/if}
							</td>
							<td class="text-right whitespace-nowrap">
								{#if darfSchreiben && z.lernmittel}
									<Button onclick={() => onBescheid(z.schueler_id)}>Bescheid erstellen</Button>
								{/if}
							</td>
						</tr>
					{/each}
					{#each bescheideStore.zeilen as b (b.id)}
						{@const stand = bescheidStatus(b, datum)}
						<tr>
							<!-- Optik gehört in die Zelle, nicht an sie (Tabellen-Ratsche): <td> trägt
							     nur Breite, Ausrichtung, Umbruch und Ziffernform. -->
							<td><span class="font-mono text-sm">{b.referenznummer}</span></td>
							<td>
								{#if b.schueler_id}
									{@render schueler(b.schueler_id, b.schueler_name, b.klasse)}
								{:else}
									<span class="font-semibold text-on-surface">(Angaben getilgt)</span>
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
								<StatusChip
									ton={stand.ton}
									icon={stand.icon}
									text={stand.text}
									tip={stand.tip}
									detail={stand.detail}
								/>
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
