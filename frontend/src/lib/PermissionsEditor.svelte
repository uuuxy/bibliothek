<script>
	import Switch from './components/ui/Switch.svelte';
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
	 * @property {boolean} [schreibgeschuetzt] - PUT /api/admin/permissions ist Admin-only; mit
	 *   manage_users sieht man die Matrix, kann sie aber nicht ändern.
	 */

	/** @type {Props} */
	let { metadata, permissionsState, updatingKeys, onToggle, schreibgeschuetzt = false } = $props();
</script>

<!-- DRY: ein Toggle-Block für Mitarbeiter, Lehrer und Helfer.
     Seit 08.09.2026 der M3-Schalter aus ui/ statt eines sr-only/peer-checked-Nachbaus
     (40×24 px, ohne zugänglichen Namen). Das {#key} setzt den Schalter nach jedem
     Speichern auf den Stand der Matrix zurück — schlägt der PUT fehl, bliebe der
     Schalter sonst umgelegt, obwohl das Recht unverändert ist. -->
{#snippet roleToggle(item, roleLabel, roleKey)}
	{@const isUpdating = updatingKeys[`${roleKey}-${item.key}`]}
	{@const wert = permissionsState[roleKey]?.[item.key] ?? false}
	<div
		class="flex items-center gap-3"
		title={schreibgeschuetzt ? 'Die Rechte-Matrix kann nur ein Administrator ändern' : undefined}
	>
		<span class="text-xs font-bold text-slate-500 tracking-wider w-24 text-right">{roleLabel}</span>
		{#key isUpdating}
			<Switch
				checked={wert}
				disabled={isUpdating || schreibgeschuetzt}
				label="{roleLabel} Rechte umschalten"
				onchange={() => onToggle(roleKey, item.key, wert)}
			/>
		{/key}
	</div>
{/snippet}

<div class="space-y-12">
	{#each metadata as cat, _i (_i)}
		<div>
			<div class="pb-3 mb-1 border-b border-slate-200 flex items-center gap-3">
				<cat.icon class="text-on-surface-variant h-5 w-5" aria-hidden="true" />
				<h3 class="font-bold text-slate-800 text-lg tracking-tight">{cat.category}</h3>
			</div>

			<div class="divide-y divide-slate-200">
				{#each cat.items as item, _i (_i)}
					<div
						class="py-6 px-1 flex flex-col md:flex-row md:items-center justify-between gap-4 hover:bg-slate-50/30 transition-colors"
					>
						<div class="max-w-xl space-y-1">
							<span class="font-semibold text-slate-800 text-base tracking-tight">{item.label}</span
							>
							<p class="text-sm text-slate-500 leading-relaxed font-medium">{item.desc}</p>
						</div>

						<div class="flex items-center gap-8 md:gap-12 shrink-0">
							<!-- Admin (Read-only) -->
							<div class="flex items-center gap-3">
								<span class="text-xs font-bold text-slate-400 tracking-wider w-24 text-right"
									>ADMIN</span
								>
								<Switch checked disabled label="Administrator hat immer alle Rechte" />
							</div>

							{@render roleToggle(item, 'MITARBEITER', 'mitarbeiter')}
							{@render roleToggle(item, 'KOLLEGIUM', 'kollegium')}
							<!-- HELFER fehlte hier: Das Backend fuehrt und liefert die Rechte dieser
							     Rolle (seed.go, user_admin_permissions.go), aendern liess sie sich
							     ueber die Oberflaeche aber nicht — sie war nur ueber die Vorgabe im
							     Seed steuerbar. Audit-Befund vom 01.08.2026. -->
							{@render roleToggle(item, 'HELFER', 'helfer')}
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/each}
</div>
