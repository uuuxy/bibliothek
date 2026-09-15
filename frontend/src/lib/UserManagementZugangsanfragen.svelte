<script>
	/**
	 * @component UserManagementZugangsanfragen
	 * Offene Zugangsanfragen aus der Selbstanmeldung (Migration 086), als Zeile ÜBER
	 * der Benutzertabelle. Ohne sie lag ein Antrag als grauer Punkt „Inaktiv" zwischen
	 * 160 Kollegiumszeilen — niemand schaltet frei, was niemand sieht.
	 *
	 * Eigene Datei, weil UserManagement.svelte an der Dateigrößen-Ratsche steht.
	 *
	 * Littera-Treffer (16.09.2026): Eine aus Littera übernommene Lehrkraft hat nur eine
	 * Platzhalter-Adresse (internal/littera/schreiber_personen.go: …@littera.invalid). Meldet
	 * sie sich selbst an, findet die Anmeldung sie nicht und legt eine zweite Zeile an —
	 * Ausweis und Ausleihen am ersten Eintrag, Anmeldung am zweiten. Der Server verbindet
	 * bewusst nicht über den Namen (auth/selbstanmeldung_littera_pg_test.go); die Zeile hier
	 * nennt deshalb den gleichnamigen Eintrag, damit die Freischaltung ihn nicht übersieht.
	 *
	 * @prop {any[]} users - die geladene Benutzerliste (/api/benutzer)
	 */
	/** @type {{ users: any[] }} */
	let { users } = $props();

	const LITTERA_PLATZHALTER = '@littera.invalid';

	const anfragen = $derived(users.filter((u) => !u.aktiv && u.zugang_beantragt_am));
	const namen = $derived(
		anfragen.map((u) => `${u.vorname} ${u.nachname}`.trim() || u.email).join(', ')
	);

	/** @param {string | undefined} s */
	const norm = (s) => (s ?? '').trim().toLowerCase();

	const treffer = $derived(
		anfragen.flatMap((a) =>
			users
				.filter(
					(u) =>
						norm(u.email).endsWith(LITTERA_PLATZHALTER) &&
						norm(u.vorname) === norm(a.vorname) &&
						norm(u.nachname) === norm(a.nachname)
				)
				.map((u) => ({ name: `${a.vorname} ${a.nachname}`.trim(), ausweis: u.barcode_id || '' }))
		)
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
		{#each treffer as t (`${t.name}|${t.ausweis}`)}
			<span class="mt-2 block">
				{t.name} steht schon aus der Littera-Übernahme im Bestand{t.ausweis
					? ` (Ausweis ${t.ausweis})`
					: ''}: Anfrage löschen und dort unter „Bearbeiten“ die E-Mail eintragen — sonst hat die
				Lehrkraft zwei Einträge.
			</span>
		{/each}
	</p>
{/if}
