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
	 * ZUGEORDNET statt gelöscht (16.09.2026, abends): Bis hierher riet die Zeile, die
	 * Anfrage zu löschen und die Adresse am vorhandenen Eintrag nachzutragen. Der Rat war
	 * erzwungen — benutzer_email_unique gibt die Adresse nicht frei, solange das
	 * Anfrage-Konto sie hält — und er hinterließ die Leserzeile des Wächters als Waise:
	 * ohne Konto, ohne Ausweis, mit dem aus der Adresse geratenen Namen. Also genau den
	 * Doppeleintrag, vor dem die Zeile warnt. Der Knopf macht daraus einen Schritt: Das
	 * Konto zieht auf den vorhandenen Eintrag um, die Zeile des Wächters geht darin auf
	 * (api/zugangsanfrage_zuordnen_pg_test.go).
	 *
	 * @prop {any[]} users - die geladene Benutzerliste (/api/benutzer)
	 * @prop {() => any} onZugeordnet - Liste neu laden, nachdem zwei Einträge einer wurden
	 */
	import { apiFetch } from './apiFetch.js';
	import { ordneZu } from './zugangsanfrageZuordnen.js';
	import { hatRecht } from './menu.js';
	import { authStore } from './stores/authStore.svelte.js';
	import Button from './components/ui/Button.svelte';

	/** @type {{ users: any[], onZugeordnet?: () => any }} */
	let { users, onZugeordnet } = $props();

	const LITTERA_PLATZHALTER = '@littera.invalid';

	// Zusammenführen hängt am Recht merge_students (api/routes_students.go). Ohne das
	// Recht bleibt der Hinweis stehen, aber ohne Knopf — der Server lehnte ihn mit 403 ab.
	const darfZuordnen = $derived(hatRecht(authStore.currentUser, 'merge_students'));

	// Der Knopf braucht BEIDE Kennungen. Fehlt eine, bleibt die WARNUNG trotzdem stehen:
	// Die Dublette ist dann genauso da, nur von Hand aufzulösen — ein stilles Verschwinden
	// des Hinweises wäre der schlechtere Ausgang.
	/** @param {{ zielID: string, quelleID: string }} d */
	const istZuordenbar = (d) => Boolean(d.zielID && d.quelleID);

	const anfragen = $derived(users.filter((u) => !u.aktiv && u.zugang_beantragt_am));
	const namen = $derived(
		anfragen.map((u) => `${u.vorname} ${u.nachname}`.trim() || u.email).join(', ')
	);

	/** @param {string | undefined} s */
	const norm = (s) => (s ?? '').trim().toLowerCase();

	/** @param {any} a @param {any} u */
	const gleicherName = (a, u) =>
		norm(u.vorname) === norm(a.vorname) && norm(u.nachname) === norm(a.nachname);

	// Beide Quellen führen zur selben Handlung, also tragen sie dieselbe Form: `zielID`
	// ist der vorhandene Eintrag (er BLEIBT, mit Ausweis und Büchern), `quelleID` die
	// Leserzeile des Antrags (sie geht darin auf). Die Richtung ist der Kern — verkehrt
	// herum verschwände der Eintrag mit Ausweis und Büchern, und der aus der Adresse
	// geratene Name bliebe übrig.
	const trefferAusListe = $derived(
		anfragen.flatMap((a) =>
			users
				.filter((u) => norm(u.email).endsWith(LITTERA_PLATZHALTER) && gleicherName(a, u))
				.map((u) => ({
					schluessel: `lit|${a.id}|${u.id}`,
					name: `${a.vorname} ${a.nachname}`.trim(),
					ausweis: u.barcode_id || '',
					herkunft: 'steht schon aus der Littera-Übernahme im Bestand',
					zielID: u.leser_id || '',
					quelleID: a.leser_id || ''
				}))
		)
	);

	/** @type {{ schluessel: string, name: string, ausweis: string, herkunft: string, zielID: string, quelleID: string }[]} */
	let ohneKonto = $state([]);

	/** @type {string} */
	let laeuft = $state('');
	/** @type {string} */
	let fehler = $state('');

	const dubletten = $derived([...trefferAusListe, ...ohneKonto]);

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
			/** @type {typeof ohneKonto} */
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
								schluessel: `ohne|${a.id}|${k.id}`,
								name: `${a.vorname} ${a.nachname}`.trim(),
								ausweis: k.barcode_id || '',
								herkunft: 'steht schon in der Leserdatei, bisher ohne Zugang',
								zielID: k.id,
								quelleID: a.leser_id
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

	/** @param {(typeof dubletten)[number]} d */
	async function zuordnen(d) {
		laeuft = d.schluessel;
		fehler = await ordneZu(d.zielID, d.quelleID);
		laeuft = '';
		if (fehler) return;
		// Die Trefferliste stammt aus einer Suche VOR dem Zusammenführen — eine ihrer
		// beiden Zeilen gibt es jetzt nicht mehr. Leeren, statt sie stehen zu lassen:
		// Der $effect füllt sie neu, sobald die Benutzerliste nachgeladen ist.
		ohneKonto = [];
		await onZugeordnet?.();
	}
</script>

{#if anfragen.length > 0}
	<p
		class="mb-4 rounded-sm bg-secondary-container px-4 py-3 text-sm text-on-secondary-container"
		role="status"
	>
		<strong>{anfragen.length} Zugangsanfrage{anfragen.length === 1 ? '' : 'n'}</strong>
		aus der Selbstanmeldung {anfragen.length === 1 ? 'wartet' : 'warten'} auf Freischaltung:
		{namen} — Person prüfen, dann „Bearbeiten“ → Aktiv.
		{#each dubletten as d (d.schluessel)}
			<span class="mt-2 flex flex-wrap items-center gap-2">
				<span>
					{d.name}
					{d.herkunft}{d.ausweis ? ` (Ausweis ${d.ausweis})` : ''}:
					{#if darfZuordnen && istZuordenbar(d)}
						Beide Einträge gehören zusammen — dann hängen Ausweis, Ausleihen und Anmeldung an einem.
					{:else}
						Beide Einträge gehören zusammen. Wer Einträge zusammenführen darf, kann daraus einen
						machen — sonst hat die Person zwei.
					{/if}
				</span>
				{#if darfZuordnen && istZuordenbar(d)}
					<Button
						variant="secondary"
						size="sm"
						disabled={laeuft !== ''}
						onclick={() => zuordnen(d)}
					>
						{laeuft === d.schluessel ? 'Wird zugeordnet …' : 'Das ist dieselbe Person'}
					</Button>
				{/if}
			</span>
		{/each}
		{#if fehler}
			<span class="mt-2 block font-semibold text-error">{fehler}</span>
		{/if}
	</p>
{/if}
