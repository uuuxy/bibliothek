<!-- @component Die Schlagworte im Staging-Fenster der Bestellsuche (Migration 138).

     Gespeichert wird die Menge als Ganzes (PUT /api/buecher/titel/{id}/schlagworte),
     deshalb lädt der Teil zuerst die vorhandenen. Bis sie da sind — oder wenn das
     scheitert — bleibt das Feld gesperrt; sonst ersetzte das erste Wort still alle, die
     der Titel schon trägt. Wie Signatur und Lernmittel daneben schreibt er nur, wenn
     jemand etwas geändert hat.

     Eigene Datei, weil OrderStaging mit dem Feld über der 200-Zeilen-Marke lag. -->
<script>
	import { ladeTitelSchlagworte, setzeTitelSchlagworte } from '../../utils/schlagworte.js';
	import { erzeugeSchlagwortVorschlaege } from '../../utils/schlagwortVorschlaege.svelte.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import ChipFeld from '../ui/ChipFeld.svelte';
	import { untrack } from 'svelte';

	/**
	 * angebote: der Schlagwort-Vorschlag aus der DNB (POST /api/buecher/aus-isbn) — Wörter der
	 * eigenen Liste, die der DNB-Satz nennt. Angeboten zum Anklicken, nicht eingetragen: Der
	 * Satz nennt etwa „Deutsch" für die Sprache, und ein gleichnamiges Schlagwort für das Fach
	 * stünde sonst still am Titel.
	 * angeboteNeu: Normdatei-Wörter des Satzes, die die Liste noch nicht kennt (docs/OFFEN.md
	 * 4.25, entschieden am 30.09.2026: angeboten mit dem Zusatz „neu", nie vorbelegt).
	 * @type {{ titelId: string, angebote?: string[], angeboteNeu?: string[] }}
	 */
	let { titelId, angebote = [], angeboteNeu = [] } = $props();

	/** @type {string[]} */
	let schlagworte = $state([]);
	/** @type {string[] | null} */
	let beiStart = $state(null);
	let fehlen = $state(false);
	const vorschlaege = erzeugeSchlagwortVorschlaege();

	$effect(() => {
		let abgebrochen = false;
		vorschlaege.lade();
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
		vorschlaege={vorschlaege.liste}
		ontippen={vorschlaege.getippt}
		{angebote}
		angeboteEtikett="Vorschläge aus der DNB"
		{angeboteNeu}
		angeboteNeuEtikett="Neue Schlagworte aus der DNB"
		disabled={beiStart === null}
		hint={fehlen
			? 'Konnten nicht geladen werden — bitte später im Buchformular eintragen.'
			: undefined}
	/>
</div>
