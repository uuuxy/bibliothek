<script>
	import { erzeugeAuswahl } from './bestandstabelleAuswahl.svelte.js';
	import Tabelle from '../../../../lib/components/ui/Tabelle.svelte';
	import BestandstabelleLeiste from '$lib/components/admin/BestandstabelleLeiste.svelte';
	import BestandstabelleZeile from '$lib/components/admin/BestandstabelleZeile.svelte';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import Kaestchen from '../../../../lib/components/ui/Kaestchen.svelte';

	/**
	 * @type {{
	 *   titelListe: any[],
	 *   laedt: boolean,
	 *   onOpenDetail: (book: any) => void,
	 *   onCreateNew: () => void,
	 *   onScan: () => void,
	 *   onDelete: (ids: string[]) => Promise<void> | void,
	 *   onAssignClass: (ids: string[]) => void,
	 *   onRetryCovers: () => void
	 * }}
	 */
	let {
		titelListe,
		laedt,
		onOpenDetail,
		onCreateNew,
		onScan,
		onDelete,
		onAssignClass,
		onRetryCovers
	} = $props();

	// Auswahl für die Massenaktionen: bestandstabelleAuswahl.svelte.js. Der Effekt hält sie auf
	// dem, was in der Liste steht — sonst trifft „Löschen" nach dem Filtern Unsichtbares.
	const auswahl = erzeugeAuswahl();
	$effect(() => auswahl.angleichen(titelListe));

	// Performance: Begrenzung der gerenderten DOM-Elemente
	let maxVisible = $state(50);

	// Die Markierung bleibt: Was gelöscht ist, verlässt die Liste und mit ihr die Auswahl
	// (angleichen); nach einer Ablehnung oder „Abbrechen" steht sie noch da. Solange das Löschen
	// läuft, geht kein zweiter Auftrag hinaus.
	let loescht = $state(false);
	async function loesche() {
		if (loescht) return;
		loescht = true;
		try {
			await onDelete(auswahl.ids);
		} finally {
			loescht = false;
		}
	}

	function ordneKlasseZu() {
		// Auswahl bleibt bestehen, bis der Dialog abgeschlossen/abgebrochen ist —
		// der Picker hält die IDs bereits über seine Prop.
		onAssignClass(auswahl.ids);
	}
</script>

<div class="w-full">
	<BestandstabelleLeiste
		titelAnzahl={titelListe.length}
		anzahlGewaehlt={auswahl.anzahl}
		{loescht}
		onDelete={loesche}
		onAssignClass={ordneKlasseZu}
		{onScan}
		{onCreateNew}
		{onRetryCovers}
	/>

	<div class="overflow-x-auto">
		<Tabelle beschriftung="Titel im Bestand">
			<thead class="font-medium">
				<tr>
					<th class="w-10">
						<Kaestchen
							aria-label="Alle Bücher auswählen"
							checked={auswahl.alleGewaehlt(titelListe)}
							onclick={() => auswahl.alleUmschalten(titelListe)}
						/>
					</th>
					<th class="w-20">Cover</th>
					<th>Titel</th>
					<th>Fach</th>
					<th>Jahrgang</th>
					<th>Art</th>
					<th>Standort</th>
					<th class="text-right">Zuletzt geprüft</th>
					<th class="text-right">Bestand</th>
					<th class="w-10"></th>
				</tr>
			</thead>

			<tbody>
				<!-- Die Reihenfolge kommt vom Server: nach dem Titel (GET /api/books). -->
				{#each titelListe.slice(0, maxVisible) as book (book.id)}
					<BestandstabelleZeile
						titel={book}
						gewaehlt={auswahl.enthaelt(book.id)}
						{onOpenDetail}
						onToggleSelect={(id) => auswahl.umschalten(id)}
					/>
				{/each}

				{#if titelListe.length === 0 && !laedt}
					<tr>
						<td colspan="10" class="text-center">Keine Bücher gefunden.</td>
					</tr>
				{/if}
			</tbody>
		</Tabelle>

		{#if titelListe.length > maxVisible}
			<div class="flex justify-center p-4">
				<Button variant="secondary" onclick={() => (maxVisible += 50)}>
					Weitere Bücher laden ({titelListe.length - maxVisible} verbleibend)
				</Button>
			</div>
		{/if}
	</div>
</div>
