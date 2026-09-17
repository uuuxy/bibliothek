<!-- @component DamageReportModal — Verlust oder Schaden an einem entliehenen Buch melden.

     Seit 07.09.2026 auf Modal.svelte (Register 05.09.: elf Overlays bauten ihr Markup
     selbst). Ebene „darueber": Der Dialog öffnet aus der Schülerakte heraus, die selbst
     ein Overlay ist — auf der Grundebene läge er unsichtbar dahinter. -->
<script>
	import Modal from './Modal.svelte';
	import Button from './components/ui/Button.svelte';
	import Feld from './components/ui/Feld.svelte';
	import Radio from './components/ui/Radio.svelte';
	import { apiFetch } from './apiFetch.js';

	// `book.ohneForderung` = der Entleiher ist ein Kollege. Gesetzt wird es beim Öffnen
	// (useStudentProfile: openDamageModal), damit die Entscheidung an EINER Stelle steht. Dann entsteht keine Forderung
	// (entschieden am 16.09.2026): Der Bescheid ist ein Schreiben an Erziehungsberechtigte
	// und braucht Klasse und Anschrift, die von einer Lehrkraft nirgends stehen; gehaftet
	// wird gegenüber dem Dienstherrn und nur bei Vorsatz oder grober Fahrlässigkeit — das
	// stellt die Schulleitung fest, nicht die Bücherei. Gebucht wird trotzdem, was den
	// Bestand angeht. Entschieden wird es am Server (repository/schaden_melden.go); hier
	// steht nur, was der Dialog zeigt und fragt.
	/** @type {{ book: any, onCancel: () => void, onSubmit: (grund: string, betrag: number, art: string) => void, isSubmitting?: boolean }} */
	let { book, onCancel, onSubmit, isSubmitting } = $props();
	const ohneForderung = $derived(!!book?.ohneForderung);

	let damageReason = $state('Verloren');
	// Startwert 0, nicht 15: Bis zum 17.09.2026 stand hier eine feste 15,00 € ohne jeden
	// Bezug zum Buch — das Protokoll des Medienzentrums vom 16.09.2026 nennt genau das
	// („fehlende Restwertberechnung z.Zt. Handeingabe"). Der Server rechnet den Vorschlag
	// und sagt dazu, WIE er entstanden ist; bis seine Antwort da ist, steht hier lieber 0
	// als eine Zahl, die gleich wegspringt.
	let damageAmount = $state(0);
	/** Der Satz unter dem Betragsfeld: Herleitung, Ladehinweis oder Fehlermeldung. */
	let herleitung = $state('');
	// Die Fallgruppe steht im Bescheid an die Eltern (welches Kästchen, ob Rückgabe
	// verlangt wird) — deshalb eine Wahl, kein Rückschluss aus dem Freitext.
	let art = $state('nicht_zurueckgegeben');

	// Der Vorschlag kommt vom Server, weil dort die Regeln liegen: die Staffel der
	// Arbeitshilfe für Lernmittel, der Neuwert ohne Abschlag für den Bücherei-Bestand
	// (api/ersatzwert_vorschlag_handler.go). Zwei Rechnungen an zwei Orten wären dieselbe
	// Geschichte wie bei den Fristen — und die Zahl landet in einer Forderung.
	//
	// Ohne Forderung (Kollege) wird nicht gefragt: Dann gibt es kein Betragsfeld.
	$effect(() => {
		const exemplarId = book?.id;
		if (!exemplarId || ohneForderung) return;

		let abgebrochen = false;
		herleitung = 'Vorschlag wird berechnet …';

		apiFetch(`/api/buecher/exemplare/${exemplarId}/ersatzwert-vorschlag`)
			.then(async (res) => {
				if (!res.ok) throw new Error(String(res.status));
				return res.json();
			})
			.then((v) => {
				if (abgebrochen) return;
				damageAmount = v.betrag ?? 0;
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

	function handleSubmit() {
		onSubmit(damageReason, ohneForderung ? 0 : damageAmount, art);
	}
</script>

{#if book}
	<Modal open={true} onclose={onCancel} ebene="darueber" beschriftetDurch="schaden-titel">
		<div class="p-6">
			<h3 id="schaden-titel" class="text-xl font-bold text-on-surface mb-2">
				Verlust/Schaden melden
			</h3>
			<p class="text-sm text-on-surface-variant mb-4">
				Für <strong>{book.titel}</strong> ({book.barcode_id}).
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
						label="Nicht zurückgegeben (verloren)"
					/>
					<Radio bind:group={art} value="beschaedigt" label="Beschädigt zurückgegeben" />
				</fieldset>
				<Feld
					id="damage-reason"
					label="Grund"
					bind:value={damageReason}
					placeholder="z.B. Wasserschaden, Verloren..."
				/>
				{#if !ohneForderung}
					<Feld
						id="damage-amount"
						label="Ersatzbetrag"
						type="number"
						step="0.01"
						min="0"
						hint={herleitung}
						bind:value={damageAmount}
					>
						{#snippet nachlaufend()}€{/snippet}
					</Feld>
				{/if}
				<div class="flex gap-3 justify-end pt-4">
					<Button variant="ghost" onclick={onCancel} disabled={isSubmitting}>Abbrechen</Button>
					<Button
						variant="danger-solid"
						onclick={handleSubmit}
						disabled={isSubmitting || !damageReason.trim() || damageAmount < 0}
					>
						{isSubmitting ? 'Wird gemeldet...' : 'Melden'}
					</Button>
				</div>
			</div>
		</div>
	</Modal>
{/if}
