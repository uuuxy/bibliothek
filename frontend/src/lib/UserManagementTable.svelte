<script>
	import { Search } from '@lucide/svelte';
	import Button from './components/ui/Button.svelte';
	import StatusChip from './components/ui/StatusChip.svelte';
	import SuchZustand from './components/ui/SuchZustand.svelte';
	import Tabelle from './components/ui/Tabelle.svelte';
	import { rollenName } from './benutzerRollen.js';

	/**
	 * @typedef {Object} Props
	 * @property {boolean} loadingUsers
	 * @property {any[]} filteredUsers
	 * @property {(user: any) => void} openEditUserModal
	 * @property {(user: any) => void} openDeleteConfirm
	 */
	/** @type {Props} */
	let { loadingUsers, filteredUsers, openEditUserModal, openDeleteConfirm } = $props();
</script>

{#if loadingUsers}
	<div class="animate-pulse p-12 text-center font-medium text-on-surface-variant">
		Lade Systembenutzer...
	</div>
{:else if filteredUsers.length === 0}
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
				{#each filteredUsers as user, _i (_i)}
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
								<span class="inline-flex items-center gap-1.5 text-success">
									<span class="h-1.5 w-1.5 rounded-full bg-success"></span> Aktiv
								</span>
							{:else if user.zugang_beantragt_am}
								<!-- Selbstanmeldung: wartet auf Freischaltung, ist also kein bewusst
								     abgeschaltetes Konto. -->
								<span
									class="inline-flex items-center gap-1.5 rounded-full bg-secondary-container px-2 py-0.5 text-xs font-medium text-on-secondary-container"
								>
									<span class="h-1.5 w-1.5 rounded-full bg-tertiary"></span> Zugang beantragt
								</span>
							{:else}
								<span class="inline-flex items-center gap-1.5 text-on-surface-variant">
									<span class="h-1.5 w-1.5 rounded-full bg-outline-variant"></span> Inaktiv
								</span>
							{/if}
						</td>
						<td class="space-x-2 text-right whitespace-nowrap">
							<Button variant="secondary" size="sm" onclick={() => openEditUserModal(user)}>
								Bearbeiten
							</Button>
							<Button variant="danger" size="sm" onclick={() => openDeleteConfirm(user)}>
								Löschen
							</Button>
						</td>
					</tr>
				{/each}
			</tbody>
		</Tabelle>
	</div>
{/if}
