<!-- @component SchadenMeldenDialog — Verlust oder Schaden an einem entliehenen Buch melden.

     Ebene „darueber": Der Dialog öffnet aus der Schülerakte heraus, die selbst ein Overlay
     ist; auf der Grundebene läge er unsichtbar dahinter. -->
<script>
	import Dialog from './Dialog.svelte';
	import Button from './components/ui/Button.svelte';
	import Feld from './components/ui/Feld.svelte';
	import Radio from './components/ui/Radio.svelte';
	import { apiFetch } from './apiFetch.js';

	// `book.ohneForderung` = der Entleiher ist ein Kollege. Gesetzt wird es beim Öffnen
	// (useLeserakte: oeffneSchaden), damit die Entscheidung an einer Stelle steht. Dann
	// entsteht keine Forderung: Der Bescheid ist ein Schreiben an Erziehungsberechtigte und
	// braucht Klasse und Anschrift, die von einer Lehrkraft nirgends stehen; gehaftet wird
	// gegenüber dem Dienstherrn und nur bei Vorsatz oder grober Fahrlässigkeit, und das stellt
	// die Schulleitung fest, nicht die Bücherei. Gebucht wird trotzdem, was den Bestand
	// angeht. Entschieden wird es am Server (repository/schaden_melden.go); hier steht nur,
	// was der Dialog zeigt und fragt.
	/** @type {{ ausleihe: any, onCancel: () => void, onSubmit: (grund: string, betrag: number, art: string) => void, isSubmitting?: boolean }} */
	let { ausleihe, onCancel, onSubmit, isSubmitting } = $props();
	const ohneForderung = $derived(!!ausleihe?.ohneForderung);

	let grund = $state('Verloren');
	// Startwert 0, kein fester Betrag: Der Server rechnet den Vorschlag am Buch und sagt dazu,
	// wie er entstanden ist. Bis seine Antwort da ist, steht hier lieber 0 als eine Zahl, die
	// gleich wegspringt.
	let schadensbetrag = $state(0);
	// Hat jemand den Betrag selbst eingetragen? Dann bleibt er stehen, auch wenn der Vorschlag
	// erst danach eintrifft — sonst entstünde die Forderung über den Vorschlag statt den Betrag.
	let vonHand = false;
	/** Der Satz unter dem Betragsfeld: Herleitung, Ladehinweis oder Fehlermeldung. */
	let herleitung = $state('');
	// Die Fallgruppe steht im Bescheid an die Eltern (welches Kästchen, ob Rückgabe
	// verlangt wird) — deshalb eine Wahl, kein Rückschluss aus dem Freitext.
	let art = $state('nicht_zurueckgegeben');

	// Der Vorschlag kommt vom Server, weil dort die Regeln liegen: die Staffel der
	// Arbeitshilfe für Lernmittel, der Neuwert ohne Abschlag für den Bücherei-Bestand
	// (internal/service/ersatzwert_vorschlag.go). Zwei Rechnungen an zwei Orten wären dieselbe
	// Geschichte wie bei den Fristen — und die Zahl landet in einer Forderung.
	//
	// Ohne Forderung (Kollege) wird nicht gefragt: Dann gibt es kein Betragsfeld.
	$effect(() => {
		const exemplarId = ausleihe?.id;
		if (!exemplarId || ohneForderung) return;

		let abgebrochen = false;
		vonHand = false;
		herleitung = 'Vorschlag wird berechnet …';

		apiFetch(`/api/buecher/exemplare/${exemplarId}/ersatzwert-vorschlag`)
			.then(async (res) => {
				if (!res.ok) throw new Error(String(res.status));
				return res.json();
			})
			.then((v) => {
				if (abgebrochen) return;
				if (!vonHand) schadensbetrag = v.betrag ?? 0;
				herleitung = v.herleitung ?? '';
			})
			.catch(() => {
				if (abgebrochen) return;
				// Kein stilles 0,00 €: Wer hier nichts liest, hielte den Startwert für
				// einen berechneten Vorschlag.
				herleitung = 'Vorschlag konnte nicht berechnet werden — bitte Betrag selbst eintragen.';
			});

		return () => {
			abgebrochen = true;
		};
	});

	function melde() {
		onSubmit(grund, ohneForderung ? 0 : schadensbetrag, art);
	}
</script>

{#if ausleihe}
	<Dialog open={true} onclose={onCancel} ebene="darueber" beschriftetDurch="schaden-titel">
		<div class="p-6">
			<h3 id="schaden-titel" class="text-xl font-bold text-on-surface mb-2">
				Verlust/Schaden melden
			</h3>
			<p class="text-sm text-on-surface-variant mb-4">
				Für <strong>{ausleihe.titel}</strong> ({ausleihe.barcode_id}).
				{#if ohneForderung}
					Die Ausleihe wird beendet und das Exemplar ausgesondert. Eine Forderung entsteht nicht:
					Ersatz von einer Lehrkraft zu verlangen ist Sache der Schulleitung, nicht der Bücherei.
				{:else}
					Die Ausleihe wird beendet und eine Forderung angelegt; sie steht danach unter „Gebühren
					&amp; Schäden" und im Mahnwesen unter „Schadensersatz". Der Bescheid an die Eltern ist der
					nächste, eigene Schritt.
				{/if}
			</p>

			<div class="space-y-4">
				<fieldset class="space-y-2">
					<legend class="text-sm font-semibold text-on-surface">Was ist passiert?</legend>
					<Radio
						bind:group={art}
						value="nicht_zurueckgegeben"
						etikett="Nicht zurückgegeben (verloren)"
					/>
					<Radio bind:group={art} value="beschaedigt" etikett="Beschädigt zurückgegeben" />
				</fieldset>
				<Feld
					id="damage-reason"
					etikett="Grund"
					bind:value={grund}
					placeholder="z.B. Wasserschaden, Verloren..."
				/>
				{#if !ohneForderung}
					<Feld
						id="damage-amount"
						etikett="Ersatzbetrag"
						type="number"
						step="0.01"
						min="0"
						hint={herleitung}
						bind:value={schadensbetrag}
						oninput={() => (vonHand = true)}
					>
						{#snippet nachlaufend()}€{/snippet}
					</Feld>
				{/if}
				<div class="flex gap-3 justify-end pt-4">
					<Button variant="ghost" onclick={onCancel} disabled={isSubmitting}>Abbrechen</Button>
					<Button
						variant="danger-solid"
						onclick={melde}
						disabled={isSubmitting || !grund.trim() || schadensbetrag < 0}
					>
						{isSubmitting ? 'Wird gemeldet...' : 'Melden'}
					</Button>
				</div>
			</div>
		</div>
	</Dialog>
{/if}
