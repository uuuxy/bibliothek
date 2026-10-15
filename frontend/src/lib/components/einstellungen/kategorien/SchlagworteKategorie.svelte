<script>
	/**
	 * @component SchlagworteKategorie
	 * Die Pflegeseite der Schlagworte (docs/OFFEN.md 4.20, Stufe 1; Server seit Migration 143).
	 * Eine freie Liste bleibt wie in Littera durch Pflege brauchbar: Jede Zeile ist ein Wort mit
	 * seiner Titelzahl (SchlagwortTabelle); Umbenennen, Zusammenführen und Verweis anlegen öffnen
	 * einen Dialog (SchlagwortPflegeDialog), Löschen fragt mit der Zahl der Titel nach, der
	 * Schalter setzt die Filter-Markierung für das Portal. Markierte Zeilen löscht die
	 * AuswahlLeiste auf einmal — dieselbe Tür wie das Löschen aus dem Menü. Die Regeln (kein
	 * Verweis als Filter, keine Kette …) stehen im Server; die Seite bietet nur an, was dort
	 * erlaubt ist.
	 */
	import { onMount } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { Trash } from '@lucide/svelte';
	import { apiPut, apiPost } from '../../../apiFetch.js';
	import { toastStore } from '../../../stores/toastStore.svelte.js';
	import { loeschenBestaetigen } from '../../../stores/bestaetigung.svelte.js';
	import KategorieRahmen from '../KategorieRahmen.svelte';
	import SchlagwortPflegeDialog from '../SchlagwortPflegeDialog.svelte';
	import SchlagwortTabelle from '../SchlagwortTabelle.svelte';
	import { loeschFrage, loeschErgebnis, zaehlSatz } from '../schlagwortPflege.js';
	import { erzeugeSchlagwortPflegeListe } from '../schlagwortPflegeListe.svelte.js';
	import Suchfeld from '../../ui/Suchfeld.svelte';
	import AuswahlLeiste from '../../ui/AuswahlLeiste.svelte';
	import Button from '../../ui/Button.svelte';
	import Ladekreis from '../../ui/Ladekreis.svelte';
	import LadeFehler from '../../ui/LadeFehler.svelte';

	/** @typedef {import('../schlagwortPflege.js').SchlagwortZeile} Zeile */

	// Gesucht wird am Server über alle Wörter (schlagwortPflegeListe.svelte.js, 30.09.2026); die
	// Antwort bringt höchstens 200 Zeilen und sagt, wie viele zur Suche passen.
	const pflege = erzeugeSchlagwortPflegeListe();
	const liste = $derived(pflege.liste);
	let auftrag = $state(
		/** @type {{ art: 'umbenennen' | 'zusammenfuehren' | 'verweis', zeile: Zeile } | null} */ (null)
	);

	const sichtbar = $derived(liste?.zeilen ?? []);

	// Markiert zählt nur, was zu sehen ist (wie leserAuswahl.svelte.js): Wer die Suche ändert,
	// löscht nicht, was er nicht mehr vor sich hat.
	const auswahl = new SvelteSet();
	const markiert = $derived(sichtbar.filter((z) => auswahl.has(z.id)));
	/** @param {string} id */
	function umschalten(id) {
		if (!auswahl.delete(id)) auswahl.add(id);
	}
	function alleUmschalten() {
		const alle = markiert.length === sichtbar.length;
		auswahl.clear();
		if (!alle) for (const z of sichtbar) auswahl.add(z.id);
	}

	onMount(pflege.lade);

	/** @param {Zeile} z @param {string} id */
	function waehle(z, id) {
		if (id === 'loeschen') loeschen([z]);
		else auftrag = { art: /** @type {any} */ (id), zeile: z };
	}

	/** @param {Zeile[]} gewaehlt - eins aus dem Menü oder die markierten */
	async function loeschen(gewaehlt) {
		const frage = loeschFrage(gewaehlt);
		if (!(await loeschenBestaetigen(frage.titel, frage.text))) return;
		try {
			const ids = gewaehlt.map((z) => z.id);
			const antwort = await apiPost('/api/schlagworte/loeschen', { ids });
			toastStore.addToast(loeschErgebnis(gewaehlt, antwort), 'success');
			for (const id of ids) auswahl.delete(id);
		} catch {
			// Meldung kam bereits aus apiPost; gelöscht ist nichts (alle oder keins). Neu laden
			// zeigt, was ein anderer inzwischen geändert hat.
		}
		await pflege.lade();
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
		await pflege.lade();
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
		<p>
			Die als Filter markierten Wörter stehen in „Mein Portal“ unter „Reservieren & Melden“ als
			Filter, sobald ein Titel mit dem Wort im öffentlichen Katalog steht — Lernmittel stehen dort
			nicht. Gesucht wird über alle Wörter und Verweise, markiert oder nicht.
		</p>
	{/snippet}

	{#if pflege.ladeFehler}
		<LadeFehler onerneut={pflege.lade} />
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
					bind:wert={pflege.suche}
					oninput={pflege.angestossen}
					platzhalter="Schlagwort suchen …"
					etikett="Schlagwort suchen"
					klasse="w-72"
				/>
				<p class="text-sm text-on-surface-variant">
					{zaehlSatz(liste, liste.filter)}
				</p>
			</div>

			<SchlagwortTabelle
				zeilen={sichtbar}
				{auswahl}
				onumschalten={umschalten}
				onalle={alleUmschalten}
				onfilter={setzeFilter}
				onwahl={waehle}
			/>

			{#if liste.treffer === 0}
				<p class="text-sm text-on-surface-variant">Kein Schlagwort passt zur Suche.</p>
			{:else if liste.treffer > sichtbar.length}
				<p class="text-sm text-on-surface-variant">
					{sichtbar.length} von {liste.treffer} angezeigt — die Suche grenzt ein.
				</p>
			{/if}
			{#if markiert.length > 0}
				<AuswahlLeiste
					satz="{markiert.length} markiert"
					beschriftung="Aktionen für die markierten Schlagworte"
					onleeren={() => auswahl.clear()}
				>
					<Button variant="danger" onclick={() => loeschen(markiert)}>
						<Trash class="h-4 w-4" aria-hidden="true" />
						Löschen
					</Button>
				</AuswahlLeiste>
			{/if}
		</div>
	{/if}
</KategorieRahmen>

<SchlagwortPflegeDialog {auftrag} onclose={() => (auftrag = null)} onfertig={pflege.lade} />
