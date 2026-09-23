<script>
	/**
	 * @component SchlagwortPflegeDialog
	 * Die drei Eingaben der Schlagwort-Pflege in einem Dialog (ui/EingabeDialog): umbenennen,
	 * zusammenführen, Verweis anlegen. Welche, sagt `auftrag`; null = geschlossen.
	 *
	 * Das Ziel beim Zusammenführen wird getippt und aus den vorhandenen Wörtern vorgeschlagen
	 * (datalist wie im ChipFeld am Titel); die Aktion bleibt gesperrt, bis die Eingabe ein
	 * vorhandenes anderes Wort trifft — ein neues Wort anzulegen ist Umbenennen, nicht
	 * Zusammenführen. Fehler des Servers (409 „gibt es schon", Regel verletzt) zeigt apiFetch als
	 * Meldung mit seinem Satz; der Dialog bleibt dann offen.
	 *
	 * @prop {{ art: 'umbenennen' | 'zusammenfuehren' | 'verweis', zeile: any } | null} auftrag
	 * @prop {any[]} zeilen - alle geladenen Schlagworte, für Vorschläge und Ziel.
	 * @prop {() => void} onclose
	 * @prop {() => Promise<void>} onfertig - lädt die Liste neu.
	 */
	import { apiPost, apiPut } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import EingabeDialog from '../ui/EingabeDialog.svelte';
	import Feld from '../ui/Feld.svelte';

	/** @type {{ auftrag: { art: 'umbenennen' | 'zusammenfuehren' | 'verweis', zeile: any } | null, zeilen: any[], onclose: () => void, onfertig: () => Promise<void> }} */
	let { auftrag, zeilen, onclose, onfertig } = $props();

	const eigen = $props.id();
	const listeId = `${eigen}-woerter`;
	let laeuft = $state(false);

	// Beim Öffnen vorbelegt (Umbenennen beginnt mit der alten Schreibweise), danach die
	// Eingabe; ein neuer Auftrag setzt sie zurück.
	let eingabe = $derived(auftrag?.art === 'umbenennen' ? auftrag.zeile.wort : '');

	const wort = $derived(auftrag?.zeile.wort ?? '');
	const neu = $derived(eingabe.trim());
	const ziel = $derived(
		auftrag?.art === 'zusammenfuehren'
			? zeilen.find((z) => z.id !== auftrag.zeile.id && z.wort.toLowerCase() === neu.toLowerCase())
			: undefined
	);
	const gueltig = $derived(
		!laeuft &&
			neu !== '' &&
			(auftrag?.art === 'umbenennen' ? neu !== wort : auftrag?.art === 'verweis' || Boolean(ziel))
	);

	const texte = $derived(
		{
			umbenennen: {
				titel: `„${wort}“ umbenennen`,
				aktion: 'Umbenennen',
				label: 'Neue Schreibweise'
			},
			zusammenfuehren: {
				titel: `„${wort}“ zusammenführen`,
				aktion: 'Zusammenführen',
				label: 'Mit Schlagwort'
			},
			verweis: { titel: `Verweis auf „${wort}“`, aktion: 'Verweis anlegen', label: 'Schreibweise' }
		}[auftrag?.art ?? 'umbenennen']
	);

	const hinweis = $derived.by(() => {
		if (!auftrag) return '';
		const n = auftrag.zeile.titel;
		if (auftrag.art === 'umbenennen')
			return n > 0 ? `Alle ${n} Titel tragen danach die neue Schreibweise.` : '';
		if (auftrag.art === 'zusammenfuehren')
			return (
				(n > 0 ? `Die ${n} Titel bekommen das gewählte Wort; ` : '') +
				`„${wort}“ bleibt als Verweis darauf.`
			);
		return `Wer diese Schreibweise am Titel einträgt, bekommt „${wort}“. Trägt sie schon Titel, werden sie umgestellt.`;
	});

	async function ausfuehren() {
		if (!auftrag || !gueltig) return;
		const { art, zeile } = auftrag;
		laeuft = true;
		try {
			if (art === 'umbenennen') {
				const r = await apiPut(`/api/schlagworte/${zeile.id}/wort`, { wort: neu });
				toastStore.addToast(`Umbenannt in „${r?.wort ?? neu}“.`, 'success');
			} else if (art === 'zusammenfuehren' && ziel) {
				const r = await apiPost(`/api/schlagworte/${zeile.id}/zusammenfuehren`, {
					ziel_id: ziel.id
				});
				toastStore.addToast(
					`Zusammengeführt: ${r?.titel ?? 0} Titel tragen jetzt „${ziel.wort}“.`,
					'success'
				);
			} else {
				await apiPost('/api/schlagworte/verweise', { wort: neu, ziel_id: zeile.id });
				toastStore.addToast(`„${neu}“ verweist jetzt auf „${zeile.wort}“.`, 'success');
			}
		} catch {
			return; // Meldung kam bereits aus apiFetch; der Dialog bleibt offen.
		} finally {
			laeuft = false;
		}
		onclose();
		await onfertig();
	}
</script>

<EingabeDialog
	open={auftrag !== null}
	titel={texte.titel}
	aktion={texte.aktion}
	{gueltig}
	{onclose}
	onbestaetigen={ausfuehren}
>
	<Feld
		label={texte.label}
		bind:value={eingabe}
		hint={hinweis}
		list={auftrag?.art === 'zusammenfuehren' ? listeId : undefined}
		autocomplete="off"
	/>
	{#if auftrag?.art === 'zusammenfuehren'}
		<datalist id={listeId}>
			{#each zeilen.filter((z) => z.id !== auftrag?.zeile.id && !z.verweis_auf_id) as z (z.id)}
				<option value={z.wort}>{z.titel} Titel</option>
			{/each}
		</datalist>
	{/if}
</EingabeDialog>
