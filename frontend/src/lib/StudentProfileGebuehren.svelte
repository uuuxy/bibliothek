<!-- @component StudentProfileGebuehren — der Reiter „Gebühren & Schäden" der Akte: die
     Forderungen und die Bescheide dazu.

     Eigener Reiter, damit beides nicht unter einer langen Ausleihliste aus dem Fenster
     rutscht; die Zahl am Reiter und die Zeile im Konto-Status sagen vorher, ob etwas offen ist. -->
<script>
	import StudentGebuehrenCard from './StudentGebuehrenCard.svelte';
	import StudentBescheideCard from './StudentBescheideCard.svelte';
	import ListenNichtGeladen from './components/students/ListenNichtGeladen.svelte';

	/**
	 * @type {{
	 *   schuelerId?: string,
	 *   gebuehren: any[],
	 *   bescheide: any[],
	 *   fehlendeListen?: string[],
	 *   canEdit: boolean,
	 *   onChanged: () => void
	 * }}
	 */
	let {
		schuelerId = '',
		gebuehren = [],
		bescheide = [],
		fehlendeListen = [],
		canEdit = false,
		onChanged
	} = $props();
</script>

<div class="relative flex flex-col gap-6 h-full min-h-100 animate-fade-in mt-4">
	<ListenNichtGeladen listen={fehlendeListen} onErneut={onChanged} />

	{#if fehlendeListen.length === 0 && gebuehren.length === 0 && bescheide.length === 0}
		<p class="text-sm text-on-surface-variant">Keine Gebühren, Schäden oder Bescheide.</p>
	{/if}

	<StudentGebuehrenCard {schuelerId} {gebuehren} {canEdit} {onChanged} />

	<StudentBescheideCard {bescheide} />
</div>
