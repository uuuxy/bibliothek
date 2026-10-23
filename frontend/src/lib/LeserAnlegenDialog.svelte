<!-- @component LeserAnlegenDialog — „Neuen Leser anlegen".

     Fragt zuerst nach der Art: An ihr hängt, welche Angaben Pflicht sind. Ein Schüler
     braucht Klasse und Geburtsdatum, denn der LUSD-Import erkennt ihn nur daran wieder; ein
     Kollege hat beides nicht und bekommt statt Stammdaten den Hinweis, dass hier kein
     Zugang entsteht.

     Die Regeln stehen doppelt: hier als verständliche Meldung vor dem Absenden, im
     Backend als Bedingung (pruefeLeserAngaben). Das Backend ist die Wahrheit; diese
     Seite erspart nur den Umweg über eine Fehlermeldung vom Server. -->
<script>
	import Dialog from './Dialog.svelte';
	import { apiClient, leseApiFehler } from './apiFetch.js';
	import Button from './components/ui/Button.svelte';
	import SchuelerEingabeFelder from './components/leser/SchuelerEingabeFelder.svelte';
	import LeserArtWahl from './components/leser/LeserArtWahl.svelte';
	import KollegiumEingabeFelder from './components/leser/KollegiumEingabeFelder.svelte';
	import { leserArtText, istKollegium, artMitKonto } from './leserArt.js';
	import { TriangleAlert } from '@lucide/svelte';
	import { erzeugeDialogAnfrage } from './components/ui/dialogAnfrage.svelte.js';

	let { open = false, klassen = [], klassenFehler = false, onclose, onsuccess } = $props();

	let art = $state('schueler');
	let newVorname = $state('');
	let newNachname = $state('');
	let newKlasse = $state('');
	let newBarcode = $state('');
	let newGeburtsdatum = $state('');
	let newEmail = $state('');
	let fehler = $state('');
	let duplicateConflict = $state('');
	let speichert = $state(false);

	const kollege = $derived(istKollegium({ art }));
	// Praktikum und Fachbereich bekommen keinen Zugang zu „Mein Portal" und deshalb keine
	// Schul-E-Mail, dieselbe Regel wie im Server (leserart.HatKonto).
	const mitKonto = $derived(artMitKonto(art));

	// Formular zurücksetzen, sobald der Dialog aufgeht — samt Art: Wer zuletzt eine
	// Lehrkraft angelegt hat, legt beim nächsten Mal nicht ungewollt die zweite an.
	$effect(() => {
		if (open) {
			art = 'schueler';
			newVorname = '';
			newNachname = '';
			newKlasse = '';
			newBarcode = '';
			newGeburtsdatum = '';
			newEmail = '';
			fehler = '';
			duplicateConflict = '';
		}
	});

	/** Die Pflichtangaben — an die Art gepaart, wie im Backend. @returns {string} leer = in Ordnung */
	function fehlendeAngabe() {
		if (!newVorname.trim() || !newNachname.trim())
			return 'Vorname und Nachname sind Pflichtfelder.';
		if (kollege) {
			// Die Schuladresse ist bei einer Lehrkraft Pflicht, und zwar nicht als
			// Kontaktangabe: Aus ihr entsteht das Anmeldekonto, und weil sie eindeutig ist,
			// findet die spätere Selbstanmeldung über „Mein Portal" diesen Eintrag wieder
			// — statt einen zweiten anzulegen.
			if (mitKonto && !newEmail.trim())
				return 'Die Schul-E-Mail-Adresse fehlt. Ohne sie steht die Person doppelt in der Leserdatei, sobald sie sich über „Mein Portal“ selbst anmeldet.';
			return '';
		}
		if (!newKlasse.trim()) return 'Klasse ist ein Pflichtfeld.';
		if (!newGeburtsdatum.trim())
			return 'Geburtsdatum fehlt. Ohne Geburtsdatum kann der LUSD-Import diesen Schüler später nicht wiedererkennen — er würde doppelt angelegt.';
		return '';
	}

	const anfrage = erzeugeDialogAnfrage(() => open);

	async function legeAn() {
		fehler = fehlendeAngabe();
		duplicateConflict = '';
		if (fehler) return;

		const name = `${newVorname.trim()} ${newNachname.trim()}`;
		const artText = leserArtText(art);
		speichert = true;
		const lauf = anfrage.beginne();
		try {
			const res = await apiClient.post('/api/schueler', {
				art,
				vorname: newVorname.trim(),
				nachname: newNachname.trim(),
				klasse: kollege ? '' : newKlasse.trim(),
				barcode_id: newBarcode.trim(),
				geburtsdatum: kollege ? null : newGeburtsdatum.trim(),
				// Nur wo ein Zugang dazugehört: Ein Schüler, ein Praktikum und ein Fachbereich
				// bekommen kein Konto, und der Server weist eine Adresse dort ausdrücklich ab.
				email: mitKonto ? newEmail.trim() : ''
			});
			if (res.ok) {
				lauf.erfolg(`${name} ist angelegt.`);
				onsuccess?.();
				return;
			}
			// Der Server erklärt den Doppeleintrag, seine Meldung nennt den Fall
			// (Namensgleichheit beim Kollegium, Name + Geburtsdatum beim Schüler): Ein fester
			// Satz träfe nur eine der Arten.
			const doppelt = res.status === 409;
			const meldung = await leseApiFehler(
				res,
				doppelt ? 'Diese Person gibt es bereits.' : `Fehler beim Anlegen (${artText}).`
			);
			if (doppelt) lauf.fehler(meldung, (t) => (duplicateConflict = t));
			else lauf.fehler(meldung, (t) => (fehler = t));
		} catch (err) {
			lauf.fehler('Netzwerkfehler beim Anlegen.', (t) => (fehler = t));
			console.error(err);
		} finally {
			speichert = false;
		}
	}
</script>

<Dialog {open} onclose={() => onclose?.()} size="md">
	{#snippet header()}
		<h3 class="text-base font-bold text-on-surface">Neuen Leser anlegen</h3>
	{/snippet}
	<div class="p-6 space-y-4">
		<LeserArtWahl bind:art disabled={speichert} />

		{#if duplicateConflict}
			<div
				class="p-4 bg-warning-container rounded-xl flex items-start gap-3 text-sm font-semibold text-on-warning-container"
			>
				<TriangleAlert class="h-5 w-5 shrink-0 mt-0.5" aria-hidden="true" />
				<p>{duplicateConflict}</p>
			</div>
		{/if}

		{#if fehler}
			<div class="p-3 bg-error-container rounded-xl text-xs font-semibold text-on-error-container">
				{fehler}
			</div>
		{/if}

		{#if kollege}
			<KollegiumEingabeFelder
				{art}
				bind:vorname={newVorname}
				bind:nachname={newNachname}
				bind:barcode={newBarcode}
				bind:email={newEmail}
			/>
		{:else}
			<SchuelerEingabeFelder
				bind:vorname={newVorname}
				bind:nachname={newNachname}
				bind:geburtsdatum={newGeburtsdatum}
				bind:klasse={newKlasse}
				bind:barcode={newBarcode}
				{klassen}
				{klassenFehler}
			/>
		{/if}

		<div class="flex justify-end gap-3 pt-2 border-t border-outline-variant">
			<Button variant="secondary" onclick={() => onclose?.()}>Abbrechen</Button>
			<Button onclick={legeAn} disabled={speichert}>
				{speichert ? 'Speichern...' : 'Speichern'}
			</Button>
		</div>
	</div>
</Dialog>
