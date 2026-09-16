<script>
	/**
	 * @component UserManagementZugangsanfragen
	 * Offene Zugangsanfragen aus der Selbstanmeldung (Migration 086), als Zeile ÜBER
	 * der Benutzertabelle. Ohne sie lag ein Antrag als grauer Punkt „Inaktiv" zwischen
	 * 160 Kollegiumszeilen — niemand schaltet frei, was niemand sieht.
	 *
	 * Eigene Datei, weil UserManagement.svelte an der Dateigrößen-Ratsche steht.
	 *
	 * Die Zeile warnt vor DEMSELBEN Fehler aus zwei Quellen — beide Male steht die Person
	 * schon in der Leserdatei, und beide Male würde die Freischaltung einen zweiten
	 * Eintrag festschreiben (Ausweis und Ausleihen am ersten, die Anmeldung am zweiten):
	 *
	 *  1. Littera-Treffer (16.09.2026): Eine aus Littera übernommene Lehrkraft hat nur eine
	 *     Platzhalter-Adresse (internal/littera/schreiber_personen.go: …@littera.invalid).
	 *     Der Server verbindet bewusst nicht über den Namen
	 *     (auth/selbstanmeldung_littera_pg_test.go).
	 *  2. Leserzeile OHNE Konto (16.09.2026, abends): Der Altbestand — ein Kollege, der vor
	 *     der Schul-E-Mail-Pflicht von Hand eingetragen wurde. Er steht in der Leserdatei,
	 *     hat aber kein Konto, also findet ihn die Selbstanmeldung nicht. In `users` steht
	 *     er ebenfalls nicht — die Warnung oben konnte ihn gar nicht sehen. Gesucht wird
	 *     er deshalb dort, wo er steht: über die Kandidatensuche des Zusammenführens, von
	 *     der Leserzeile des Antrags aus (die der Wächter trg_benutzer_hat_leserzeile als
	 *     Kollegen anlegt — die Suche hält die Seite und liefert nur Kollegium).
	 *
	 * @prop {any[]} users - die geladene Benutzerliste (/api/benutzer)
	 */
	import { apiFetch } from './apiFetch.js';

	/** @type {{ users: any[] }} */
	let { users } = $props();

	const LITTERA_PLATZHALTER = '@littera.invalid';

	const anfragen = $derived(users.filter((u) => !u.aktiv && u.zugang_beantragt_am));
	const namen = $derived(
		anfragen.map((u) => `${u.vorname} ${u.nachname}`.trim() || u.email).join(', ')
	);

	/** @param {string | undefined} s */
	const norm = (s) => (s ?? '').trim().toLowerCase();

	/** @param {any} a @param {any} u */
	const gleicherName = (a, u) =>
		norm(u.vorname) === norm(a.vorname) && norm(u.nachname) === norm(a.nachname);

	const treffer = $derived(
		anfragen.flatMap((a) =>
			users
				.filter((u) => norm(u.email).endsWith(LITTERA_PLATZHALTER) && gleicherName(a, u))
				.map((u) => ({ name: `${a.vorname} ${a.nachname}`.trim(), ausweis: u.barcode_id || '' }))
		)
	);

	/** @type {{ name: string, ausweis: string }[]} */
	let ohneKonto = $state([]);

	// Die Suche braucht das Recht merge_students. Wer es nicht hat (eine Leitung etwa),
	// bekommt 403 — dann bleibt die Zusatzzeile einfach aus. Eine Fehlermeldung an dieser
	// Stelle wäre Lärm über ein Recht, das mit dem Freischalten nichts zu tun hat.
	$effect(() => {
		const offen = anfragen;
		if (offen.length === 0) {
			ohneKonto = [];
			return;
		}
		let verworfen = false;
		(async () => {
			/** @type {{ name: string, ausweis: string }[]} */
			const gefunden = [];
			for (const a of offen) {
				if (!a.leser_id || !a.nachname) continue;
				try {
					const res = await apiFetch(
						`/api/schueler/${a.leser_id}/zusammenfuehren-kandidaten?q=${encodeURIComponent(a.nachname)}`
					);
					if (!res.ok) continue;
					for (const k of (await res.json()) ?? []) {
						if (gleicherName(a, k)) {
							gefunden.push({
								name: `${a.vorname} ${a.nachname}`.trim(),
								ausweis: k.barcode_id || ''
							});
						}
					}
				} catch {
					// Netzwerkfehler: Die Freischaltung selbst hängt nicht daran.
				}
			}
			if (!verworfen) ohneKonto = gefunden;
		})();
		return () => {
			verworfen = true;
		};
	});
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
		{#each ohneKonto as t (`ohne|${t.name}|${t.ausweis}`)}
			<span class="mt-2 block">
				{t.name} steht schon in der Leserdatei, bisher ohne Zugang{t.ausweis
					? ` (Ausweis ${t.ausweis})`
					: ''}: Anfrage löschen und dort unter „Bearbeiten“ die Schul-E-Mail eintragen — dann
				gehören Ausweis, Ausleihen und Anmeldung zu einem Eintrag.
			</span>
		{/each}
	</p>
{/if}
