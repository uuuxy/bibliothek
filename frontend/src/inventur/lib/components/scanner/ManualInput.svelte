<script>
	import Button from '../../../../lib/components/ui/Button.svelte';
	import Feld from '../../../../lib/components/ui/Feld.svelte';
	/**
	 * @type {{
	 *   onSubmit: (isbn: string) => void,
	 *   disabled: boolean
	 * }}
	 */
	let { onSubmit, disabled } = $props();
	let manualISBN = $state('');

	/**
	 * @param {KeyboardEvent} event
	 */
	function handleKeydown(event) {
		if (event.key === 'Enter') {
			event.preventDefault();
			onSubmit(manualISBN);
			manualISBN = '';
		}
	}
</script>

<div class="mt-4">
	<!-- Beschriftung bleibt eigenes <label>: Der Knopf steht in der Feldzeile, nicht
	     unter der Beschriftung — mit `label=` am Feld säße er neben beiden Zeilen. -->
	<label for="manual-isbn" class="text-sm font-medium text-on-surface-variant"
		>Handscanner / ISBN-Eingabe</label
	>
	<div class="mt-1.5 flex gap-2">
		<Feld
			id="manual-isbn"
			bind:value={manualISBN}
			onkeydown={handleKeydown}
			placeholder="ISBN scannen oder eintippen"
		/>
		<Button
			onclick={() => {
				onSubmit(manualISBN);
				manualISBN = '';
			}}
			{disabled}
			class="px-5"
		>
			Senden
		</Button>
	</div>
</div>
