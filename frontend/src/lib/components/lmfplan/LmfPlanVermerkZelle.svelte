<!-- @component LmfPlanVermerkZelle — die Besonderheiten einer Planer-Zeile. Sie stehen als Text
     da wie die Klasse daneben, ein Klick öffnet an ihrer Stelle das Feld: Sechzig leere Felder
     wären kein Plan mehr, sondern ein Formular. Enter oder ein Klick daneben übernimmt, Escape
     stellt den Text von vorher wieder her. Eine Zeile ohne Klasse braucht den Text; fehlt er,
     steht das Feld offen und rot. -->
<script>
	import { tick } from 'svelte';
	import Feld from '../ui/Feld.svelte';

	/** @type {{ vermerk: string, i: number, ohneKlasse: boolean }} */
	let { vermerk = $bindable(), i, ohneKlasse } = $props();

	let offen = $state(false);
	let vorher = '';
	/** @type {HTMLButtonElement | undefined} */
	let knopf = $state();
	const fehlt = $derived(ohneKlasse && !vermerk.trim());

	// Auch das Feld einer Pflichtzeile öffnet sich so: Sonst verschwände es beim ersten
	// Buchstaben, weil der Text die Pflicht dann schon erfüllt.
	function beginnen() {
		if (offen) return;
		vorher = vermerk;
		offen = true;
	}

	async function oeffnen() {
		beginnen();
		await tick();
		document.getElementById(`lmf-zeile-vermerk-${i}`)?.focus();
	}

	// Die Taste beendet das Schreiben, der Fokus geht zurück auf die Zelle. Fehlt der Pflichttext,
	// bleibt das Feld stehen und im Fokus, also auch offen: Sonst verschwände es beim nächsten
	// Buchstaben.
	async function beenden() {
		if (fehlt) return;
		offen = false;
		await tick();
		knopf?.focus();
	}

	/** @param {KeyboardEvent} e */
	function taste(e) {
		if (e.key === 'Enter') {
			e.preventDefault();
			beenden();
		} else if (e.key === 'Escape') {
			vermerk = vorher;
			beenden();
		}
	}
</script>

<td>
	{#if offen || fehlt}
		<Feld
			id="lmf-zeile-vermerk-{i}"
			aria-label="Besonderheiten Zeile {i + 1}"
			bind:value={vermerk}
			placeholder={ohneKlasse ? 'Pflicht ohne Klasse' : ''}
			ungueltig={fehlt}
			feld="w-full max-w-maske"
			onfocus={beginnen}
			onblur={() => (offen = false)}
			onkeydown={taste}
		/>
	{:else}
		<!-- Maße wie das Feld (36 px, Rand, 12 px Einzug): Beim Öffnen springt weder die Zeile
		     noch der Text. -->
		<button
			bind:this={knopf}
			type="button"
			class="flex h-9 w-full max-w-maske cursor-pointer items-center rounded-xl border border-transparent px-3 text-left text-on-surface hover:bg-surface-container"
			title={vermerk.trim() || undefined}
			onclick={oeffnen}
		>
			<span class="sr-only">Besonderheiten Zeile {i + 1}:</span>
			<!-- Eine Zeile wie im Feld; der ganze Text steht im Tooltip und nach dem Klick im Feld. -->
			{#if vermerk.trim()}
				<span class="min-w-0 truncate">{vermerk}</span>
			{:else}
				<span
					class="text-on-surface-variant opacity-0 group-hover:opacity-100 group-focus-within:opacity-100"
				>
					Besonderheit eintragen
				</span>
			{/if}
		</button>
	{/if}
</td>
