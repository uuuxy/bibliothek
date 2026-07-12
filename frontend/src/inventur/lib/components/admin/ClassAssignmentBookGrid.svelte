<script>
	let { books = [], selectedBookIds = $bindable(new Set()) } = $props();

	let searchQuery = $state('');

	const filteredBooks = $derived(
		books.filter(
			(b) =>
				b.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
				b.subject.toLowerCase().includes(searchQuery.toLowerCase()) ||
				b.author.toLowerCase().includes(searchQuery.toLowerCase())
		)
	);

	/**
	 * @param {string} id
	 */
	function toggleBook(id) {
		if (selectedBookIds.has(id)) {
			selectedBookIds = new Set([...selectedBookIds].filter((bId) => bId !== id));
		} else {
			selectedBookIds = new Set([...selectedBookIds, id]);
		}
	}

	/**
	 * @param {Event & { target: any }} event
	 */
	function handleImageError(event) {
		event.target.style.display = 'none';
		event.target.nextElementSibling.style.display = 'flex';
	}
</script>

<div class="mb-4 px-1">
	<p class="text-xs uppercase text-gray-500 font-bold mb-1">BÜCHER FINDEN</p>

	<div
		class="bg-emerald-50 border border-surface-variant/10 rounded-full flex items-center px-4 sm:px-6 py-3 sm:py-4 shadow-sm hover:shadow-md transition-shadow group focus-within:ring-2 focus-within:ring-emerald-300"
	>
		<svg
			xmlns="http://www.w3.org/2000/svg"
			width="24"
			height="24"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
			class="text-gray-500 mr-2 sm:mr-4 group-focus-within:text-emerald-600 transition-colors hidden sm:block"
			><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"
			></line></svg
		>
		<input
			id="book-search-field"
			name="book-search-field-hidden"
			type="search"
			autocomplete="off"
			spellcheck="false"
			data-lpignore="true"
			data-form-type="other"
			placeholder="Suche Titel, Fach, oder ISBN..."
			bind:value={searchQuery}
			class="grow text-base sm:text-xl w-full min-w-0 bg-transparent border-none outline-none focus:ring-0 text-gray-900 placeholder:text-gray-400 font-medium"
		/>
		<span
			class="ml-2 sm:ml-4 text-[10px] sm:text-xs font-bold text-gray-500 bg-black/5 px-2 py-1 sm:px-3 sm:py-1 rounded-full whitespace-nowrap"
			>{filteredBooks.length} Treffer</span
		>
	</div>
</div>

<div
	class="grid grid-cols-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 sm:gap-6 pb-2 mt-6 sm:mt-8"
>
	{#each filteredBooks as book (book.id)}
		<button
			onclick={() => toggleBook(book.id)}
			aria-pressed={selectedBookIds.has(book.id)}
			class="group relative flex flex-col text-left rounded-3xl overflow-hidden transition-all duration-300 transform active:scale-95
            {selectedBookIds.has(book.id)
				? 'bg-primary-50 ring-4 ring-primary-500 shadow-xl scale-[1.02]'
				: 'bg-white hover:bg-surface-container-low shadow-md hover:shadow-xl border border-surface-variant/10 hover:border-primary-200'}"
		>
			<!-- Selection Overlay -->
			{#if selectedBookIds.has(book.id)}
				<div
					class="absolute top-4 right-4 z-10 bg-primary-600 text-white p-1.5 rounded-full shadow-lg border-2 border-white animate-in zoom-in-50 duration-200"
				>
					<svg
						width="18"
						height="18"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="3"
						stroke-linecap="round"
						stroke-linejoin="round"><path d="M20 6 9 17l-5-5" /></svg
					>
				</div>
			{/if}

			<!-- Cover -->
			<div class="aspect-2/3 w-full overflow-hidden bg-surface-container relative shrink-0">
				{#if book.coverUrl}
					<img
						src={book.coverUrl}
						alt={book.title}
						class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
						onerror={handleImageError}
					/>
					<div class="w-full h-full hidden items-center justify-center bg-gray-100 text-gray-300">
						<svg
							width="48"
							height="48"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
							><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20" /></svg
						>
					</div>
				{:else}
					<div class="w-full h-full flex items-center justify-center bg-gray-100 text-gray-300">
						<svg
							width="48"
							height="48"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
							><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20" /></svg
						>
					</div>
				{/if}
				<div
					class="absolute inset-0 bg-linear-to-t from-black/20 to-transparent group-hover:from-black/40 transition-colors"
				></div>
			</div>

			<!-- Content -->
			<div class="p-5 flex flex-col grow justify-end space-y-3 w-full">
				<div class="flex flex-wrap gap-1.5 items-start">
					<span
						class="px-2.5 py-0.5 bg-primary-100 text-primary-900 text-[10px] font-black uppercase rounded-lg tracking-wider"
						>{book.subject}</span
					>
					<span
						class="px-2.5 py-0.5 bg-surface-container-high text-surface-variant text-[10px] font-black uppercase rounded-lg tracking-wider"
					>
						Kl. {book.gradeLevel}
						{#if book.track}
							({book.track})
						{/if}
					</span>
				</div>
				<div>
					<h3
						class="font-bold text-primary-950 leading-tight line-clamp-2 group-hover:text-primary-700 transition-colors"
					>
						{book.title}
					</h3>
				</div>
			</div>
		</button>
	{/each}
</div>
