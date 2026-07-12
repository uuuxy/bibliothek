<script>
	/**
	 * @component PermissionsEditor
	 * Reine Darstellung des Rechte-Editors (flach, edge-to-edge). Logik/State liegen
	 * im Eltern-PermissionManager; hier nur Anzeige + Toggle-Callback.
	 *
	 * @typedef {Object} Props
	 * @property {any[]} metadata - Kategorien/Items (permissionMetadata.js).
	 * @property {Record<string, Record<string, boolean>>} permissionsState
	 * @property {Record<string, boolean>} updatingKeys
	 * @property {(role: string, key: string, currentVal: boolean) => void} onToggle
	 */

	/** @type {Props} */
	let { metadata, permissionsState, updatingKeys, onToggle } = $props();
</script>

<!-- DRY: ein Toggle-Block für Mitarbeiter & Lehrer -->
{#snippet roleToggle(item, roleLabel, roleKey)}
	{@const isUpdating = updatingKeys[`${roleKey}-${item.key}`]}
	<div class="flex items-center gap-3">
		<span class="text-xs font-bold text-slate-500 tracking-wider w-16 text-right">{roleLabel}</span>
		<button
			onclick={() => onToggle(roleKey, item.key, permissionsState[roleKey]?.[item.key] ?? false)}
			disabled={isUpdating}
			class="relative inline-flex items-center cursor-pointer group focus:outline-none"
			aria-label="{roleLabel} Rechte umschalten"
		>
			<input
				type="checkbox"
				checked={permissionsState[roleKey]?.[item.key] ?? false}
				class="sr-only peer"
				readonly
			/>
			<div
				class="w-10 h-6 bg-slate-200 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-350 after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600 peer-focus:ring-2 peer-focus:ring-blue-500/20"
			></div>
			{#if isUpdating}
				<div class="absolute inset-0 flex items-center justify-center bg-white/70 rounded-full">
					<div
						class="w-3.5 h-3.5 border-2 border-slate-500 border-t-transparent rounded-full animate-spin"
					></div>
				</div>
			{/if}
		</button>
	</div>
{/snippet}

<div class="space-y-12">
	{#each metadata as cat, _i (_i)}
		<div>
			<div class="pb-3 mb-1 border-b border-gray-200 flex items-center gap-3">
				<span class="text-xl">{cat.icon}</span>
				<h3 class="font-bold text-slate-800 text-lg tracking-tight">{cat.category}</h3>
			</div>

			<div class="divide-y divide-gray-200">
				{#each cat.items as item, _i (_i)}
					<div
						class="py-6 px-1 flex flex-col md:flex-row md:items-center justify-between gap-4 hover:bg-slate-50/30 transition-colors"
					>
						<div class="max-w-xl space-y-1">
							<span class="font-semibold text-slate-850 text-base tracking-tight">{item.label}</span
							>
							<p class="text-sm text-gray-500 leading-relaxed font-medium">{item.desc}</p>
						</div>

						<div class="flex items-center gap-8 md:gap-12 shrink-0">
							<!-- Admin (Read-only) -->
							<div class="flex items-center gap-3">
								<span class="text-xs font-bold text-slate-400 tracking-wider w-16 text-right"
									>ADMIN</span
								>
								<label class="relative inline-flex items-center opacity-60 cursor-not-allowed">
									<input type="checkbox" checked disabled class="sr-only peer" />
									<div
										class="w-10 h-6 bg-blue-100 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"
									></div>
								</label>
							</div>

							{@render roleToggle(item, 'MITARBEITER', 'mitarbeiter')}
							{@render roleToggle(item, 'LEHRER', 'lehrer')}
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/each}
</div>
