<!-- @component TitelZusammenfuehrenDialog — zwei Titel, ein Buch, etwa einer aus einer Liste
     ohne ISBN und einer aus dem Bestands-Import: suchen, antippen, eine Zeile Vorschau,
     bestätigen. Der Titel mit ISBN bleibt; was wandert und welche Felder kommen, rechnet der
     Server (repository/titel_zusammenfuehren.go). Material 3, Dialogs: höchstens zwei Aktionen,
     die bestätigende erst nach der Wahl bedienbar; Lists: „A list item can include supporting
     text below the label text." -->
<script>
	import { Merge, CircleAlert } from '@lucide/svelte';
	import { apiClient, leseApiFehler } from '../../../../lib/apiFetch.js';
	import { snackbars } from '../../../../lib/stores/snackbars.svelte.js';
	import Dialog from '../../../../lib/components/ui/Dialog.svelte';
	import Button from '../../../../lib/components/ui/Button.svelte';
	import Suchfeld from '../../../../lib/components/ui/Suchfeld.svelte';
	import { erzeugeDialogAnfrage } from '../../../../lib/components/ui/dialogAnfrage.svelte.js';
	import TitelZusammenfuehrenVorschau from './TitelZusammenfuehrenVorschau.svelte';
	import { erzeugeAuflagenSuche } from './auflagenSuche.svelte.js';
	import { erzeugeZusammenfuehrenVorschau, exemplare } from './zusammenfuehrenVorschau.svelte.js';

	/** @type {{ open: boolean, titel: any, onZusammengefuehrt: (titelId: string) => void }} */
	let { open = $bindable(false), titel, onZusammengefuehrt } = $props();

	/** @type {any | null} */
	let anderer = $state(null);
	let laeuft = $state(false);
	let fehler = $state('');
	// Angeboten wird jeder Titel außer diesem, auch einer ohne Exemplar.
	const s = erzeugeAuflagenSuche(
		() => [titel.id],
		() => true
	);
	const v = erzeugeZusammenfuehrenVorschau();
	const anfrage = erzeugeDialogAnfrage(() => open);
	// Der Titel mit ISBN bleibt: Er trägt meist die vollständigen Angaben, der andere füllt nur
	// Lücken. Hat der andere keine ISBN, bleibt dieser, der offene. Es zählt die gespeicherte ISBN
	// (geladen): Eine nur eingetippte verwirft „Verwerfen und weiter".
	const andererBleibt = $derived(Boolean(anderer?.isbn) && !(titel.geladen ?? titel).isbn);
	const ziel = $derived(andererBleibt ? anderer?.id : titel.id);
	const quelle = $derived(andererBleibt ? titel.id : anderer?.id);

	// Zurückgesetzt wird beim Öffnen wie beim Schließen, sonst stünde beim nächsten Titel der
	// vorige Treffer da (dieselbe Regel wie in den anderen Dialogen dieser Art).
	$effect(() => {
		if (open) leeren();
		return leeren;
	});
	// Jeder angetippte Titel fragt die Vorschau neu.
	$effect(() => {
		if (anderer) v.lade(ziel, quelle);
	});

	function leeren() {
		s.zuruecksetzen();
		v.leeren();
		anderer = null;
		fehler = '';
	}
	// Wer weitertippt, sucht einen anderen Titel; die Wahl gilt dann nicht mehr.
	function tippen() {
		anderer = null;
		v.leeren();
		s.tippen();
	}
	const zeigeFehler = (/** @type {string} */ text) => (fehler = text);

	async function zusammenfuehren() {
		if (!v.vorschau || laeuft) return;
		laeuft = true;
		fehler = '';
		const lauf = anfrage.beginne();
		try {
			const res = await apiClient.post(`/api/buecher/titel/${ziel}/zusammenfuehren`, {
				quelle_id: quelle
			});
			if (!res.ok) {
				lauf.fehler(await leseApiFehler(res), zeigeFehler);
				return;
			}
			const erg = await res.json();
			snackbars.addToast(
				`Zusammengeführt: „${erg.titel}“ hat jetzt ${exemplare(erg.exemplare_danach)}.`,
				'success'
			);
			if (lauf.nochOffen()) open = false;
			onZusammengefuehrt(erg.titel_id);
		} catch {
			lauf.fehler('Netzwerkfehler beim Zusammenführen.', zeigeFehler);
		} finally {
			laeuft = false;
		}
	}
</script>

<Dialog {open} onclose={() => (open = false)} size="2xl" beschriftetDurch="titel-zf-titel">
	{#snippet header()}
		<h3 id="titel-zf-titel" class="flex items-center gap-2 text-lg font-bold text-on-surface">
			<Merge class="h-5 w-5 text-primary" aria-hidden="true" />
			Mit anderem Titel zusammenführen
		</h3>
	{/snippet}
	<div class="space-y-5 px-6 py-6 text-on-surface">
		<p class="text-sm text-on-surface-variant">Für ein Buch, das zweimal im Katalog steht.</p>

		<div class="space-y-2">
			<Suchfeld
				bind:wert={s.suche}
				platzhalter="Titel oder ISBN …"
				etikett="Anderen Titel suchen"
				oninput={tippen}
				autofokus
			/>
			{#if s.fehler}
				<p role="alert" class="text-xs font-bold text-error">{s.fehler}</p>
			{:else if s.treffer.length > 0}
				<ul
					class="max-h-72 divide-y divide-outline-variant/40 overflow-y-auto rounded-xl border border-outline-variant"
				>
					{#each s.treffer as t (t.id)}
						<li>
							<button
								type="button"
								aria-pressed={anderer?.id === t.id}
								class="w-full cursor-pointer px-3 py-2 text-left transition-colors hover:bg-surface-container aria-pressed:bg-secondary-container aria-pressed:text-on-secondary-container"
								onclick={() => (anderer = t)}
							>
								<span class="block truncate text-sm">{t.title}</span>
								<span class="block truncate text-xs text-on-surface-variant"
									>{t.isbn || 'ohne ISBN'} · {exemplare(t.gesamt)}</span
								>
							</button>
						</li>
					{/each}
				</ul>
			{:else if s.suche.trim().length >= 2}
				<p class="text-xs text-on-surface-variant">Kein anderer Titel gefunden.</p>
			{/if}
		</div>

		{#if anderer}
			<TitelZusammenfuehrenVorschau vorschau={v.vorschau} laedt={v.laedt} fehler={v.fehler} />
		{/if}

		{#if fehler}
			<div
				role="alert"
				class="flex items-start gap-2 rounded-xl bg-error-container p-3 text-on-error-container"
			>
				<CircleAlert class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
				<p class="text-xs font-bold leading-tight">{fehler}</p>
			</div>
		{/if}
	</div>

	<div
		class="flex justify-end gap-3 border-t border-outline-variant bg-surface-container-low px-6 py-4"
	>
		<Button variant="secondary" onclick={() => (open = false)}>Abbrechen</Button>
		<Button variant="danger-solid" onclick={zusammenfuehren} disabled={!v.vorschau || laeuft}>
			{laeuft ? 'Wird zusammengeführt …' : 'Zusammenführen'}
		</Button>
	</div>
</Dialog>
