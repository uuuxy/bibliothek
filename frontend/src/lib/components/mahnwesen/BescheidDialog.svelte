<!-- @component BescheidDialog — der eine Schritt, in dem ein Mensch entscheidet.

     Alles am Bescheid ist Vorgabe oder Einstellung: Nummer, Frist, Konto, Anschrift,
     Texte. Nur zwei Dinge entscheidet die Schule, und genau die zeigt dieser Dialog:
     WELCHE Bücher auf den Brief kommen und mit WELCHEM Betrag. Die Staffel schlägt vor
     und nennt ihre Herleitung; wer sie überschreibt, tut es im Ermessen der Schule.

     Bauform nach M3: ein Basis-Dialog (kein Vollbild — das ist die Form für kleine
     Bildschirme), Bestätigung mit konkretem Verb („Bescheid erstellen"), keine Schritte.
     Die Referenznummer entsteht erst beim Erstellen und wird nie zweimal vergeben; der
     Dialog sagt das, damit niemand den Knopf zum Ausprobieren drückt. -->
<script>
	import Modal from '../../Modal.svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import BescheidPositionen from './BescheidPositionen.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { apiGet, apiPost } from '../../apiFetch.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';

	/**
	 * @type {{
	 *   schuelerId: string,
	 *   onclose: () => void,
	 *   onErstellt: () => Promise<void> | void
	 * }}
	 */
	let { schuelerId, onclose, onErstellt } = $props();

	/** @type {any} */
	let vorschlag = $state(null);
	let laedt = $state(true);
	let sendet = $state(false);
	/** @type {Record<string, boolean>} */
	let gewaehlt = $state({});
	/** @type {Record<string, number>} */
	let betraege = $state({});
	let frist = $state('');

	// Der Vorschlag wird EINMAL geholt; danach gehört das Formular dem Menschen.
	$effect(() => {
		laden();
	});

	async function laden() {
		laedt = true;
		try {
			const daten = await apiGet(`/api/schueler/${schuelerId}/bescheid-vorschlag`);
			vorschlag = daten;
			frist = daten?.frist_bis ?? '';
			for (const p of daten?.positionen ?? []) {
				gewaehlt[p.schadensfall_id] = !!p.ist_lernmittel;
				betraege[p.schadensfall_id] = p.betrag;
			}
		} catch {
			vorschlag = null;
		} finally {
			laedt = false;
		}
	}

	const positionen = $derived(vorschlag?.positionen ?? []);
	const ausgewaehlt = $derived(
		positionen.filter((/** @type {any} */ p) => gewaehlt[p.schadensfall_id])
	);
	const summe = $derived(
		ausgewaehlt.reduce(
			(/** @type {number} */ s, /** @type {any} */ p) =>
				s + (Number(betraege[p.schadensfall_id]) || 0),
			0
		)
	);
	// Der Brief ist der des Landes (Wortlaut und Konto der Lernmittelfreiheit). Die Rechnung
	// der Schülerbücherei ist noch nicht gebaut (Konzept 4.7, Etappe 3); ihre Forderungen
	// stehen deshalb gesperrt in der Liste, und der Server weist alles andere ab.
	const mittel = 'land';
	const fehlt = $derived(vorschlag?.fehlende_angaben ?? []);
	const bereit = $derived(ausgewaehlt.length > 0 && frist !== '' && fehlt.length === 0 && !sendet);

	/** @param {number} n */
	const euro = (n) =>
		Number(n).toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) +
		' €';

	async function erstellen() {
		if (!bereit) return;
		sendet = true;
		try {
			const bescheid = await apiPost(`/api/schueler/${schuelerId}/bescheide`, {
				mittel,
				frist_bis: frist,
				positionen: ausgewaehlt.map((/** @type {any} */ p) => ({
					schadensfall_id: p.schadensfall_id,
					betrag: Number(betraege[p.schadensfall_id]) || 0
				}))
			});
			toastStore.addToast(`Bescheid ${bescheid?.referenznummer ?? ''} erstellt.`, 'success', {
				label: 'Brief öffnen',
				onClick: () => window.open(`/api/bescheide/${bescheid.id}/pdf`, '_blank')
			});
			await onErstellt();
			onclose();
		} catch {
			/* apiFetch zeigt den Fehler-Toast */
		} finally {
			sendet = false;
		}
	}
</script>

<Modal open={true} {onclose} size="2xl" beschriftetDurch="bescheid-titel">
	<div class="space-y-5 p-6">
		<div>
			<h2 id="bescheid-titel" class="text-lg font-bold text-on-surface">
				Schadensersatz-Bescheid{vorschlag?.schueler_name ? ` für ${vorschlag.schueler_name}` : ''}
			</h2>
			<p class="mt-1 text-sm text-on-surface-variant">
				Die Beträge sind Vorschläge nach der Staffel der Schule und im Ermessen änderbar. Die
				Referenznummer wird beim Erstellen vergeben und nie zweimal.
			</p>
		</div>

		{#if laedt}
			<div class="flex items-center gap-2 py-8 text-sm text-on-surface-variant">
				<Ladekreis size="sm" /> Offene Forderungen werden geladen …
			</div>
		{:else if fehlt.length > 0}
			<!-- Kein stumm gesperrter Knopf: Der Dialog sagt, WAS fehlt und wo es steht. -->
			<div
				class="rounded-xl border border-error bg-error-container p-4 text-sm text-on-error-container"
			>
				<p class="font-semibold">Es fehlen Angaben für den Bescheid.</p>
				<ul class="mt-1 list-inside list-disc">
					{#each fehlt as f (f)}<li>{f}</li>{/each}
				</ul>
				<p class="mt-2">Einzutragen in den Einstellungen unter „Schadensersatz".</p>
			</div>
		{:else if positionen.length === 0}
			<p class="py-8 text-sm text-on-surface-variant">
				Für dieses Kind ist keine offene Forderung erfasst, die noch auf keinem Bescheid steht. Ein
				Bescheid entsteht aus einer Forderung — die legt „Schaden melden" bei der Rückgabe an.
			</p>
		{:else}
			<BescheidPositionen {positionen} bind:gewaehlt bind:betraege />

			<div
				class="flex flex-wrap items-end justify-between gap-4 border-t border-outline-variant pt-4"
			>
				<Feld
					id="bescheid-frist"
					label="Zahlungsfrist (steht als Datum im Brief)"
					type="date"
					bind:value={frist}
					feld="w-44"
				/>
				<div class="text-right">
					<div class="text-xs font-semibold text-on-surface-variant">Gesamtbetrag</div>
					<div class="text-2xl font-bold text-on-surface tabular-nums">{euro(summe)}</div>
				</div>
			</div>
		{/if}

		<div class="flex justify-end gap-2 pt-2">
			<Button variant="ghost" onclick={onclose} disabled={sendet}>Abbrechen</Button>
			<Button onclick={erstellen} disabled={!bereit}>
				{#if sendet}<Ladekreis size="sm" farbe="aktuell" />{/if}
				Bescheid erstellen
			</Button>
		</div>
	</div>
</Modal>
