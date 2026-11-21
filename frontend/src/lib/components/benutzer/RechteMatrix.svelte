<script>
	import Icon from '../ui/Icon.svelte';
	import Switch from '../ui/Switch.svelte';
	/**
	 * @component RechteMatrix
	 * Reine Darstellung des Rechte-Editors (flach, edge-to-edge). Logik/State liegen
	 * im Eltern-BenutzerUndRechte; hier nur Anzeige + Toggle-Callback.
	 *
	 * @typedef {Object} Props
	 * @property {any[]} beschreibungen - Kategorien/Items (rechteKategorien.js).
	 * @property {Record<string, Record<string, boolean>>} rechteJeRolle
	 * @property {Record<string, boolean>} updatingKeys
	 * @property {(role: string, key: string, currentVal: boolean) => void} onToggle
	 * @property {boolean} [schreibgeschuetzt] - PUT /api/admin/permissions ist Admin-only; mit
	 *   manage_users sieht man die Matrix, kann sie aber nicht ändern.
	 */

	/** @type {Props} */
	let {
		beschreibungen,
		rechteJeRolle,
		updatingKeys,
		onToggle,
		schreibgeschuetzt = false
	} = $props();
</script>

<!-- Ein Schalter-Block für alle Rollen. Das {#key} setzt den Schalter nach jedem Speichern
     auf den Stand der Matrix zurück — schlägt der PUT fehl, bliebe der Schalter sonst
     umgelegt, obwohl das Recht unverändert ist. -->
{#snippet roleToggle(item, roleLabel, roleKey)}
	{@const isUpdating = updatingKeys[`${roleKey}-${item.key}`]}
	{@const wert = rechteJeRolle[roleKey]?.[item.key] ?? false}
	<div class="flex items-center gap-3">
		<span class="text-body-small-emphasized text-on-surface-variant tracking-wider w-24 text-right"
			>{roleLabel}</span
		>
		{#key isUpdating}
			<Switch
				checked={wert}
				disabled={isUpdating || schreibgeschuetzt}
				etikett="{roleLabel} Rechte umschalten"
				onchange={() => onToggle(roleKey, item.key, wert)}
			/>
		{/key}
	</div>
{/snippet}

<div class="space-y-12">
	<!-- Gesperrte Schalter bekommen keinen Fokus: Die Erklärung steht sichtbar über der Matrix,
	     nicht in einer Sprechblase (M3 Tooltips: „Don't hide critical information within
	     tooltips“). -->
	{#if schreibgeschuetzt}
		<p class="text-sm text-on-surface-variant">Nur ein Administrator kann die Rechte ändern</p>
	{/if}
	{#each beschreibungen as cat, _i (_i)}
		<div>
			<div class="pb-3 mb-1 border-b border-outline-variant flex items-center gap-3">
				<Icon name={cat.icon} class="text-on-surface-variant" />
				<h3 class="text-on-surface text-lg tracking-tight">{cat.category}</h3>
			</div>

			<div class="divide-y divide-outline-variant">
				{#each cat.items as item, _i (_i)}
					<div class="py-6 px-1 flex flex-col md:flex-row md:items-center justify-between gap-4">
						<div class="max-w-xl space-y-1">
							<span class="text-on-surface text-body-large-emphasized tracking-tight"
								>{item.label}</span
							>
							<p class="text-sm text-on-surface-variant leading-relaxed font-medium">{item.desc}</p>
						</div>

						<div class="flex items-center gap-8 md:gap-12 shrink-0">
							<!-- Admin (Read-only) -->
							<div class="flex items-center gap-3">
								<span
									class="text-body-small-emphasized text-on-surface-variant tracking-wider w-24 text-right"
									>ADMIN</span
								>
								<Switch checked disabled etikett="Administrator hat immer alle Rechte" />
							</div>

							{@render roleToggle(item, 'LEITUNG', 'leitung')}
							{@render roleToggle(item, 'MITARBEITER', 'mitarbeiter')}
							<!-- Kollegium hat keine Spalte: Es ist keine Rolle, sondern der Ausgangszustand
							     jeder Lehrkraft. Seine Rechte stehen fest in internal/db/rechte_vorgabe.go; was „Mein Portal“
							     kann, ist eine Produktentscheidung und kein Schalter je Schule. -->
							{@render roleToggle(item, 'HELFER', 'helfer')}
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/each}
</div>
