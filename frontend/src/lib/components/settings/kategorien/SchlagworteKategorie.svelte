<script>
	/**
	 * @component SchlagworteKategorie
	 * Die Pflegeseite der Schlagworte (docs/OFFEN.md 4.20, Stufe 1; Server seit Migration 143).
	 * Eine freie Liste bleibt wie in Littera durch Pflege brauchbar: Jede Zeile ist ein Wort mit
	 * seiner Titelzahl; Umbenennen, Zusammenführen und Verweis anlegen öffnen einen Dialog
	 * (SchlagwortPflegeDialog), Löschen fragt mit der Zahl der Titel nach, der Schalter setzt die
	 * Filter-Markierung für das Portal. Die Regeln (kein Verweis als Filter, keine Kette …) stehen
	 * im Server; die Seite bietet nur an, was dort erlaubt ist.
	 *
	 * Bauform nach M3 Lists: Zählung als „trailing text" („meta-information … such as a price,
	 * count"), Schalter „to toggle settings on or off", das Menü als „supplementary action … in
	 * the trailing position".
	 */
	import { onMount } from 'svelte';
	import { apiGet, apiPut, apiDelete } from '../../../apiFetch.js';
	import { toastStore } from '../../../stores/toastStore.svelte.js';
	import { loeschenBestaetigen } from '../../../stores/bestaetigung.svelte.js';
	import KategorieRahmen from '../KategorieRahmen.svelte';
	import SchlagwortPflegeDialog from '../SchlagwortPflegeDialog.svelte';
	import { menueEintraege, loeschFolgen, zaehlSatz } from '../schlagwortPflege.js';
	import Tabelle from '../../ui/Tabelle.svelte';
	import Suchfeld from '../../ui/Suchfeld.svelte';
	import Switch from '../../ui/Switch.svelte';
	import Menue from '../../ui/Menue.svelte';
	import Ladekreis from '../../ui/Ladekreis.svelte';
	import LadeFehler from '../../ui/LadeFehler.svelte';

	/** @typedef {import('../schlagwortPflege.js').SchlagwortZeile} Zeile */

	// Mehr Zeilen machen die Seite träge und helfen niemandem beim Suchen: Die Suche grenzt ein.
	const ANZEIGE_MAX = 200;

	let liste = $state(
		/** @type {{ zeilen: Zeile[], gesamt: number, verweise: number } | null} */ (null)
	);
	let ladeFehler = $state(false);
	let suche = $state('');
	let auftrag = $state(
		/** @type {{ art: 'umbenennen' | 'zusammenfuehren' | 'verweis', zeile: Zeile } | null} */ (null)
	);

	const zeilen = $derived(liste?.zeilen ?? []);
	const treffer = $derived.by(() => {
		const s = suche.trim().toLowerCase();
		if (!s) return zeilen;
		return zeilen.filter(
			(z) => z.wort.toLowerCase().includes(s) || z.verweise.some((v) => v.toLowerCase().includes(s))
		);
	});
	const filterZahl = $derived(zeilen.filter((z) => z.ist_filter).length);

	// Sequenznummer wie in useStudentProfile: Zwei schnell umgelegte Schalter laden die Liste
	// zweimal, und kam die ältere Antwort zuletzt, zeigte ein Schalter den alten Stand.
	let laufNr = 0;

	async function laden() {
		const meine = ++laufNr;
		try {
			const antwort = await apiGet('/api/schlagworte/pflege');
			if (meine !== laufNr) return; // eine jüngere Liste ist schon unterwegs oder da
			liste = antwort;
			ladeFehler = false;
		} catch {
			if (meine === laufNr) ladeFehler = true; // Meldung kam bereits aus apiGet.
		}
	}
	onMount(laden);

	/** @param {Zeile} z @param {string} id */
	function waehle(z, id) {
		if (id === 'loeschen') loeschen(z);
		else auftrag = { art: /** @type {any} */ (id), zeile: z };
	}

	/** @param {Zeile} z */
	async function loeschen(z) {
		if (!(await loeschenBestaetigen(`„${z.wort}“ löschen?`, loeschFolgen(z)))) return;
		try {
			await apiDelete(`/api/schlagworte/${z.id}`);
		} catch {
			return; // Meldung kam bereits aus apiDelete.
		}
		toastStore.addToast(`„${z.wort}“ gelöscht.`, 'success');
		await laden();
	}

	/** @param {Zeile} z @param {boolean} an */
	async function setzeFilter(z, an) {
		// Die Zeile zieht mit dem Schalter mit (die Zählung stimmt sofort) und springt bei einem
		// Fehler zurück; das Neuladen danach bringt den gespeicherten Stand.
		z.ist_filter = an;
		try {
			await apiPut(`/api/schlagworte/${z.id}/filter`, { ist_filter: an });
		} catch {
			z.ist_filter = !an; // Meldung kam bereits aus apiPut.
		}
		await laden();
	}
</script>

<KategorieRahmen
	titel="Schlagworte"
	kurz="Die Wörter am Titel: umbenennen, zusammenführen, Schreibweisen umleiten und die Filter im Portal wählen."
>
	{#snippet mehr()}
		<p>
			Ein Verweis leitet eine Schreibweise auf ein Wort: Wer am Titel „Tierfantasy“ einträgt,
			bekommt „Fantasy“. Umbenennen und Zusammenführen lassen die alte Schreibweise als Verweis
			stehen; wer das nicht will, wählt es im Dialog ab.
		</p>
		<p>Die als Filter markierten Wörter erscheinen im Portal mit dem nächsten Ausbau der Suche.</p>
	{/snippet}

	{#if ladeFehler}
		<LadeFehler onerneut={laden} />
	{:else if !liste}
		<div class="flex justify-center py-8"><Ladekreis size="lg" /></div>
	{:else if liste.gesamt === 0}
		<p class="text-sm text-on-surface-variant">
			Noch keine Schlagworte. Sie entstehen am Titel, im Buchformular und beim Bestellen.
		</p>
	{:else}
		<div class="flex flex-col gap-4">
			<div class="flex flex-wrap items-center gap-4">
				<Suchfeld
					bind:wert={suche}
					platzhalter="Schlagwort suchen …"
					etikett="Schlagwort suchen"
					klasse="w-72"
				/>
				<p class="text-sm text-on-surface-variant">
					{zaehlSatz(liste, filterZahl)}
				</p>
			</div>

			<div class="overflow-x-auto">
				<Tabelle beschriftung="Schlagworte mit Titelzahl, Filter und Aktionen">
					<thead>
						<tr>
							<th>Schlagwort</th>
							<th class="text-right">Titel</th>
							<th>Filter im Portal</th>
							<th><span class="sr-only">Aktionen</span></th>
						</tr>
					</thead>
					<tbody>
						{#each treffer.slice(0, ANZEIGE_MAX) as z (z.id)}
							<tr>
								<td>
									<div>{z.wort}</div>
									{#if z.verweis_auf_id}
										<div class="text-xs text-on-surface-variant">Verweis auf „{z.verweis_auf}“</div>
									{:else if z.verweise.length}
										<div class="text-xs text-on-surface-variant">auch: {z.verweise.join(', ')}</div>
									{/if}
								</td>
								<td class="text-right tabular-nums">{z.verweis_auf_id ? '' : z.titel}</td>
								<td>
									{#if !z.verweis_auf_id}
										<Switch
											checked={z.ist_filter}
											label="„{z.wort}“ als Filter im Portal"
											onchange={(an) => setzeFilter(z, an)}
										/>
									{/if}
								</td>
								<td class="w-12 text-right">
									<Menue
										etikett="Aktionen für „{z.wort}“"
										eintraege={menueEintraege(z)}
										onwahl={(id) => waehle(z, id)}
									/>
								</td>
							</tr>
						{/each}
					</tbody>
				</Tabelle>
			</div>

			{#if treffer.length === 0}
				<p class="text-sm text-on-surface-variant">Kein Schlagwort passt zur Suche.</p>
			{:else if treffer.length > ANZEIGE_MAX}
				<p class="text-sm text-on-surface-variant">
					{ANZEIGE_MAX} von {treffer.length} angezeigt — die Suche grenzt ein.
				</p>
			{/if}
			{#if liste.gesamt > zeilen.length}
				<p class="text-sm text-on-surface-variant">
					Geladen sind die ersten {zeilen.length} von {liste.gesamt} Einträgen (Schlagworte und Verweise)
					in alphabetischer Folge.
				</p>
			{/if}
		</div>
	{/if}
</KategorieRahmen>

<SchlagwortPflegeDialog {auftrag} {zeilen} onclose={() => (auftrag = null)} onfertig={laden} />
