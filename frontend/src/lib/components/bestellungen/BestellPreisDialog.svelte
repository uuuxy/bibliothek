<!-- @component Den Preis einer Bestellposition ändern, wie ihn die Rechnung des Händlers nennt.

     Der Preis beim Bestellen ist manchmal veraltet, und der Händler rechnet mit einem Rabatt oder
     einem neuen Preis ab. Die Rechnung kommt auf Papier; hier wird der Preis nachgetragen, auch
     lange nach der Bestellung. Das Programm zieht den Einkaufspreis der Exemplare aus dieser
     Bestellung und die Summe der Bestellung mit, und die Änderung steht im Protokoll. -->
<script>
	import { untrack } from 'svelte';
	import EingabeDialog from '../ui/EingabeDialog.svelte';
	import Feld from '../ui/Feld.svelte';
	import { apiPut } from '../../apiFetch.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';

	/**
	 * @type {{
	 *   bestellungId: string,
	 *   position: any,
	 *   onclose: () => void,
	 *   onAktualisieren: () => Promise<unknown> | void
	 * }}
	 * position — die Position aus der Bestellung (id, titel_name, menge, einzelpreis); null = zu.
	 */
	let { bestellungId, position, onclose, onAktualisieren } = $props();

	let preis = $state(/** @type {string | number} */ (''));
	let sendet = $state(false);

	// Beim Öffnen mit dem gespeicherten Preis füllen; ein Neuladen der Bestellung während der
	// Eingabe nähme sonst den getippten Preis mit.
	$effect(() => {
		if (position) preis = untrack(() => Number(position.einzelpreis));
	});

	const zahl = $derived(Number(preis));
	const gueltig = $derived(
		!sendet &&
			String(preis).trim() !== '' &&
			Number.isFinite(zahl) &&
			zahl >= 0 &&
			zahl <= 99999.99 &&
			zahl !== Number(position?.einzelpreis)
	);

	async function speichern() {
		if (!gueltig || !position) return;
		sendet = true;
		try {
			await apiPut(`/api/bestellungen/${bestellungId}/positionen/${position.id}/einzelpreis`, {
				einzelpreis: Math.round(zahl * 100) / 100
			});
			snackbars.addToast('Preis geändert.', 'success');
			await onAktualisieren();
			onclose();
		} catch {
			/* apiFetch zeigt den Fehler-Toast */
		} finally {
			sendet = false;
		}
	}
</script>

<EingabeDialog
	open={position !== null}
	titel="Preis ändern"
	aktion="Preis ändern"
	{gueltig}
	{onclose}
	onbestaetigen={speichern}
>
	<Feld
		id="bestellung-einzelpreis"
		etikett="Einzelpreis in €"
		type="number"
		step="0.01"
		min="0"
		bind:value={preis}
		hint="Gilt für {position?.menge ?? 0} Exemplare von „{position?.titel_name ??
			''}“ aus dieser Bestellung."
	/>
</EingabeDialog>
