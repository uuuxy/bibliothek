<script>
	import { authStore } from '../../stores/authStore.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import { darfVerwerfen } from '../../stores/verlassenSchutz.svelte.js';
	import { hatRecht } from '../../menue.js';
	import BackupStatusBadge from '../system/BackupStatusBadge.svelte';
	import { ChevronsLeft } from '@lucide/svelte';
	import SeitenleisteZiele from './SeitenleisteZiele.svelte';
	import SeitenleisteFuss from './SeitenleisteFuss.svelte';
	import logoUrl from '../../../assets/logo.png';
	import { MediaQuery } from 'svelte/reactivity';
	import { meldungenWeichenAus } from '../../actions/meldungenWeichenAus.js';

	// Unter 1280 px Fensterbreite (Tailwind `xl`) beginnt die Navigation eingeklappt: Erst ab
	// dieser Breite hat der Titel in der Ausleihliste der Leserakte neben ihr Platz zum Lesen
	// (e2e/ausleihliste-zeilen.spec.js). Wer den Doppelpfeil benutzt, hat gewählt, und die
	// Wahl gilt bis zum Neuladen vor der Fensterbreite.
	const schmalesFenster = new MediaQuery('(width < 80rem)');
	const zu = $derived(uiStore.sidebarWahl ?? schmalesFenster.current);

	// Eingeklappt klappt die Leiste wie bei Gmail auf, solange die Maus über ihr steht, und
	// legt sich dabei über den Inhalt, statt ihn zu verschieben. Erst nach 200 ms, damit ein
	// Zeiger, der nur über sie hinwegfährt, nichts aufklappt; Touch und Stift tippen wie bisher.
	let schwebt = $state(false);
	// Über den Inhalt (z-40) hebt sie sich nur, solange sie breit ist, und beim Einfahren noch
	// 300 ms: Sonst läge sie über den Dialogen, die im Inhalt stehen.
	let faehrtEin = $state(false);
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let schwebeUhr;
	const schmal = $derived(zu && !schwebt);
	const obenAuf = $derived((zu && schwebt) || faehrtEin);

	/** @param {PointerEvent} e */
	function zeigerKommt(e) {
		if (e.pointerType !== 'mouse' || !zu) return;
		clearTimeout(schwebeUhr);
		schwebeUhr = setTimeout(() => (schwebt = true), 200);
	}

	function zeigerGeht() {
		clearTimeout(schwebeUhr);
		faehrtEin = schwebt;
		schwebt = false;
		if (faehrtEin) schwebeUhr = setTimeout(() => (faehrtEin = false), 300);
	}

	// Abmelden baut jeden Bildschirm ab; Ungespeichertes geht nur nach Rückfrage verloren.
	async function meldeAb() {
		if (!(await darfVerwerfen('Verwerfen und abmelden'))) return;
		authStore.handleLogout(() => {
			uiStore.activeTab = 'kiosk';
		});
	}
</script>

<!-- Doppelpfeil zum Ein-/Ausklappen. Er steht vorn, damit er beim Aufklappen unter der Maus
     bleibt. Wer klickt, hat gewählt: Die Leiste klappt erst beim nächsten Überfahren wieder auf. -->
{#snippet klappPfeil(/** @type {boolean} */ einklappen)}
	<button
		onclick={() => {
			uiStore.sidebarWahl = einklappen;
			zeigerGeht();
		}}
		class="icon-btn text-on-surface-variant hover:bg-surface-container shrink-0"
		aria-label={einklappen ? 'Navigation einklappen' : 'Navigation ausklappen'}
		data-tip={einklappen ? 'Navigation einklappen' : 'Navigation ausklappen'}
	>
		<ChevronsLeft class="h-4.5 w-4.5 {einklappen ? '' : 'rotate-180'}" aria-hidden="true" />
	</button>
{/snippet}

<!-- Die Leiste hält im Seitenaufbau ihre Breite (64 oder 256 px); die Fläche darin liegt
     darüber und wird beim Überfahren breiter. Über dem Inhalt (data-ueber-inhalt) trägt sie
     Googles Werte für die überlagernde Leiste (nav-rail-expanded, modal-container-*:
     surface-container, Erhebung Stufe 2 wie die Menüs, Ecke large) und gibt dafür den Rahmen
     ab. z-40 liegt über den angedockten Leisten des Inhalts und unter den Dialogen. -->
<aside
	class="no-print relative h-screen shrink-0 transition-all duration-300 {zu ? 'w-16' : 'w-64'}"
	use:meldungenWeichenAus={'links'}
	onpointerenter={zeigerKommt}
	onpointerleave={zeigerGeht}
>
	<div
		class="border-outline-variant bg-surface absolute inset-y-0 left-0 flex flex-col justify-between overflow-x-hidden overflow-y-auto border-r transition-all duration-300
			data-ueber-inhalt:bg-surface-container data-ueber-inhalt:rounded-r-2xl data-ueber-inhalt:border-r-0 data-ueber-inhalt:shadow-xl
			{schmal ? 'w-16' : 'w-64'} {obenAuf ? 'z-40' : ''}"
		data-ueber-inhalt={zu && !schmal ? '' : undefined}
	>
		<div>
			<div class="border-outline-variant/60 flex h-16 shrink-0 items-center gap-3 border-b px-4">
				{@render klappPfeil(!zu)}
				{#if !schmal}
					<img src={logoUrl} alt="Logo" class="animate-fade-in h-12 w-12 shrink-0 object-contain" />
					<span
						class="text-on-surface animate-fade-in whitespace-nowrap text-xl font-bold tracking-tight"
						>Bibliosys</span
					>
				{/if}
			</div>

			<SeitenleisteZiele {zu} {schmal} nachWahl={zeigerGeht} />
		</div>

		<div class="border-outline-variant/60 mt-auto border-t">
			<!-- Backup-Wächter: dasselbe Recht wie seine Route (manage_settings) -->
			{#if hatRecht(authStore.currentUser, 'manage_settings')}
				<BackupStatusBadge collapsed={schmal} />
			{/if}

			<SeitenleisteFuss collapsed={schmal} benutzer={authStore.currentUser} onLogout={meldeAb} />
		</div>
	</div>
</aside>
