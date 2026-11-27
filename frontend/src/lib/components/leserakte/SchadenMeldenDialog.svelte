<!-- @component SchadenMeldenDialog — Verlust oder Schaden an einem entliehenen Buch melden.

     Ebene „darueber": Der Dialog öffnet aus der Schülerakte heraus, die selbst ein Overlay
     ist; auf der Grundebene läge er unsichtbar dahinter. -->
<script>
	import { tick } from 'svelte';
	import Dialog from '../ui/Dialog.svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import Radio from '../ui/Radio.svelte';
	import Pflichthinweis from '../ui/Pflichthinweis.svelte';
	import { apiFetch, leseApiFehler } from '../../apiFetch.js';
	import { fehlertext } from '../../utils/fehlertext.js';

	// `book.ohneForderung` = der Entleiher ist ein Kollege. Gesetzt wird es beim Öffnen
	// (useLeserakte: oeffneSchaden), damit die Entscheidung an einer Stelle steht. Dann
	// entsteht keine Forderung: Der Bescheid ist ein Schreiben an Erziehungsberechtigte und
	// braucht Klasse und Anschrift, die von einer Lehrkraft nirgends stehen; gehaftet wird
	// gegenüber dem Dienstherrn und nur bei Vorsatz oder grober Fahrlässigkeit, und das stellt
	// die Schulleitung fest, nicht die Bücherei. Gebucht wird trotzdem, was den Bestand
	// angeht. Entschieden wird es am Server (internal/repository/schaden_melden.go); hier steht nur,
	// was der Dialog zeigt und fragt.
	/** @type {{ ausleihe: any, onCancel: () => void, onSubmit: (grund: string, betrag: number, art: string) => void, isSubmitting?: boolean }} */
	let { ausleihe, onCancel, onSubmit, isSubmitting } = $props();
	const ohneForderung = $derived(!!ausleihe?.ohneForderung);
	// Ein Gerät bleibt in der Geräteverwaltung: beschädigt als defekt, verloren als verloren
	// (internal/repository/schaden_geraet.go). Die Forderung entsteht wie beim Buch; einen Vorschlag
	// gibt es nicht, weil keine Regel einen Ersatzwert für Geräte nennt.
	const geraet = $derived(!!ausleihe?.ist_geraet);

	let grund = $state('Verloren');
	// Startwert 0, kein fester Betrag: Der Server rechnet den Vorschlag am Buch und sagt dazu,
	// wie er entstanden ist. Bis seine Antwort da ist, steht hier lieber 0 als eine Zahl, die
	// gleich wegspringt.
	let schadensbetrag = $state(0);
	// Leer oder keine Zahl hieße beim Server 0 €, also „ohne Forderung“, ohne dass es jemand wählte.
	const betragFehlt = $derived(
		!ohneForderung && !(Number.isFinite(schadensbetrag) && schadensbetrag >= 0)
	);
	// Hat jemand den Betrag selbst eingetragen? Dann bleibt er stehen, auch wenn der Vorschlag
	// erst danach eintrifft — sonst entstünde die Forderung über den Vorschlag statt den Betrag.
	let vonHand = false;
	/** Der Satz unter dem Betragsfeld: Herleitung, Ladehinweis oder Fehlermeldung. */
	let herleitung = $state('');
	// Die Fallgruppe steht im Bescheid an die Eltern (welches Kästchen, ob Rückgabe
	// verlangt wird) — deshalb eine Wahl, kein Rückschluss aus dem Freitext.
	let art = $state('nicht_zurueckgegeben');
	const geraetZustand = $derived(art === 'beschaedigt' ? 'defekt' : 'verloren');
	// Der Knopf bleibt bedienbar: Fehlt Grund oder Betrag, nennt der Klick das Feld, wie in
	// BuchFormular (ARCHITEKTUR, „Felder und Eingabe“).
	let verlangt = $state(false);
	const grundFehlt = $derived(verlangt && !grund.trim());
	const betragFalsch = $derived(verlangt && betragFehlt);

	// Der Vorschlag kommt vom Server, weil dort die Regeln liegen: die Staffel der
	// Arbeitshilfe für Lernmittel, der Neuwert ohne Abschlag für den Bücherei-Bestand
	// (internal/service/ersatzwert_vorschlag.go). Zwei Rechnungen an zwei Orten wären dieselbe
	// Geschichte wie bei den Fristen — und die Zahl landet in einer Forderung.
	//
	// Ohne Forderung (Kollege) wird nicht gefragt: Dann gibt es kein Betragsfeld.
	$effect(() => {
		const exemplarId = ausleihe?.id;
		if (!exemplarId || ohneForderung || geraet) return;

		let abgebrochen = false;
		vonHand = false;
		herleitung = 'Vorschlag wird berechnet …';

		apiFetch(`/api/buecher/exemplare/${exemplarId}/ersatzwert-vorschlag`)
			.then(async (res) => {
				if (!res.ok) throw new Error(await leseApiFehler(res));
				return res.json();
			})
			.then((v) => {
				if (abgebrochen) return;
				if (!vonHand) schadensbetrag = v.betrag ?? 0;
				herleitung = v.herleitung ?? '';
			})
			.catch((e) => {
				if (abgebrochen) return;
				// Kein stilles 0,00 €: Wer hier nichts liest, hielte den Startwert für
				// einen berechneten Vorschlag. Der Grund steht dabei, etwa „Keine Verbindung zum Server".
				herleitung = `Vorschlag konnte nicht berechnet werden: ${fehlertext(e).replace(/\.$/, '')}. Bitte Betrag selbst eintragen.`;
			});

		return () => {
			abgebrochen = true;
		};
	});

	async function melde() {
		verlangt = true;
		if (!grund.trim() || betragFehlt) {
			await tick();
			document.getElementById(!grund.trim() ? 'damage-reason' : 'damage-amount')?.focus();
			return;
		}
		onSubmit(grund, ohneForderung ? 0 : schadensbetrag, art);
	}
</script>

{#if ausleihe}
	<Dialog open={true} onclose={onCancel} ebene="darueber" beschriftetDurch="schaden-titel">
		<div class="p-6">
			<h3 id="schaden-titel" class="dialog-titel mb-2">Verlust oder Schaden melden</h3>
			<p class="text-sm text-on-surface-variant mb-4">
				Für <strong>{ausleihe.titel}</strong>, {geraet ? 'Gerät' : 'Exemplar'}
				{ausleihe.barcode_id}.
				{#if geraet && ohneForderung}
					Die Ausleihe wird beendet, das Gerät steht danach als {geraetZustand} in der Geräteverwaltung
					und ist nicht ausleihbar. Eine Forderung entsteht nicht: Ersatz von einer Lehrkraft zu verlangen
					ist Sache der Schulleitung, nicht der Bücherei.
				{:else if geraet}
					Die Ausleihe wird beendet, das Gerät steht danach als {geraetZustand} in der Geräteverwaltung
					und ist nicht ausleihbar. Die Forderung steht danach unter „Gebühren &amp; Schäden" und wird
					dort bezahlt; einen Bescheid gibt es für Geräte nicht.
				{:else if ohneForderung}
					Die Ausleihe wird beendet und das Exemplar ausgesondert. Eine Forderung entsteht nicht:
					Ersatz von einer Lehrkraft zu verlangen ist Sache der Schulleitung, nicht der Bücherei.
				{:else}
					Die Ausleihe wird beendet und eine Forderung angelegt; sie steht danach unter „Gebühren
					&amp; Schäden" und im Mahnwesen unter „Schadensersatz". Der Bescheid an die Eltern ist der
					nächste, eigene Schritt.
				{/if}
			</p>

			<div class="space-y-4">
				<Pflichthinweis />
				<fieldset class="space-y-2">
					<legend class="text-title-small text-on-surface">Was ist passiert?</legend>
					<Radio
						bind:group={art}
						value="nicht_zurueckgegeben"
						etikett="Verloren, nicht zurückgegeben"
					/>
					<Radio bind:group={art} value="beschaedigt" etikett="Beschädigt zurückgegeben" />
				</fieldset>
				<Feld
					id="damage-reason"
					etikett="Grund"
					aria-required="true"
					bind:value={grund}
					placeholder="z. B. Wasserschaden, Verloren …"
					ungueltig={grundFehlt}
					hint={grundFehlt ? 'Bitte den Grund eintragen' : ''}
				/>
				{#if !ohneForderung}
					<Feld
						id="damage-amount"
						etikett="Ersatzbetrag"
						type="number"
						step="0.01"
						min="0"
						aria-required="true"
						ungueltig={betragFalsch}
						hint={betragFalsch
							? 'Bitte den Ersatzbetrag eintragen'
							: geraet
								? 'Für Geräte gibt es keinen Vorschlag'
								: herleitung}
						bind:value={schadensbetrag}
						oninput={() => (vonHand = true)}
					>
						{#snippet nachlaufend()}€{/snippet}
					</Feld>
				{/if}
				<div class="flex gap-3 justify-end pt-4">
					<Button variant="ghost" onclick={onCancel}>Abbrechen</Button>
					<Button variant="danger-solid" onclick={melde} disabled={isSubmitting}>
						{isSubmitting ? 'Wird gemeldet …' : 'Melden'}
					</Button>
				</div>
			</div>
		</div>
	</Dialog>
{/if}
