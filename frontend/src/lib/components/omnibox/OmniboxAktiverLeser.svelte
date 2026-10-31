<script>
	import Leserakte from '../leserakte/Leserakte.svelte';
	import LeserakteKopf from '../leserakte/LeserakteKopf.svelte';
	import OmniboxThekeHinweise from './OmniboxThekeHinweise.svelte';
	import { omniboxStore } from '../../stores/omnibox.svelte.js';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { schuelerRechte } from '../leser/schuelerRechte.js';

	let { profil = $bindable(null) } = $props();

	// In aktiverLeser steht ein Leser, Schüler oder Kollege, und beide bekommen dieselbe Akte:
	// Die Theke muss sehen, welche Bücher der Mensch vor ihr hat, und das ist bei einem
	// Kollegen dieselbe Frage wie bei einem Kind. Was an ihm anders ist (keine Klasse, Frist
	// ein Jahr), sagen die Akte selbst und der Hinweis darüber.
	const leser = $derived(omniboxStore.aktiverLeser);
	// Die Akte steht hinter view_students; die Theke hängt an perform_actions. Ohne das Recht, ab
	// Werk bei Helfern, zeigt sie den Kopf aus dem, was die Antwort auf den Scan schon trägt.
	const darfAkte = $derived(schuelerRechte(authStore.currentUser).einsehen);

	function abwaehlen() {
		omniboxStore.aktiverLeser = null;
		omniboxStore.letzteFremdrueckgabe = null;
	}
</script>

{#if leser}
	<!-- Fremdrückgabe-, Abholfach- und Kollegiums-Banner (200-Zeilen-Regel: eigene Datei). -->
	<OmniboxThekeHinweise />
	{#if darfAkte}
		<Leserakte
			bind:this={profil}
			{leser}
			startReiter="ausleihen"
			onMerged={omniboxStore.uebernimmZusammengefuehrt}
			onDeselect={abwaehlen}
			onReturnClick={(barcode) => omniboxStore.gibZurueck(barcode, () => profil?.ladeNeu())}
		/>
	{:else}
		<!-- Dasselbe Raster wie in der Akte: Der Kopf steht an seinem Platz, daneben der Satz, warum
		     dort nichts weiter steht. -->
		<div
			class="w-full grid grid-cols-1 lg:grid-cols-[320px_minmax(0,1fr)] items-stretch text-on-surface font-sans no-print"
		>
			<LeserakteKopf
				{leser}
				timestamp={0}
				showWebcam={false}
				onDeselect={abwaehlen}
				schliessenEtikett="Leser schließen"
			/>
			<p class="px-6 pt-4 text-sm text-on-surface-variant">
				Die Akte mit Ausleihen und Gebühren braucht das Recht „Leserdatei anzeigen“.
			</p>
		</div>
	{/if}
{/if}
