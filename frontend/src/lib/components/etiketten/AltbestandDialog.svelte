<!-- @component AltbestandDialog — der Stichtag, ab dem der Altbestand als beklebt gilt.

     Warum ein Dialog und kein aufklappbarer Block mehr: Die Aktion ist unumkehrbar und
     trifft bei einem gewachsenen Bestand fünfstellig viele Exemplare. Sie stand bis zum
     04.09.2026 als <details> am FUSS der Liste — also hinter 300 Zeilen Scrollen. Genau
     dort war sie am schlechtesten erreichbar, obwohl sie bei einem Altbestand ohne
     Etikett-Vermerk die eigentliche Aufgabe der Seite ist: Von 30.674 offenen Etiketten
     sind fast alle Altbestand und keines davon zu drucken.

     M3 stellt unumkehrbare Aktionen hinter einen Dialog, der die Folge benennt, bevor er
     sie ausführt — hier die betroffene Anzahl, die vor dem Bestätigen geladen wird. -->
<script>
	import { apiGet, apiPost } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';
	import Modal from '../../Modal.svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';

	/** @type {{ open: boolean, onclose: () => void, onfertig: () => void }} */
	let { open, onclose, onfertig } = $props();

	let stichtag = $state('');
	let betroffen = $state(0);
	let arbeitet = $state(false);

	// Der Dialog bleibt gemountet, der Zustand überlebt also das Schließen: Ohne Reset
	// trägt der nächste Aufruf den Stichtag des vorigen mit — und zeigt eine Anzahl, die
	// zu einem Datum gehört, das niemand mehr im Kopf hat.
	$effect(() => {
		if (open) {
			stichtag = '';
			betroffen = 0;
		}
	});

	async function zaehle() {
		if (!stichtag) {
			betroffen = 0;
			return;
		}
		try {
			const daten = await apiGet(
				`/api/exemplare/etiketten-offen/anzahl?bis=${encodeURIComponent(stichtag)}`
			);
			betroffen = daten?.anzahl ?? 0;
		} catch {
			betroffen = 0;
		}
	}

	async function aufraeumen() {
		if (!stichtag || betroffen === 0 || arbeitet) return;
		arbeitet = true;
		try {
			const daten = await apiPost('/api/exemplare/etiketten-altbestand', { bis: stichtag });
			toastStore.addToast(`${daten?.markiert ?? 0} Exemplare als erledigt vermerkt.`, 'success');
			onfertig();
			onclose();
		} catch {
			toastStore.addToast('Der Altbestand konnte nicht vermerkt werden.', 'error');
		} finally {
			arbeitet = false;
		}
	}
</script>

<Modal {open} {onclose} size="lg">
	{#snippet header()}
		<h3 class="text-lg font-semibold text-on-surface">Altbestand aufräumen</h3>
	{/snippet}
	<div class="space-y-5">
		<p class="text-sm leading-relaxed text-on-surface-variant">
			Für Exemplare aus der Zeit vor dieser Funktion wurde nie vermerkt, ob ein Etikett gedruckt
			wurde — sie stehen deshalb alle in der Liste, auch die längst beklebten. Wähle den Tag, bis zu
			dem dein Bestand beklebt ist; alles bis dahin gilt danach als erledigt.
		</p>

		<Feld
			id="altbestand-stichtag"
			label="Beklebt bis"
			type="date"
			bind:value={stichtag}
			onchange={zaehle}
		/>

		{#if stichtag}
			<p
				class="rounded-xl bg-error-container px-4 py-3 text-sm text-on-error-container"
				role="status"
			>
				{#if betroffen === 0}
					Bis zu diesem Tag steht kein Exemplar mehr offen — es gäbe nichts zu vermerken.
				{:else}
					<strong class="font-semibold">{betroffen.toLocaleString('de-DE')} Exemplare</strong>
					werden als erledigt vermerkt. Das lässt sich nicht rückgängig machen.
				{/if}
			</p>
		{/if}

		<div class="flex justify-end gap-2 pt-1">
			<Button variant="ghost" size="lg" onclick={onclose}>Abbrechen</Button>
			<Button
				variant="danger-solid"
				size="lg"
				onclick={aufraeumen}
				disabled={!stichtag || betroffen === 0 || arbeitet}
			>
				{arbeitet ? 'Wird vermerkt …' : 'Als erledigt vermerken'}
			</Button>
		</div>
	</div>
</Modal>
