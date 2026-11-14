<script>
	import { Pencil, Search, Trash } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import StatusChip from '../ui/StatusChip.svelte';
	import SuchZustand from '../ui/SuchZustand.svelte';
	import Tabelle from '../ui/Tabelle.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { rollenName } from './benutzerRollen.js';
	import { authStore } from '../../stores/authStore.svelte.js';

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
		Lade Systembenutzer …
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
					<th>Anmeldung</th>
					<th>Ausweisnummer</th>
					<th>Rolle</th>
					<th>Status</th>
					<th class="w-px text-right whitespace-nowrap">Aktionen</th>
				</tr>
			</thead>
			<tbody class="font-medium">
				{#each gefiltert as user, _i (_i)}
					<!-- Das eigene Konto löscht niemand; der Server lehnt es ab, der Knopf sagt es vorher. -->
					{@const eigenes = user.id === authStore.currentUser?.user_id}
					<tr>
						<td>
							<span class="font-semibold">{user.vorname} {user.nachname}</span>
						</td>
						<!-- Eine Adresse hat keine Leerzeichen; ohne Umbruch an beliebiger Stelle
						     schöbe eine lange die Knöpfe aus dem Fenster. -->
						<td class="min-w-32 wrap-anywhere">{user.email}</td>
						<!-- Welcher Server das Passwort prüft; eine Eigenschaft, kein Zustand, deshalb
						     ohne Abzeichen. -->
						<td>{user.anmeldeverfahren === 'ldap' ? 'LDAP' : 'IMAP'}</td>
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
						<!-- Symbolknöpfe mit Sprechblase wie in den Zeilen von Gmail und Drive; 16 px Abstand,
						     damit sich die Trefferflächen nicht berühren. -->
						<td class="text-right whitespace-nowrap">
							<div class="inline-flex items-center gap-4">
								<Button
									variant="symbol"
									size="sm"
									onclick={() => oeffneKonto(user)}
									data-tip="Bearbeiten"
									aria-label="{user.vorname} {user.nachname} bearbeiten"
								>
									<Pencil class="h-4 w-4" aria-hidden="true" />
								</Button>
								<Button
									variant="symbol"
									size="sm"
									onclick={() => frageLoeschen(user)}
									disabled={eigenes}
									data-tip={eigenes ? 'Das eigene Konto lässt sich nicht löschen' : 'Löschen'}
									aria-label="{user.vorname} {user.nachname} löschen"
								>
									<Trash class="h-4 w-4" aria-hidden="true" />
								</Button>
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</Tabelle>
	</div>
{/if}
