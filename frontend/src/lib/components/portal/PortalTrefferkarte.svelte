<script>
	/**
	 * @component PortalTrefferkarte
	 * Ein Suchtreffer im Kollegiums-Portal: Cover, Titelangaben, Verfügbarkeit, die
	 * Warteschlange — und das aufklappbare Formular für die Klassensatz-Reservierung.
	 *
	 * Aus KollegiumPortal.svelte herausgelöst (23.08.2026): Die Datei trug 389 Zeilen,
	 * davon 150 diese eine Karte. Die stehende Vorgabe im Haus sind 200 Zeilen je Datei.
	 *
	 * Die Formularfelder sind Feld wie überall sonst — Rahmen statt Füllung,
	 * Beschriftung über dem Feld. Vorher waren es drei handgebaute Felder mit eigener
	 * Rundung und eigenem Fokusring.
	 *
	 * @prop {any} book
	 * @prop {any} form - Formularzustand dieses Titels (reaktives Objekt des Aufrufers).
	 * @prop {{ klasse: string, anzahl: number, erstellt_am: string }[]} warteschlange
	 * @prop {() => void} ontoggle
	 * @prop {() => void} onsenden
	 */
	import { BookOpen } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import Feld from '../ui/Feld.svelte';
	import { coverSrc } from '../../utils/coverSrc.js';

	/** @type {{ book: any, form: any, warteschlange: { klasse: string, anzahl: number, erstellt_am: string }[], ontoggle: () => void, onsenden: () => void }} */
	let { book, form, warteschlange, ontoggle, onsenden } = $props();

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
</script>

<div class="w-full">
	<div class="flex gap-4 p-4">
		<div
			class="flex h-20 w-16 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-outline-variant bg-surface-container-low"
		>
			{#if bild}
				<img src={bild} alt="Cover" class="h-full w-full object-cover" loading="lazy" />
			{:else}
				<BookOpen class="h-7 w-7 text-outline" aria-hidden="true" />
			{/if}
		</div>

		<div class="min-w-0 flex-1">
			<h3 class="truncate text-sm leading-tight font-medium text-on-surface">
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
						class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-label-small font-medium {book.verfuegbar >
						0
							? 'bg-secondary-container text-on-secondary-container'
							: 'bg-error-container text-on-error-container'}"
					>
						{book.verfuegbar > 0
							? `${book.verfuegbar} von ${book.gesamt} verfügbar`
							: `nicht verfügbar (${book.gesamt} im Bestand)`}
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

		<!-- Die Bestätigung ERSETZT den Knopf nicht: Eine Lehrkraft, die denselben Titel
		     für 8a bestellt hat, braucht ihn direkt danach für 8b. Vorher blieb
		     „✓ Gesendet" für immer stehen und der einzige Weg zurück war ein Reload. -->
		<div class="flex shrink-0 flex-col items-end justify-between gap-2">
			{#if form.success}
				<span class="text-xs font-medium text-primary" title={form.success}>✓ Gesendet</span>
			{/if}
			<Button
				variant={form.open || form.success ? 'secondary' : 'primary'}
				size="sm"
				onclick={ontoggle}
			>
				{#if form.open}
					Abbrechen
				{:else if form.success}
					Weitere Klasse reservieren
				{:else}
					Klassensatz reservieren
				{/if}
			</Button>
		</div>
	</div>

	{#if form.open}
		<div
			class="flex flex-col gap-4 border-t border-outline-variant bg-surface-container-low px-4 py-4"
		>
			<p class="text-sm font-medium text-on-surface">Klassensatz-Reservierung</p>
			<div class="grid grid-cols-2 gap-4">
				<Feld bind:value={form.klasse} label="Klasse *" type="text" placeholder="z. B. 8b" />
				<Feld type="number" bind:value={form.anzahl} label="Anzahl" min={1} max={200} />
			</div>
			<label class="grid gap-y-1.5">
				<span class="text-sm font-medium text-on-surface-variant">Notiz (optional)</span>
				<textarea
					bind:value={form.notiz}
					rows="2"
					placeholder="z. B. Benötigt ab 15. September …"
					class="w-full resize-none rounded-sm border border-outline-variant bg-surface-container-lowest px-3 py-2 text-base text-on-surface transition-colors placeholder:text-outline focus:border-primary focus:outline-none"
				></textarea>
			</label>
			<!-- Anstellen bleibt erlaubt — aber die Lehrkraft soll es VOR dem Absenden wissen,
			     nicht erst aus der Bestätigung. -->
			{#if reichtNicht}
				<p class="text-xs text-on-surface-variant" role="status">
					Reicht aktuell nicht: {rechnerischFrei} rechnerisch frei — du stellst dich hinter
					{warteschlange.map((o) => o.klasse).join(', ')} an.
				</p>
			{/if}
			{#if form.error}
				<p class="text-xs text-error">{form.error}</p>
			{/if}
			<div class="flex justify-end">
				<Button onclick={onsenden} disabled={form.loading}>
					{form.loading ? 'Wird gesendet …' : 'Anfrage senden'}
				</Button>
			</div>
		</div>
	{/if}
</div>
