<!-- @component StudentProfileAusleihen — der Reiter „Ausleihen & Vormerkungen" der Akte: die
     entliehenen Bücher und die Vormerkungen.

     Gebühren und Bescheide stehen im eigenen Reiter (StudentProfileGebuehren); jeder Reiter
     nennt nur die Listen, die er selbst zeigt. -->
<script>
	import BorrowedBooksCard from './BorrowedBooksCard.svelte';
	import StudentVormerkungenCard from './StudentVormerkungenCard.svelte';
	import ListenNichtGeladen from './components/students/ListenNichtGeladen.svelte';

	/**
	 * @type {{
	 *   buecher: any[],
	 *   vormerkungen: any[],
	 *   fehlendeListen?: string[],
	 *   onReturnClick?: (barcode: string) => void,
	 *   onDamageClick?: (buch: any) => void,
	 *   onChanged: () => void,
	 *   rightTop?: import('svelte').Snippet
	 * }}
	 */
	let {
		buecher = [],
		vormerkungen = $bindable([]),
		fehlendeListen = [],
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
	<ListenNichtGeladen listen={fehlendeListen} onErneut={onChanged} />

	<BorrowedBooksCard books={buecher} {onReturnClick} {onDamageClick} />

	{#if vormerkungen.length > 0}
		<StudentVormerkungenCard bind:vormerkungen />
	{/if}
</div>
