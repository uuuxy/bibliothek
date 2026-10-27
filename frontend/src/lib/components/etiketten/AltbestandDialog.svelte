<!-- @component AltbestandDialog — der Stichtag, ab dem der Altbestand als beklebt gilt.

     Ein Dialog und kein aufklappbarer Block am Fuß der Liste: Die Aktion ist unumkehrbar
     und trifft bei einem gewachsenen Bestand fünfstellig viele Exemplare, und bei einem
     Altbestand ohne Etikett-Vermerk ist sie die eigentliche Aufgabe der Seite. Dann sind
     fast alle offenen Etiketten Altbestand und keines davon zu drucken.

     M3 stellt unumkehrbare Aktionen hinter einen Dialog, der die Folge benennt, bevor er
     sie ausführt — hier die betroffene Anzahl, die vor dem Bestätigen geladen wird. -->
<script>
	import { apiGet, apiPost } from '../../apiFetch.js';
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import Dialog from '../ui/Dialog.svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import { fehlertext } from '../../utils/fehlertext.js';

	/** @type {{ open: boolean, onclose: () => void, onfertig: () => void }} */
	let { open, onclose, onfertig } = $props();

	let stichtag = $state('');
	let betroffen = $state(0);
	// Der Grund, wenn die Zählung scheiterte. Ohne ihn hieße die 0 „nichts mehr offen".
	let zaehlfehler = $state('');
	let zaehlt = $state(false);
	let arbeitet = $state(false);
	// Die jüngste Zählung: Wer den Tag schnell wechselt, sähe sonst die Zahl eines anderen Tages,
	// und der Knopf vermerkte bis zum neuen.
	let zaehlLauf = 0;

	// Der Dialog bleibt gemountet, der Zustand überlebt also das Schließen: Ohne Reset
	// trägt der nächste Aufruf den Stichtag des vorigen mit — und zeigt eine Anzahl, die
	// zu einem Datum gehört, das niemand mehr im Kopf hat.
	$effect(() => {
		if (open) {
			zaehlLauf++;
			stichtag = '';
			betroffen = 0;
			zaehlfehler = '';
			zaehlt = false;
		}
	});

	async function zaehle() {
		const meine = ++zaehlLauf;
		betroffen = 0;
		zaehlfehler = '';
		zaehlt = stichtag !== '';
		if (!zaehlt) return;
		try {
			const daten = await apiGet(
				`/api/exemplare/etiketten-offen/anzahl?bis=${encodeURIComponent(stichtag)}`
			);
			if (meine === zaehlLauf) betroffen = daten?.anzahl ?? 0;
		} catch (e) {
			if (meine === zaehlLauf) zaehlfehler = fehlertext(e);
		} finally {
			if (meine === zaehlLauf) zaehlt = false;
		}
	}

	async function aufraeumen() {
		if (!stichtag || betroffen === 0 || arbeitet) return;
		arbeitet = true;
		try {
			const daten = await apiPost('/api/exemplare/etiketten-altbestand', { bis: stichtag });
			snackbars.addToast(`${daten?.markiert ?? 0} Exemplare als erledigt vermerkt.`, 'success');
			onfertig();
			onclose();
		} catch {
			/* apiPost meldet jeden Fehler selbst, mit dem Grund; der Dialog bleibt offen */
		} finally {
			arbeitet = false;
		}
	}
</script>

<Dialog {open} {onclose} size="lg">
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
			etikett="Beklebt bis"
			type="date"
			bind:value={stichtag}
			onchange={zaehle}
		/>

		{#if stichtag}
			<p
				class="rounded-xl bg-error-container px-4 py-3 text-sm text-on-error-container"
				role="status"
			>
				{#if zaehlt}
					Wird gezählt …
				{:else if zaehlfehler}
					Nicht gezählt: {zaehlfehler}
				{:else if betroffen === 0}
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
</Dialog>
