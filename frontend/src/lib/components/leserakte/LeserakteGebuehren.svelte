<!-- @component LeserakteGebuehren — der Reiter „Gebühren & Schäden" der Akte: die
     Forderungen und die Bescheide dazu.

     Eigener Reiter, damit beides nicht unter einer langen Ausleihliste aus dem Fenster
     rutscht; die Zahl am Reiter und die Zeile im Konto-Status sagen vorher, ob etwas offen ist. -->
<script>
	import LeserGebuehrenKarte from './LeserGebuehrenKarte.svelte';
	import LeserBescheideKarte from './LeserBescheideKarte.svelte';
	import ListenNichtGeladen from '../leser/ListenNichtGeladen.svelte';

	/**
	 * @type {{
	 *   schuelerId?: string,
	 *   gebuehren: any[],
	 *   bescheide: any[],
	 *   fehlendeListen?: { name: string, grund: string }[],
	 *   darfBearbeiten: boolean,
	 *   onChanged: () => void
	 * }}
	 */
	let {
		schuelerId = '',
		gebuehren = [],
		bescheide = [],
		fehlendeListen = [],
		darfBearbeiten = false,
		onChanged
	} = $props();
</script>

<div class="relative flex flex-col gap-6 h-full min-h-100 animate-fade-in mt-4">
	<ListenNichtGeladen listen={fehlendeListen} onErneut={onChanged} />

	{#if fehlendeListen.length === 0 && gebuehren.length === 0 && bescheide.length === 0}
		<p class="text-sm text-on-surface-variant">Keine Gebühren, Schäden oder Bescheide.</p>
	{/if}

	<LeserGebuehrenKarte {schuelerId} {gebuehren} {darfBearbeiten} {onChanged} />

	<LeserBescheideKarte {bescheide} />
</div>
