<script>
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
	<div class="border border-slate-100 bg-white rounded-3xl overflow-hidden shadow-xs">
		<div class="overflow-x-auto">
			<table class="w-full text-left border-collapse">
				<thead>
					<tr
						class="bg-slate-50 border-b border-slate-100 text-xs font-bold text-slate-400 uppercase tracking-wider"
					>
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
								: user.rolle === 'lehrer'
									? 'bg-emerald-50 text-emerald-700 border border-emerald-100'
									: user.rolle === 'helfer'
										? 'bg-purple-50 text-purple-700 border border-purple-100'
										: 'bg-amber-50 text-amber-700 border border-amber-100'}
						<tr class="hover:bg-slate-50/50 transition-colors">
							<td class="p-4"
								><span class="font-semibold text-slate-800">{user.vorname} {user.nachname}</span
								></td
							>
							<td class="p-4 text-slate-500 text-xs">{user.email}</td>
							<td class="p-4">
								{#if user.barcode_id}
									<span
										class="text-xs bg-slate-50 border border-slate-200/60 text-slate-600 py-0.5 px-2 rounded-md"
										>{user.barcode_id}</span
									>
								{:else}
									<span class="text-xs text-slate-400 italic">Keine</span>
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
									<span class="inline-flex items-center gap-1.5 text-xs text-emerald-600">
										<span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span> Aktiv
									</span>
								{:else}
									<span class="inline-flex items-center gap-1.5 text-xs text-slate-400">
										<span class="w-1.5 h-1.5 rounded-full bg-slate-350"></span> Inaktiv
									</span>
								{/if}
							</td>
							<td class="p-4 text-right space-x-2 shrink-0">
								<button
									onclick={() => openEditUserModal(user)}
									class="px-2.5 py-1 text-xs font-semibold text-slate-600 bg-slate-50 border border-slate-200 rounded-lg hover:bg-slate-100 hover:text-slate-800 transition-colors cursor-pointer"
								>
									Bearbeiten
								</button>
								<button
									onclick={() => openDeleteConfirm(user)}
									class="px-2.5 py-1 text-xs font-semibold text-rose-600 bg-rose-50 border border-rose-100 rounded-lg hover:bg-rose-100 transition-colors cursor-pointer"
								>
									Löschen
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</div>
{/if}
