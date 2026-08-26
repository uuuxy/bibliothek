<script>
	/**
	 * @component UserManagementZugangsanfragen
	 * Offene Zugangsanfragen aus der Selbstanmeldung (Migration 086), als Zeile ÜBER
	 * der Benutzertabelle. Ohne sie lag ein Antrag als grauer Punkt „Inaktiv" zwischen
	 * 160 Kollegiumszeilen — niemand schaltet frei, was niemand sieht.
	 *
	 * Eigene Datei, weil UserManagement.svelte an der Dateigrößen-Ratsche steht.
	 *
	 * @prop {any[]} users - die geladene Benutzerliste (/api/benutzer)
	 */
	/** @type {{ users: any[] }} */
	let { users } = $props();

	const anfragen = $derived(users.filter((u) => !u.aktiv && u.zugang_beantragt_am));
	const namen = $derived(
		anfragen.map((u) => `${u.vorname} ${u.nachname}`.trim() || u.email).join(', ')
	);
</script>

{#if anfragen.length > 0}
	<p
		class="mb-4 rounded-sm bg-secondary-container px-4 py-3 text-sm text-on-secondary-container"
		role="status"
	>
		<strong>{anfragen.length} Zugangsanfrage{anfragen.length === 1 ? '' : 'n'}</strong>
		aus der Selbstanmeldung {anfragen.length === 1 ? 'wartet' : 'warten'} auf Freischaltung:
		{namen} — Person prüfen, dann „Bearbeiten“ → Aktiv.
	</p>
{/if}
