<script>
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import { meldungsPlatz, RAND } from './meldungsPlatz.js';
	import Icon from '../ui/Icon.svelte';
	import Button from '../ui/Button.svelte';

	let hoehe = $state(0);
	let platz = $state({ links: 0, unten: RAND });

	// Gemessen wird, solange eine Meldung steht: beim Erscheinen, beim Scrollen (eine
	// Auswahlleiste klebt am unteren Rand ihrer Liste), wenn eine Leiste kommt, geht oder ihre
	// Größe ändert, und beim Ein- und Ausklappen der Navigation. Höchstens einmal je Bild, wie
	// use:resthoehe.
	$effect(() => {
		if (snackbars.toasts.length === 0) return;
		const meiden = [...snackbars.ausweichen];
		const stapel = hoehe;
		/** @param {'unten' | 'links'} seite */
		const kaesten = (seite) =>
			meiden.filter(([, s]) => s === seite).map(([el]) => el.getBoundingClientRect());
		const messen = () => {
			platz = meldungsPlatz({
				fensterHoehe: window.innerHeight,
				hoehe: stapel,
				navigation: kaesten('links'),
				leisten: kaesten('unten')
			});
		};
		let geplant = 0;
		const planen = () => {
			if (!geplant)
				geplant = requestAnimationFrame(() => {
					geplant = 0;
					messen();
				});
		};
		messen();
		window.addEventListener('scroll', planen, true);
		window.addEventListener('resize', planen);
		const beobachter = new ResizeObserver(planen);
		for (const [el] of meiden) beobachter.observe(el);
		return () => {
			window.removeEventListener('scroll', planen, true);
			window.removeEventListener('resize', planen);
			beobachter.disconnect();
			cancelAnimationFrame(geplant);
		};
	});
</script>

<!-- Unten und mittig über der Fläche rechts der Navigation (Material 3, Snackbar, „Placement":
     „Snackbars should be placed at the bottom of a UI, in front of the main content"); wo und
     warum genau, steht in meldungsPlatz.js. z-9999: Toasts müssen über allen Modals (z-50/z-60)
     und Omnibox-Alerts (z-100) liegen. no-print: Eine Meldung, die beim Druck steht, käme sonst
     aufs Papier, auf eine Ausweiskarte mitten auf die Karte. -->
<div
	class="no-print pointer-events-none fixed right-0 z-9999 flex flex-col items-center gap-2 px-4"
	style:left="{platz.links}px"
	style:bottom="{platz.unten}px"
	bind:clientHeight={hoehe}
>
	{#each snackbars.toasts as toast (toast.id)}
		<!-- Fehler sind alert (der Screenreader unterbricht), alles andere status (er
		     wartet). Maus oder Fokus auf dem Toast halten die Standzeit an — siehe
		     snackbars.pausieren.
		     Eine Fläche für jede Meldung, wie Googles Snackbar (material-components-android:
		     backgroundTint colorSurfaceInverse, Text colorOnSurfaceInverse in body medium, Ecke
		     corner extra small, Erhebung 6 dp = level3) und ohne Zeichen davor (M3, Snackbar:
		     „Avoid adding icons to snackbars"). -->
		<div
			role={toast.type === 'error' ? 'alert' : 'status'}
			onmouseenter={() => snackbars.pausieren(toast.id)}
			onmouseleave={() => snackbars.fortsetzen(toast.id)}
			onfocusin={() => snackbars.pausieren(toast.id)}
			onfocusout={() => snackbars.fortsetzen(toast.id)}
			class="pointer-events-auto flex min-h-12 w-full max-w-sm items-center gap-1 rounded-sm bg-inverse-surface py-1 pr-2 pl-4 text-sm text-inverse-on-surface shadow-level3"
		>
			<span class="w-full py-2 wrap-break-word">{toast.message}</span>
			{#if toast.aktion}
				<!-- Genau eine Folgehandlung als Textknopf in inverse-primary (actionTextColor
				     colorPrimaryInverse; M3: „The text label shouldn't share the same color as the
				     text button"). Sie schließt die Meldung selbst. -->
				<Button
					variant="ghost"
					class="shrink-0 whitespace-nowrap text-inverse-primary"
					onclick={() => {
						toast.aktion?.onClick();
						snackbars.removeToast(toast.id);
					}}
				>
					{toast.aktion.label}
				</Button>
			{/if}
			<button
				onclick={() => snackbars.removeToast(toast.id)}
				class="icon-btn shrink-0"
				aria-label="Schließen"
				data-tip="Schließen"
			>
				<Icon name="close" />
			</button>
		</div>
	{/each}
</div>
