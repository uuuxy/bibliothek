<script>
	import { apiGet, apiDelete } from '../../apiFetch.js';
	import Tabelle from '../ui/Tabelle.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { onMount } from 'svelte';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import { loeschenBestaetigen } from '../../stores/bestaetigung.svelte.js';
	import Button from '../ui/Button.svelte';
	import Icon from '../ui/Icon.svelte';
	import KlassenleitungNeu from './KlassenleitungNeu.svelte';

	/** @type {{klasse: string, lehrer_email: string}[]} */
	let mappingRows = $state([]);
	let zuordnungLaedt = $state(false);
	// Nach einem gescheiterten Laden steht weder Liste noch Formular da: Hinzufügen ersetzt die
	// Zuordnung derselben Klasse, und die wäre gerade nicht zu sehen.
	let ladeFehler = $state(false);

	async function fetchMapping() {
		zuordnungLaedt = true;
		try {
			mappingRows = (await apiGet('/api/klassen-mapping')) || [];
			ladeFehler = false;
		} catch {
			// Scheitert nur das Neuladen, bleibt die Liste von vorhin stehen; den Grund meldet apiGet.
			ladeFehler = mappingRows.length === 0;
		} finally {
			zuordnungLaedt = false;
		}
	}

	onMount(() => {
		fetchMapping();
	});

	// Die Klasse, deren Zuordnung gerade gelöscht wird; bis die Liste nachgeladen ist, sind die
	// Löschknöpfe gesperrt. Ein zweiter Klick schickte ein zweites DELETE, und nach dem Löschen
	// stand „Kein Eintrag für diese Klasse".
	let loeschtKlasse = $state(/** @type {string | null} */ (null));

	// Fragt vorher wie jedes Löschen im Haus: Nichts nimmt es zurück, und der Versand an die Klasse
	// entfällt still.
	/** @param {string} klasse */
	async function loescheZuordnung(klasse) {
		if (loeschtKlasse) return;
		const folge =
			'Mahnlisten und Kontoauszüge dieser Klasse werden danach beim Versand übersprungen';
		if (!(await loeschenBestaetigen(`Zuordnung für ${klasse} löschen?`, folge))) return;
		loeschtKlasse = klasse;
		try {
			await apiDelete(`/api/klassen-mapping/${encodeURIComponent(klasse)}`);
			snackbars.addToast(`Zuordnung für ${klasse} gelöscht`, 'success');
		} catch {
			// Toast already shown by apiDelete
		} finally {
			// Auch nach einem Fehler: Ging nur die Antwort verloren, ist die Zuordnung schon gelöscht.
			await fetchMapping();
			loeschtKlasse = null;
		}
	}
</script>

<!-- Flach & edge-to-edge: keine umschließende Box, flaches Listen-Layout (divide-y) -->
<div class="max-w-3xl space-y-8">
	<div>
		<h3 class="text-title-medium text-on-surface">E-Mail Routing für Mahnungen</h3>
		<p class="mt-1 max-w-2xl text-sm text-on-surface-variant">
			Von Hand eingetragen, die LUSD liefert es nicht; die Versetzung rückt jede Zuordnung eine
			Stufe hoch, außer vor Klasse 7 und vor der Oberstufe
		</p>
	</div>

	{#if zuordnungLaedt}
		<div class="py-8 flex justify-center">
			<Ladekreis size="lg" verzoegert />
		</div>
	{:else if ladeFehler}
		<LadeFehler
			onerneut={fetchMapping}
			titel="Zuordnungen nicht geladen"
			text="Die Klassenleitungen konnten nicht abgerufen werden. Das Formular bleibt geschlossen, ein neuer Eintrag ersetzte sonst die Zuordnung derselben Klasse, ohne dass sie zu sehen war."
		/>
	{:else if mappingRows.length === 0}
		<p class="text-sm text-on-surface-variant py-4">Noch keine Zuordnungen vorhanden</p>
	{:else}
		<Tabelle beschriftung="Klassenleitungen und ihre E-Mail-Adressen">
			<thead>
				<tr>
					<th>Klasse</th>
					<th>Lehrer-E-Mail</th>
					<th class="text-right">Aktion</th>
				</tr>
			</thead>
			<tbody>
				{#each mappingRows as row, _i (_i)}
					<tr>
						<td class="text-body-medium-emphasized">{row.klasse}</td>
						<td>{row.lehrer_email}</td>
						<td class="text-right">
							<Button
								variant="symbol"
								size="sm"
								loeschen
								onclick={() => loescheZuordnung(row.klasse)}
								disabled={loeschtKlasse !== null}
								aria-label="Zuordnung für {row.klasse} löschen"
								data-tip="Zuordnung löschen"
							>
								{#if loeschtKlasse === row.klasse}
									<Ladekreis size="sm" farbe="aktuell" />
								{:else}
									<Icon name="delete" />
								{/if}
							</Button>
						</td>
					</tr>
				{/each}
			</tbody>
		</Tabelle>
	{/if}

	<!-- Neuen Eintrag hinzufügen: flacher Eingabeblock ohne Box -->
	{#if !ladeFehler}
		<KlassenleitungNeu ongespeichert={fetchMapping} />
	{/if}
</div>
