<!-- @component ExemplarStandortDialog — den Standort markierter Exemplare ändern
     (docs/OFFEN.md 5.53), das Geschwister von ExemplarEigentumDialog. Die Form kommt aus
     ui/EingabeDialog wie in der Schlagwort-Pflege: ein Feld, ein Kästchen, zwei Aktionen.

     Das Feld ist nicht vorbelegt, weil die markierten Exemplare verschiedene Standorte tragen
     können. Vorgeschlagen werden die Standorte aus dem Bestand: Wer einen wählt, schreibt
     keinen zweiten Namen für dasselbe Regal. -->
<script>
	import EingabeDialog from './ui/EingabeDialog.svelte';
	import Feld from './ui/Feld.svelte';
	import Kaestchen from './ui/Kaestchen.svelte';
	import { apiPut } from '../apiFetch.js';
	import { scanSchutz } from '../scanErkennung.js';
	import { toastStore } from '../stores/toastStore.svelte.js';
	import { ladeStandorte } from '../utils/standorte.js';

	/**
	 * @type {{ open: boolean, exemplarIds: string[], onclose: () => void, onGeaendert: () => Promise<void> | void }}
	 */
	let { open, exemplarIds, onclose, onGeaendert } = $props();

	const eigen = $props.id();
	const listeId = `${eigen}-standorte`;

	let standort = $state('');
	let entfernen = $state(false);
	let sendet = $state(false);
	/** @type {import('../utils/standorte.js').StandortZahl[]} */
	let vorhandene = $state([]);

	$effect(() => {
		if (!open) return;
		standort = '';
		entfernen = false;
		let abgebrochen = false;
		ladeStandorte().then((liste) => {
			if (!abgebrochen) vorhandene = liste;
		});
		return () => {
			abgebrochen = true;
		};
	});

	const anzahl = $derived(exemplarIds.length);
	const neu = $derived(standort.trim());
	const gueltig = $derived(!sendet && (entfernen || neu !== ''));

	/** @param {number} n */
	const anzahlText = (n) => `${n} ${n === 1 ? 'Exemplar' : 'Exemplare'}`;

	async function speichern() {
		if (!gueltig) return;
		const weg = entfernen;
		sendet = true;
		try {
			const antwort = await apiPut('/api/exemplare/standort', {
				exemplar_ids: exemplarIds,
				standort: weg ? '' : neu
			});
			const n = antwort?.geaendert ?? 0;
			toastStore.addToast(
				n === 0
					? 'Der Standort war schon so eingetragen.'
					: `Standort ${weg ? 'entfernt' : 'geändert'}: ${anzahlText(n)}.`,
				'success'
			);
			await onGeaendert();
			onclose();
		} catch {
			/* apiFetch zeigt den Fehler-Toast */
		} finally {
			sendet = false;
		}
	}
</script>

<EingabeDialog
	{open}
	titel="Standort ändern"
	aktion={sendet ? 'Wird gespeichert …' : 'Standort ändern'}
	{gueltig}
	{onclose}
	onbestaetigen={speichern}
>
	<p class="text-sm text-on-surface-variant">Gilt für {anzahlText(anzahl)}.</p>
	<!-- Ein Scan in das offene Feld wäre sonst der Standort der markierten Exemplare. -->
	<div use:scanSchutz={() => {}}>
		<Feld
			label="Standort"
			bind:value={standort}
			list={listeId}
			autocomplete="off"
			disabled={entfernen}
		/>
	</div>
	<Kaestchen bind:checked={entfernen} label="Standort entfernen" />
	<datalist id={listeId}>
		{#each vorhandene as s (s.standort)}
			<option value={s.standort}>{anzahlText(s.anzahl)}</option>
		{/each}
	</datalist>
</EingabeDialog>
