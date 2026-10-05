<script>
	/**
	 * @component PortalTrefferkarte
	 * Ein Suchtreffer im Kollegiums-Portal: Cover, Titelangaben, Verfügbarkeit, die
	 * Warteschlange und zwei Aktionen — „Klassensatz reservieren" und „Problem melden". Je
	 * Karte ist höchstens eines der beiden Formulare offen; ihr Zustand gehört dem Aufrufer.
	 *
	 * @prop {any} book
	 * @prop {any} form - Reservierung dieses Titels (klassensatzReservierung.svelte.js).
	 * @prop {import('./problemMeldung.svelte.js').MeldeFormular} meldung - Meldung zu diesem Titel.
	 * @prop {{ klasse: string, anzahl: number, erstellt_am: string }[]} warteschlange
	 * @prop {() => void} ontoggle
	 * @prop {() => void} onsenden
	 * @prop {() => void} onmelden - öffnet oder schließt „Problem melden".
	 * @prop {() => Promise<boolean>} onmeldungsenden - true, wenn die Meldung angenommen ist.
	 * @prop {() => void} onmeldungabbrechen
	 */
	import { tick } from 'svelte';
	import { BookOpen } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import KlassensatzFormular from './KlassensatzFormular.svelte';
	import ProblemFormular from './ProblemFormular.svelte';
	import { coverSrc } from '../../utils/coverSrc.js';
	import { bestandSatz } from '../../utils/format.js';

	/** @type {{ book: any, form: any, meldung: import('./problemMeldung.svelte.js').MeldeFormular, warteschlange: { klasse: string, anzahl: number, erstellt_am: string }[], ontoggle: () => void, onsenden: () => void, onmelden: () => void, onmeldungsenden: () => Promise<boolean>, onmeldungabbrechen: () => void }} */
	let {
		book,
		form,
		meldung,
		warteschlange,
		ontoggle,
		onsenden,
		onmelden,
		onmeldungsenden,
		onmeldungabbrechen
	} = $props();

	/** @type {HTMLButtonElement | undefined} */
	let meldeKnopf = $state();

	// Schließt das Formular, geht der Fokus zurück auf den Knopf, der es geöffnet hat.
	async function schliesseMeldung() {
		onmeldungabbrechen();
		await tick();
		meldeKnopf?.focus();
	}
	async function sendeMeldung() {
		if (!(await onmeldungsenden())) return;
		await tick();
		meldeKnopf?.focus();
	}

	const bild = $derived(coverSrc(book.cover_url, book.isbn));

	// Reservieren bucht nichts — das OPAC-Abzeichen sinkt erst, wenn die Bibliothek den
	// Satz tatsächlich ausleiht. „60 von 60 verfügbar" und darunter „40 reserviert für
	// 8a" standen deshalb nebeneinander, und die Lehrkraft musste selbst rechnen. Die
	// Vormerkungen werden hier abgezogen: eine Zahl, die sagt, ob es JETZT reicht.
	const vorgemerkt = $derived(warteschlange.reduce((sum, o) => sum + (o.anzahl ?? 0), 0));
	const rechnerischFrei = $derived(
		book.verfuegbar == null ? null : Math.max(0, book.verfuegbar - vorgemerkt)
	);
	const reichtNicht = $derived(
		rechnerischFrei != null && vorgemerkt > 0 && Number(form.anzahl) > rechnerischFrei
	);

	// Steht nichts im Regal, aber etwas ist bestellt, sagt das Abzeichen „30 bestellt" wie
	// im Medienkatalog. „nicht verfügbar (0 im Bestand)" läse sich wie ein Titel, den es
	// nicht gibt — reservieren lässt er sich aber schon.
	const nurBestellt = $derived(book.gesamt === 0 && book.im_zulauf > 0);
</script>

<div class="w-full">
	<!-- Mit Umbruch: Reicht die Breite nicht für Titel und Aktionen nebeneinander, rücken die
	     Aktionen unter den Text, statt den Titel auf null zu drücken. -->
	<div class="flex flex-wrap gap-4 p-4">
		<div
			class="flex h-20 w-16 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-outline-variant bg-surface-container-low"
		>
			{#if bild}
				<img src={bild} alt="Cover" class="h-full w-full object-cover" loading="lazy" />
			{:else}
				<BookOpen class="h-7 w-7 text-outline" aria-hidden="true" />
			{/if}
		</div>

		<div class="min-w-0 flex-1 basis-48">
			<h3 class="truncate text-base leading-tight font-medium text-on-surface">
				{book.titel ?? book.title ?? 'Unbekannter Titel'}
			</h3>
			<p class="mt-0.5 text-xs text-on-surface-variant">{book.autor ?? book.author ?? ''}</p>
			{#if book.isbn}
				<p class="mt-1 text-label-small text-outline">ISBN {book.isbn}</p>
			{/if}

			<!-- Für einen Klassensatz zählt beides: wie viele gerade frei sind UND wie viele
			     es überhaupt gibt. „3 verfügbar" allein sagt einer Lehrkraft nicht, ob der
			     Titel für 28 Schüler je reichen kann. -->
			{#if book.verfuegbar != null}
				<p class="mt-1.5 text-xs">
					<span
						class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-label-small font-medium {nurBestellt
							? 'bg-surface-container-high text-on-surface-variant'
							: book.verfuegbar > 0
								? 'bg-secondary-container text-on-secondary-container'
								: 'bg-error-container text-on-error-container'}"
					>
						{#if nurBestellt}
							{bestandSatz(book.gesamt, book.verfuegbar, book.im_zulauf)}
						{:else if book.verfuegbar > 0}
							{book.verfuegbar} von {book.gesamt} verfügbar
						{:else}
							nicht verfügbar ({book.gesamt} im Bestand)
						{/if}
					</span>
					{#if vorgemerkt > 0 && rechnerischFrei != null}
						<span
							class="ml-1 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-label-small font-medium {rechnerischFrei >
							0
								? 'bg-surface-container-high text-on-surface-variant'
								: 'bg-error-container text-on-error-container'}"
						>
							{vorgemerkt} vorgemerkt · {rechnerischFrei} rechnerisch frei
						</span>
					{/if}
				</p>
			{/if}

			<!-- Die Warteschlange VOR dem Klick: Reservieren sperrt nichts — wer denselben
			     Titel will, stellt sich an. Ohne diese Zeile erführe die Lehrkraft erst aus
			     der Bestätigung, dass die 8a vor ihr dran ist. -->
			{#each warteschlange as o, _i (_i)}
				<p class="mt-1 text-xs">
					<span
						class="inline-flex items-center gap-1 rounded-full bg-surface-container-high px-2 py-0.5 text-label-small font-medium text-on-surface-variant"
					>
						{o.anzahl} reserviert für {o.klasse} (seit {o.erstellt_am})
					</span>
				</p>
			{/each}
		</div>

		<!-- Zwei Aktionen in zwei Gewichten: die häufige als umrandeter Knopf, die seltene als
		     Textknopf; der gefüllte Knopf bleibt dem Absenden im Formular. Die Bestätigung
		     ersetzt den Knopf nicht: Wer den Titel für die 8a reserviert hat, braucht ihn
		     direkt danach für die 8b. -->
		<div class="ml-auto flex max-w-full shrink-0 flex-col items-end gap-2">
			<div class="flex flex-wrap items-center justify-end gap-1">
				<Button variant="secondary" size="sm" onclick={ontoggle}>
					{#if form.open}
						Abbrechen
					{:else if form.success}
						Weitere Klasse reservieren
					{:else}
						Klassensatz reservieren
					{/if}
				</Button>
				<Button
					variant="ghost"
					size="sm"
					bind:element={meldeKnopf}
					aria-expanded={meldung.open}
					onclick={onmelden}
				>
					Problem melden
				</Button>
			</div>
			{#if form.success}
				<span class="text-xs font-medium text-primary" title={form.success}>✓ Gesendet</span>
			{/if}
		</div>
	</div>

	{#if form.open}
		<div class="border-t border-outline-variant bg-surface-container-low px-4 py-4">
			<KlassensatzFormular {form} {reichtNicht} {rechnerischFrei} {warteschlange} {onsenden} />
		</div>
	{:else if meldung.open}
		<div class="border-t border-outline-variant bg-surface-container-low px-4 py-4">
			<ProblemFormular form={meldung} onsenden={sendeMeldung} onabbrechen={schliesseMeldung} />
		</div>
	{/if}
</div>
