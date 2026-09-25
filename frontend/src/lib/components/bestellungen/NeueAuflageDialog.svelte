<!-- @component NeueAuflageDialog — „Neue Auflage bestellen" aus einer Zeile des Bestellbedarfs
     (docs/OFFEN.md 4.18, Stufe 4). Entschieden am 23.09.2026: „Der Vorschlag entsteht
     automatisch beim Nachbestellen, das Ja gibt der Besteller." Der Vorschlag geht von der
     Zeile aus, also von einem konkreten Buch: Die ISBN der neuen Auflage holt den Titel über
     dieselbe Tür wie die Titelsuche (POST /api/buecher/aus-isbn — aus dem Katalog oder neu aus
     der DNB), der Dialog zeigt ihn, und erst „Zuordnen und bestellen" fasst zusammen
     (POST …/neue-auflage, create_orders, repository/auflagen.go) und legt ihn in den Warenkorb.

     Material 3, Dialogs: höchstens zwei Aktionen, die bestätigende gesperrt, bis es etwas zu
     bestätigen gibt; die abbrechende nie. -->
<script>
	import { AlertCircle } from '@lucide/svelte';
	import { apiClient, extractApiError } from '../../apiFetch.js';
	import Modal from '../../Modal.svelte';
	import Button from '../ui/Button.svelte';
	import Suchfeld from '../ui/Suchfeld.svelte';
	import { toastStore } from '../../stores/toastStore.svelte.js';

	/** @type {{ zeile: any | null, onschliessen: () => void, onbestellt: (titel: any) => void }} */
	let { zeile, onschliessen, onbestellt } = $props();

	let isbn = $state('');
	/** Der Titel zur eingegebenen ISBN (Antwort von aus-isbn) — der Vorschlag. @type {any | null} */
	let gefunden = $state(null);
	let laeuft = $state(false);
	let fehler = $state('');

	const offen = $derived(zeile !== null);
	// Schon Teil dieses Buchs? Dann gibt es nichts zuzuordnen — die Zeile selbst oder eine
	// Auflage aus ihrer Aufschlüsselung.
	const schonDabei = $derived(
		gefunden !== null &&
			(gefunden.titel_id === zeile?.id ||
				(zeile?.auflagen ?? []).some((/** @type {any} */ a) => a.id === gefunden.titel_id))
	);

	$effect(() => {
		if (offen) leeren();
		return leeren;
	});

	function leeren() {
		isbn = '';
		gefunden = null;
		laeuft = false;
		fehler = '';
	}

	/** @param {string} [code] */
	async function suchen(code) {
		const wert = (code ?? isbn).trim();
		if (!wert || laeuft) return;
		isbn = wert;
		laeuft = true;
		fehler = '';
		gefunden = null;
		try {
			const res = await apiClient.post('/api/buecher/aus-isbn', { isbn: wert });
			if (!res.ok) {
				fehler = await extractApiError(res);
				return;
			}
			gefunden = await res.json();
		} catch {
			fehler = 'Netzwerkfehler — die ISBN-Abfrage hat den Server nicht erreicht.';
		} finally {
			laeuft = false;
		}
	}

	async function zuordnen() {
		if (!gefunden || schonDabei || laeuft || !zeile) return;
		laeuft = true;
		fehler = '';
		try {
			// Die neue Auflage eines Schulbuchs ist ein Schulbuch. Ein eben aus der DNB angelegter
			// Titel ist es noch nicht, und zusammengefasst werden nur Lernmittel.
			if (!gefunden.ist_lernmittel) {
				const lm = await apiClient.put(`/api/buecher/titel/${gefunden.titel_id}/lernmittel`, {
					ist_lernmittel: true
				});
				if (!lm.ok) {
					fehler = await extractApiError(lm);
					return;
				}
			}
			const res = await apiClient.post(`/api/buecher/titel/${zeile.id}/neue-auflage`, {
				titel_id: gefunden.titel_id
			});
			if (!res.ok) {
				fehler = await extractApiError(res);
				return;
			}
			toastStore.addToast(
				`„${gefunden.titel}" als neue Auflage zugeordnet und in den Warenkorb gelegt.`,
				'success'
			);
			onbestellt({ ...gefunden, id: gefunden.titel_id, ist_lernmittel: true });
		} catch {
			fehler = 'Netzwerkfehler — die Zuordnung hat den Server nicht erreicht.';
		} finally {
			laeuft = false;
		}
	}
</script>

<Modal open={offen} onclose={onschliessen} size="2xl" beschriftetDurch="neue-auflage-titel">
	{#snippet header()}
		<h3 id="neue-auflage-titel" class="text-lg font-bold text-on-surface">
			Neue Auflage bestellen
		</h3>
	{/snippet}
	<div class="space-y-5 px-6 py-6 text-on-surface">
		<p class="text-sm text-on-surface-variant">
			Für „{zeile?.titel}" gibt es eine neue Auflage? Die ISBN der neuen Auflage eingeben oder
			scannen. Sie wird als Auflage desselben Buchs zugeordnet und kommt in den Warenkorb.
		</p>

		<form
			onsubmit={(e) => {
				e.preventDefault();
				suchen();
			}}
		>
			<Suchfeld
				bind:wert={isbn}
				platzhalter="ISBN eingeben oder scannen …"
				etikett="ISBN der neuen Auflage"
				kamera
				onscan={(code) => suchen(code)}
				autofokus
			/>
		</form>

		{#if gefunden}
			<!-- Der Titel vorn: Hier wird geprüft, ob die ISBN das richtige Buch getroffen hat. Auflage
			     und Jahr kennt die Antwort von aus-isbn nicht. -->
			<div class="rounded-lg border border-outline-variant p-3">
				<p class="truncate text-sm text-on-surface">{gefunden.titel}</p>
				<p class="truncate text-xs text-on-surface-variant">
					{[gefunden.isbn, gefunden.verlag].filter(Boolean).join(' · ')}
				</p>
			</div>
			{#if schonDabei}
				<p class="text-sm text-on-surface-variant">
					Dieser Titel gehört schon zu diesem Buch — er lässt sich mit dem Plus der Zeile bestellen.
				</p>
			{:else if !gefunden.ist_lernmittel}
				<p class="text-sm text-on-surface-variant">
					Er wird wie die bisherige Auflage als Lernmittel geführt.
				</p>
			{/if}
		{/if}

		{#if fehler}
			<div
				role="alert"
				class="flex items-start gap-2 rounded-xl bg-error-container p-3 text-on-error-container"
			>
				<AlertCircle class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
				<p class="text-xs font-bold leading-tight">{fehler}</p>
			</div>
		{/if}
	</div>

	<div
		class="flex justify-end gap-3 border-t border-outline-variant bg-surface-container-low px-6 py-4"
	>
		<Button variant="secondary" onclick={onschliessen}>Abbrechen</Button>
		{#if gefunden}
			<Button onclick={zuordnen} disabled={schonDabei || laeuft}>
				{laeuft ? 'Wird zugeordnet …' : 'Zuordnen und bestellen'}
			</Button>
		{:else}
			<Button onclick={() => suchen()} disabled={!isbn.trim() || laeuft}>
				{laeuft ? 'Wird gesucht …' : 'Suchen'}
			</Button>
		{/if}
	</div>
</Modal>
