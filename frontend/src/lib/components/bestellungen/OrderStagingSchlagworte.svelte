<!-- @component Die Schlagworte im Staging-Fenster der Bestellsuche (Migration 138).

     Gespeichert wird die Menge als Ganzes (PUT /api/buecher/titel/{id}/schlagworte),
     deshalb lädt der Teil zuerst die vorhandenen. Bis sie da sind — oder wenn das
     scheitert — bleibt das Feld gesperrt; sonst ersetzte das erste Wort still alle, die
     der Titel schon trägt. Wie Signatur und Lernmittel daneben schreibt er nur, wenn
     jemand etwas geändert hat.

     Eigene Datei, weil OrderStaging mit dem Feld über der 200-Zeilen-Marke lag. -->
<script>
	import {
		ladeSchlagwortVorschlaege,
		ladeTitelSchlagworte,
		setzeTitelSchlagworte
	} from '../../utils/schlagworte.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import ChipFeld from '../ui/ChipFeld.svelte';
	import { untrack } from 'svelte';

	/** @type {{ titelId: string }} */
	let { titelId } = $props();

	/** @type {string[]} */
	let schlagworte = $state([]);
	/** @type {string[] | null} */
	let beiStart = $state(null);
	let fehlen = $state(false);
	/** @type {{ wert: string, beschreibung?: string }[]} */
	let vorschlaege = $state([]);

	$effect(() => {
		let abgebrochen = false;
		ladeSchlagwortVorschlaege().then((liste) => {
			if (!abgebrochen) vorschlaege = liste;
		});
		ladeTitelSchlagworte(untrack(() => titelId)).then(
			(liste) => {
				if (abgebrochen) return;
				schlagworte = liste;
				beiStart = liste;
			},
			() => {
				if (!abgebrochen) fehlen = true;
			}
		);
		return () => {
			abgebrochen = true;
		};
	});

	/** Dieselbe Menge, egal in welcher Reihenfolge — die Chips hängen neue hinten an. */
	const geaendert = $derived(
		beiStart !== null && [...schlagworte].sort().join('\n') !== [...beiStart].sort().join('\n')
	);

	/** Vom Fenster beim „In den Warenkorb" gerufen. Ein Fehler hält die Bestellung nicht auf. */
	export async function speichereWennGeaendert() {
		if (!geaendert) return;
		try {
			await setzeTitelSchlagworte(titelId, schlagworte);
		} catch {
			toastStore.addToast(
				'Schlagworte konnten nicht gespeichert werden — Titel wird trotzdem bestellt.',
				'error'
			);
		}
	}
</script>

<div class="space-y-1">
	<label for="stagedSchlagworte" class="text-xs font-medium text-on-surface-variant">
		Schlagworte
	</label>
	<ChipFeld
		id="stagedSchlagworte"
		aria-label="Schlagworte"
		bind:werte={schlagworte}
		{vorschlaege}
		disabled={beiStart === null}
		hint={fehlen
			? 'Konnten nicht geladen werden — bitte später im Buchformular eintragen.'
			: undefined}
	/>
</div>
