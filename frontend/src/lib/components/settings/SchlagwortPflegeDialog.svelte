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
	 * Umbenennen und Zusammenführen fragen mit einem Kästchen, ob die alte Schreibweise als
	 * Verweis stehen bleibt (docs/OFFEN.md 4.20, entschieden am 23.09.2026), vorbelegt mit ja:
	 * Das hält eine Maske richtig, die dabei offen war, und wer das alte Wort gewohnt ist,
	 * landet weiter richtig. Abgewählt verschwindet es ganz, wie in Littera. Wann das Kästchen
	 * erscheint und was der Hinweis sagt, steht in schlagwortPflege.js.
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
	import Kaestchen from '../ui/Kaestchen.svelte';
	import { verweisWahl, dialogHinweis } from './schlagwortPflege.js';

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
	// Jeder neue Auftrag beginnt mit „behalten"; die Wahl gilt, solange der Dialog offen ist.
	let behalten = $derived(auftrag !== null);
	const wahl = $derived(auftrag ? verweisWahl(auftrag.art, auftrag.zeile) : false);
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

	const hinweis = $derived(auftrag ? dialogHinweis(auftrag.art, auftrag.zeile, neu) : '');

	async function ausfuehren() {
		if (!auftrag || !gueltig) return;
		const { art, zeile } = auftrag;
		laeuft = true;
		try {
			if (art === 'umbenennen') {
				const r = await apiPut(`/api/schlagworte/${zeile.id}/wort`, {
					wort: neu,
					alte_als_verweis: wahl && behalten
				});
				const rest = r?.verweise ? ` „${zeile.wort}“ bleibt als Verweis.` : '';
				toastStore.addToast(`Umbenannt in „${r?.wort ?? neu}“.${rest}`, 'success');
			} else if (art === 'zusammenfuehren' && ziel) {
				const r = await apiPost(`/api/schlagworte/${zeile.id}/zusammenfuehren`, {
					ziel_id: ziel.id,
					alte_als_verweis: behalten
				});
				toastStore.addToast(
					`Zusammengeführt: ${r?.titel ?? 0} Titel ${r?.titel === 1 ? 'trägt' : 'tragen'} jetzt „${ziel.wort}“.`,
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
	{#if wahl}
		<Kaestchen bind:checked={behalten} label="„{wort}“ als Verweis behalten" />
	{/if}
	{#if auftrag?.art === 'zusammenfuehren'}
		<datalist id={listeId}>
			{#each zeilen.filter((z) => z.id !== auftrag?.zeile.id && !z.verweis_auf_id) as z (z.id)}
				<option value={z.wort}>{z.titel} Titel</option>
			{/each}
		</datalist>
	{/if}
</EingabeDialog>
