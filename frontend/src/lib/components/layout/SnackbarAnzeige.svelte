<script>
	import { snackbars } from '../../stores/snackbars.svelte.js';
	import { meldungsPlatz, RAND } from './meldungsPlatz.js';
	import { CircleAlert, CircleCheck, Info, TriangleAlert, X } from '@lucide/svelte';

	const symbole = {
		error: CircleAlert,
		success: CircleCheck,
		warning: TriangleAlert,
		info: Info
	};

	const flaechen = {
		error: 'bg-error text-on-error',
		success: 'bg-success text-on-success',
		warning: 'bg-warning text-on-warning',
		info: 'bg-inverse-surface text-inverse-on-surface'
	};

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
		{@const Symbol = symbole[toast.type] ?? Info}
		<!-- Fehler sind alert (der Screenreader unterbricht), alles andere status (er
		     wartet). Maus oder Fokus auf dem Toast halten die Standzeit an — siehe
		     snackbars.pausieren. -->
		<div
			role={toast.type === 'error' ? 'alert' : 'status'}
			onmouseenter={() => snackbars.pausieren(toast.id)}
			onmouseleave={() => snackbars.fortsetzen(toast.id)}
			onfocusin={() => snackbars.pausieren(toast.id)}
			onfocusout={() => snackbars.fortsetzen(toast.id)}
			class="pointer-events-auto flex items-start gap-2 px-4 py-3 rounded-sm text-sm max-w-sm w-full {flaechen[
				toast.type
			] ?? flaechen.info}"
		>
			<Symbol class="h-5 w-5 shrink-0" aria-hidden="true" />
			<span class="wrap-break-word w-full">{toast.message}</span>
			{#if toast.aktion}
				<!-- M3-Snackbar: genau eine Folgehandlung, textbetont und rechts neben der
				     Meldung. Sie schließt den Toast selbst — die Meldung hat ihren Zweck
				     erfüllt, sobald man ihr gefolgt ist. -->
				<button
					onclick={() => {
						toast.aktion?.onClick();
						snackbars.removeToast(toast.id);
					}}
					class="ml-2 shrink-0 rounded-sm px-1 font-semibold whitespace-nowrap underline underline-offset-2 hover:opacity-80"
				>
					{toast.aktion.label}
				</button>
			{/if}
			<button
				onclick={() => snackbars.removeToast(toast.id)}
				class="icon-btn ml-2 shrink-0 opacity-70 hover:opacity-100"
				aria-label="Schließen"
				data-tip="Schließen"
			>
				<X class="h-4 w-4" aria-hidden="true" />
			</button>
		</div>
	{/each}
</div>
