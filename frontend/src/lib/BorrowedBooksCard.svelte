<script>
	import BorrowedBooksList from './BorrowedBooksList.svelte';
	import { BookOpen } from '@lucide/svelte';

	/** @type {{ books: any[], onReturnClick?: (barcode: string) => void, onDamageClick?: (book: any) => void }} */
	let { books = [], onReturnClick = undefined, onDamageClick = undefined } = $props();
</script>

<div class="w-full h-full pt-2">
	<div class="flex items-center justify-between pb-3 border-b border-outline-variant mb-6">
		<h3 class="text-base font-medium text-on-surface-variant">
			Entliehene Bücher ({books?.length || 0})
		</h3>
	</div>

	{#if !books || books.length === 0}
		<div class="py-16 flex flex-col items-center justify-center text-on-surface-variant space-y-3">
			<BookOpen class="h-12 w-12" aria-hidden="true" />
			<span class="text-sm font-semibold">Aktuell keine Bücher entliehen.</span>
		</div>
	{:else}
		<BorrowedBooksList {books} {onReturnClick} {onDamageClick} />
	{/if}
</div>
