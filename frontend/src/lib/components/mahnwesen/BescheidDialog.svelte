<!-- @component BescheidDialog — der eine Schritt, in dem ein Mensch entscheidet.

     Alles am Bescheid ist Vorgabe oder Einstellung: Nummer, Frist, Konto, Anschrift,
     Texte. Nur zwei Dinge entscheidet die Schule, und genau die zeigt dieser Dialog:
     WELCHE Bücher auf den Brief kommen und mit WELCHEM Betrag. Die Staffel schlägt vor
     und nennt ihre Herleitung; wer sie überschreibt, tut es im Ermessen der Schule.

     Seit Stufe 2 (15.09.2026) stehen auch die überfälligen Bücher ohne Forderung in der
     Liste: Mit dem Brief bucht der Server ihren Verlust (Ausleihe endet, Exemplar gilt
     als verloren). Es braucht keine Verlustmeldung je Buch mehr, und der Betrag wird
     nur einmal gefragt. Kommt ein Buch zurück, storniert der Theke-Scan die Forderung.

     Bauform nach M3: ein Basis-Dialog (kein Vollbild — das ist die Form für kleine
     Bildschirme), Bestätigung mit konkretem Verb („Bescheid erstellen"), keine Schritte.
     Die Referenznummer entsteht erst beim Erstellen und wird nie zweimal vergeben; der
     Dialog sagt das, damit niemand den Knopf zum Ausprobieren drückt. Der Zustand lebt
     in bescheidFormular.svelte.js. -->
<script>
	import Modal from '../../Modal.svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import BescheidPositionen from './BescheidPositionen.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import { BescheidFormular } from './bescheidFormular.svelte.js';
	import { toastStore } from '../../stores/toastStore.svelte.js';

	/**
	 * @type {{
	 *   schuelerId: string,
	 *   onclose: () => void,
	 *   onErstellt: () => Promise<void> | void,
	 *   ebene?: 'basis' | 'darueber'
	 * }}
	 */
	// ebene „darueber": aus der Schülerakte heraus, die selbst ein Overlay ist.
	let { schuelerId, onclose, onErstellt, ebene = 'basis' } = $props();

	// Je Schüler ein frisches Formular; der Vorschlag wird einmal geholt. Der Anfangswert
	// der ID genügt: Der Aufrufer baut den Dialog je Schüler neu ({#key} im Mahnwesen,
	// {#if} in der Akte).
	// svelte-ignore state_referenced_locally
	const formular = new BescheidFormular(schuelerId);
	formular.laden();

	/** @param {number} n */
	const euro = (n) =>
		Number(n).toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) +
		' €';

	async function erstellen() {
		const bescheid = await formular.erstellen();
		if (!bescheid) return;
		toastStore.addToast(`Bescheid ${bescheid.referenznummer ?? ''} erstellt.`, 'success', {
			label: 'Brief öffnen',
			onClick: () => window.open(`/api/bescheide/${bescheid.id}/pdf`, '_blank')
		});
		await onErstellt();
		onclose();
	}
</script>

<Modal open={true} {onclose} size="2xl" {ebene} beschriftetDurch="bescheid-titel">
	<div class="space-y-5 p-6">
		<div>
			<h2 id="bescheid-titel" class="text-lg font-bold text-on-surface">
				Schadensersatz-Bescheid{formular.vorschlag?.schueler_name
					? ` für ${formular.vorschlag.schueler_name}`
					: ''}
			</h2>
			<p class="mt-1 text-sm text-on-surface-variant">
				Die Beträge sind Vorschläge nach der Staffel der Schule und im Ermessen änderbar. Die
				Referenznummer wird beim Erstellen vergeben und nie zweimal. Überfällige Bücher werden mit
				dem Brief als Verlust gebucht; kommt eines zurück, storniert die Theke die Forderung.
			</p>
		</div>

		{#if formular.laedt}
			<div class="flex items-center gap-2 py-8 text-sm text-on-surface-variant">
				<Ladekreis size="sm" /> Überfällige Bücher und offene Forderungen werden geladen …
			</div>
		{:else if formular.fehlt.length > 0}
			<!-- Kein stumm gesperrter Knopf: Der Dialog sagt, WAS fehlt und wo es steht. -->
			<div
				class="rounded-xl border border-error bg-error-container p-4 text-sm text-on-error-container"
			>
				<p class="font-semibold">Es fehlen Angaben für den Bescheid.</p>
				<ul class="mt-1 list-inside list-disc">
					{#each formular.fehlt as f (f)}<li>{f}</li>{/each}
				</ul>
				<p class="mt-2">Einzutragen in den Einstellungen unter „Schadensersatz".</p>
			</div>
		{:else if formular.positionen.length === 0}
			<p class="py-8 text-sm text-on-surface-variant">
				Für dieses Kind gibt es weder ein überfälliges Buch noch eine offene Forderung, die noch auf
				keinem Bescheid steht.
			</p>
		{:else}
			<BescheidPositionen
				positionen={formular.positionen}
				bind:gewaehlt={formular.gewaehlt}
				bind:betraege={formular.betraege}
			/>

			<div
				class="flex flex-wrap items-end justify-between gap-4 border-t border-outline-variant pt-4"
			>
				<Feld
					id="bescheid-frist"
					label="Zahlungsfrist (steht als Datum im Brief)"
					type="date"
					bind:value={formular.frist}
					feld="w-44"
				/>
				<div class="text-right">
					<div class="text-xs font-semibold text-on-surface-variant">Gesamtbetrag</div>
					<div class="text-2xl font-bold text-on-surface tabular-nums">{euro(formular.summe)}</div>
				</div>
			</div>
		{/if}

		<div class="flex justify-end gap-2 pt-2">
			<Button variant="ghost" onclick={onclose} disabled={formular.sendet}>Abbrechen</Button>
			<Button onclick={erstellen} disabled={!formular.bereit}>
				{#if formular.sendet}<Ladekreis size="sm" farbe="aktuell" />{/if}
				Bescheid erstellen
			</Button>
		</div>
	</div>
</Modal>
