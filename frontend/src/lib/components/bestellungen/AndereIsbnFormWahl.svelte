<!-- @component AndereIsbnFormWahl — die Frage, wenn die Bestelltür dieselbe ISBN in der anderen
     Länge im Katalog findet (ISBN-10 ↔ ISBN-13; docs/OFFEN.md 4.18 Stufe 4). POST
     /api/buecher/aus-isbn hat dann nichts angelegt und antwortet mit andere_form. Vorgeschlagen
     statt übernommen: Am Testserver führt die Rechnung von einer ISBN-10 mit falschem
     Prüfzeichen auf die ISBN-13 eines anderen Buchs.

     Zwei Listenzeilen statt eines Rückfrage-Dialogs: Dort zählt Escape als „nein", und „nein"
     hieße hier „Neu anlegen" — ein zweiter Titel ohne Entscheidung. Hier geschieht ohne Klick
     nichts. Material 3, Lists: „Use lists for communicating or selecting discrete items"; eine
     Zeile trägt Label und Supporting text, vorn das Cover wie in der Trefferliste der
     Titelsuche. Genutzt in der Titelsuche (OrderSearch) und in „Neue Auflage bestellen"
     (NeueAuflageDialog). -->
<script>
	import { Plus } from '@lucide/svelte';
	import BuchCover from '../ui/BuchCover.svelte';

	/**
	 * isbn: die eingegebene oder gescannte ISBN; vorschlag: der Titel unter der anderen Form
	 * (andere_form der Antwort); neuTitel: der Titel, den „Neu anlegen" holt, wenn er schon
	 * bekannt ist (der DNB-Treffer der Titelsuche).
	 * @type {{ isbn: string, vorschlag: any, neuTitel?: string, laeuft?: boolean, onnehmen: () => void, onneu: () => void }}
	 */
	let { isbn, vorschlag, neuTitel = '', laeuft = false, onnehmen, onneu } = $props();

	const form = $derived(
		String(vorschlag?.isbn ?? '').length === 10 ? 'zehnstelliger' : 'dreizehnstelliger'
	);
	const zeile = 'flex w-full items-center gap-3 px-3.5 py-2.5 text-left text-base';
	// Titel und ISBN je eine Zeile (M3 Lists: „Limit supporting text to one to three lines"): In
	// der schmalen Spalte der Bestellung schnitte eine gemeinsame Zeile die ISBN ab — und nach
	// ihr wird hier entschieden.
	const stuetze = 'block truncate text-sm text-on-surface-variant';
</script>

<div class="space-y-1.5">
	<p class="text-sm text-on-surface-variant">
		Im Katalog steht diese ISBN in {form} Form. Ist es dasselbe Buch?
	</p>
	<ul class="rounded-lg border border-outline-variant py-1">
		<li>
			<button type="button" class={zeile} onclick={onnehmen} disabled={laeuft}>
				<BuchCover
					coverUrl={vorschlag.cover_url}
					isbn={vorschlag.isbn}
					titel={vorschlag.titel}
					dekorativ
				/>
				<span class="min-w-0 flex-1">
					<span class="block text-on-surface">Diesen Titel nehmen</span>
					<span class={stuetze}>„{vorschlag.titel}"</span>
					<span class={stuetze}>
						{[vorschlag.isbn, vorschlag.verlag].filter(Boolean).join(' · ')}
					</span>
				</span>
			</button>
		</li>
		<li>
			<button type="button" class={zeile} onclick={onneu} disabled={laeuft}>
				<span
					class="flex h-10 aspect-3/4 shrink-0 items-center justify-center text-on-surface-variant"
					aria-hidden="true"
				>
					<Plus class="h-5 w-5" />
				</span>
				<span class="min-w-0 flex-1">
					<span class="block text-on-surface">Neu anlegen</span>
					{#if neuTitel}<span class={stuetze}>„{neuTitel}"</span>{/if}
					<span class={stuetze}>
						{neuTitel ? `${isbn} aus der DNB` : `${isbn} als eigener Titel aus der DNB`}
					</span>
				</span>
			</button>
		</li>
	</ul>
</div>
