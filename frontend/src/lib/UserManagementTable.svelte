<script>
	import Button from './components/ui/Button.svelte';

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
	<div class="p-12 text-center text-slate-400 font-medium animate-pulse">
		Lade Systembenutzer...
	</div>
{:else if filteredUsers.length === 0}
	<div
		class="p-12 rounded-3xl border border-dashed border-slate-200 bg-white text-center text-slate-400"
	>
		<span class="text-2xl block mb-2">👥</span>
		Keine Systembenutzer gefunden.
	</div>
{:else}
	<!-- Rahmen JA, Schatten NEIN: `data-table` ist eines der sechs M3-Bauteile mit
	     `outline-width: 1px` und hat KEIN container-elevation-Token (material-web
	     v0.192). Hier standen beide zusammen — die Bauform, die in der Spezifikation
	     nicht vorkommt. Beim Dialog weicht umgekehrt der Rahmen: Welcher der beiden
	     Teile geht, entscheidet die Bauteilrolle, nicht der Geschmack. -->
	<div class="border border-slate-100 bg-white rounded-xl overflow-hidden">
		<div class="overflow-x-auto">
			<table class="w-full text-left border-collapse">
				<thead>
					<tr class="border-b border-slate-100 bg-slate-50 text-sm font-medium text-slate-500">
						<th class="p-4">Name</th>
						<th class="p-4">E-Mail</th>
						<th class="p-4">Barcode</th>
						<th class="p-4">Rolle</th>
						<th class="p-4">Status</th>
						<th class="p-4 text-right">Aktionen</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100 text-sm text-slate-600 font-medium">
					{#each filteredUsers as user, _i (_i)}
						{@const roleBadge =
							user.rolle === 'admin'
								? 'bg-blue-50 text-blue-700 border border-blue-100'
								: user.rolle === 'kollegium'
									? 'bg-emerald-50 text-emerald-700 border border-emerald-100'
									: user.rolle === 'helfer'
										? 'bg-purple-50 text-purple-700 border border-purple-100'
										: 'bg-amber-50 text-amber-700 border border-amber-100'}
						<tr class="hover:bg-slate-50/50 transition-colors">
							<td class="p-4"
								><span class="font-semibold text-slate-800">{user.vorname} {user.nachname}</span
								></td
							>
							<td class="p-4 text-sm text-slate-500">{user.email}</td>
							<td class="p-4">
								{#if user.barcode_id}
									<span
										class="rounded-md border border-slate-200/60 bg-slate-50 px-2 py-0.5 text-sm text-slate-600"
										>{user.barcode_id}</span
									>
								{:else}
									<span class="text-sm text-slate-400 italic">Keine</span>
								{/if}
							</td>
							<td class="p-4">
								<span
									class="inline-flex px-2 py-0.5 rounded-md font-bold text-xs uppercase tracking-wide {roleBadge}"
								>
									{user.rolle}
								</span>
							</td>
							<td class="p-4">
								{#if user.aktiv}
									<span class="inline-flex items-center gap-1.5 text-sm text-emerald-600">
										<span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span> Aktiv
									</span>
								{:else if user.zugang_beantragt_am}
									<!-- Selbstanmeldung (Migration 086): wartet auf Freischaltung — NICHT
									     dasselbe wie ein bewusst deaktiviertes Konto. -->
									<span
										class="inline-flex items-center gap-1.5 rounded-full bg-secondary-container px-2 py-0.5 text-xs font-medium text-on-secondary-container"
									>
										<span class="w-1.5 h-1.5 rounded-full bg-tertiary"></span> Zugang beantragt
									</span>
								{:else}
									<span class="inline-flex items-center gap-1.5 text-sm text-slate-400">
										<span class="w-1.5 h-1.5 rounded-full bg-slate-300"></span> Inaktiv
									</span>
								{/if}
							</td>
							<td class="p-4 text-right space-x-2 shrink-0">
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
			</table>
		</div>
	</div>
{/if}
