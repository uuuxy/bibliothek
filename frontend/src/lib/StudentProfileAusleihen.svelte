<!-- @component StudentProfileAusleihen — der Reiter „Ausleihen & Historie" der Akte.

     Gegenstück zu StudentProfileStammdaten: Beide Reiter haben jetzt je eine Datei, und
     StudentProfile hält nur noch Kopf, Reiterleiste und die Blätter darüber.

     Herausgelöst am 12.09.2026 beim Einbau der Bescheid-Karte (#597). Die Akte stand mit
     255 Zeilen im Bestand der Größen-Ratsche, und eine geduldete Datei darf nicht weiter
     wachsen — die Ausnahme ist kein Freibrief. Reine Verschiebung: dieselben Karten in
     derselben Reihenfolge. -->
<script>
	import BorrowedBooksCard from './BorrowedBooksCard.svelte';
	import StudentVormerkungenCard from './StudentVormerkungenCard.svelte';
	import StudentGebuehrenCard from './StudentGebuehrenCard.svelte';
	import StudentBescheideCard from './StudentBescheideCard.svelte';

	/**
	 * @type {{
	 *   buecher: any[],
	 *   vormerkungen: any[],
	 *   gebuehren: any[],
	 *   bescheide: any[],
	 *   canEdit: boolean,
	 *   onReturnClick?: (barcode: string) => void,
	 *   onDamageClick?: (buch: any) => void,
	 *   onChanged: () => void,
	 *   rightTop?: import('svelte').Snippet
	 * }}
	 */
	let {
		buecher = [],
		vormerkungen = $bindable([]),
		gebuehren = [],
		bescheide = [],
		canEdit = false,
		onReturnClick = undefined,
		onDamageClick = undefined,
		onChanged,
		rightTop
	} = $props();
</script>

{@render rightTop?.()}
<div
	class="col-span-1 md:col-span-1 relative flex flex-col gap-6 h-full min-h-100 animate-fade-in mt-4"
>
	<BorrowedBooksCard books={buecher} {onReturnClick} {onDamageClick} />

	{#if vormerkungen.length > 0}
		<StudentVormerkungenCard bind:vormerkungen />
	{/if}

	<StudentGebuehrenCard {gebuehren} {canEdit} {onChanged} />

	<StudentBescheideCard {bescheide} />
</div>
