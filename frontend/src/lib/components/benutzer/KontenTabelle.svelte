<script>
	import { Search } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import StatusChip from '../ui/StatusChip.svelte';
	import SuchZustand from '../ui/SuchZustand.svelte';
	import Tabelle from '../ui/Tabelle.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { rollenName } from './benutzerRollen.js';

	/**
	 * @typedef {Object} Props
	 * @property {boolean} laedt
	 * @property {any[]} konten
	 * @property {string} [suchtext]
	 * @property {string} [ladefehler] der Grund, wenn die Konten nicht kamen
	 * @property {() => void} [ladeKonten]
	 * @property {(user: any) => void} oeffneKonto
	 * @property {(user: any) => void} frageLoeschen
	 */
	/** @type {Props} */
	let {
		laedt,
		konten,
		suchtext = '',
		ladefehler = '',
		ladeKonten,
		oeffneKonto,
		frageLoeschen
	} = $props();

	let gefiltert = $derived.by(() => {
		const suche = suchtext.trim().toLowerCase();
		if (!suche) return konten;
		return konten.filter(
			(u) =>
				u.vorname.toLowerCase().includes(suche) ||
				u.nachname.toLowerCase().includes(suche) ||
				u.email.toLowerCase().includes(suche) ||
				(u.barcode_id && u.barcode_id.toLowerCase().includes(suche)) ||
				u.rolle.toLowerCase().includes(suche)
		);
	});
</script>

{#if laedt}
	<div class="animate-pulse p-12 text-center font-medium text-on-surface-variant">
		Lade Systembenutzer...
	</div>
{:else if ladefehler}
	<!-- Nicht „Keine Systembenutzer gefunden": Die Konten kamen nicht. -->
	<LadeFehler titel="Benutzerkonten nicht geladen" text={ladefehler} onerneut={ladeKonten} />
{:else if gefiltert.length === 0}
	<SuchZustand symbol={Search} titel="Keine Systembenutzer gefunden" />
{:else}
	<div class="overflow-x-auto">
		<Tabelle beschriftung="Benutzerkonten">
			<thead>
				<tr>
					<th>Name</th>
					<th>E-Mail</th>
					<th>Barcode</th>
					<th>Rolle</th>
					<th>Status</th>
					<th class="w-px text-right whitespace-nowrap">Aktionen</th>
				</tr>
			</thead>
			<tbody class="font-medium">
				{#each gefiltert as user, _i (_i)}
					<tr>
						<td>
							<span class="font-semibold">{user.vorname} {user.nachname}</span>
						</td>
						<!-- Eine Adresse hat keine Leerzeichen; ohne Umbruch an beliebiger Stelle
						     schöbe eine lange die Knöpfe aus dem Fenster. -->
						<td class="min-w-32 wrap-anywhere">{user.email}</td>
						<td class="whitespace-nowrap">
							{#if user.barcode_id}
								<span class="font-mono">{user.barcode_id}</span>
							{:else}
								<span class="text-on-surface-variant italic">Keine</span>
							{/if}
						</td>
						<!-- Ein Abzeichen für jede Rolle: Grün und Gelb heißen im Programm „in Ordnung"
						     und „Achtung", eine Farbe je Rolle sagte hier nichts davon. -->
						<td><StatusChip text={rollenName(user.rolle)} /></td>
						<td class="whitespace-nowrap">
							{#if user.aktiv}
								<StatusChip ton="erfolg" text="Aktiv" />
							{:else if user.zugang_beantragt_am}
								<!-- Selbstanmeldung: wartet auf Freischaltung, ist also kein bewusst
								     abgeschaltetes Konto. -->
								<StatusChip ton="warten" text="Zugang beantragt" />
							{:else}
								<StatusChip text="Inaktiv" />
							{/if}
						</td>
						<td class="space-x-2 text-right whitespace-nowrap">
							<Button variant="secondary" size="sm" onclick={() => oeffneKonto(user)}>
								Bearbeiten
							</Button>
							<Button variant="danger" size="sm" onclick={() => frageLoeschen(user)}>
								Löschen
							</Button>
						</td>
					</tr>
				{/each}
			</tbody>
		</Tabelle>
	</div>
{/if}
