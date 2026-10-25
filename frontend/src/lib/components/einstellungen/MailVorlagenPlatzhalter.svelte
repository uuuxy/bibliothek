<script>
	import { vorlagenInfo } from './mailVorlagenInfo.js';

	/**
	 * @component MailVorlagenPlatzhalter
	 * Verwendung und erlaubte Platzhalter je Vorlagen-Typ, unter dem Vorlagen-Editor. Jede
	 * Vorlage hat ihre eigene Liste: Ein Platzhalter, den der Renderer dieser Vorlage nicht
	 * ersetzt, stünde wörtlich im Versand.
	 *
	 * @prop {string} typ - Vorlagen-Typ (mail_vorlagen.typ), z. B. MAHNUNG_ELTERN.
	 */
	let { typ } = $props();

	const info = $derived(vorlagenInfo[typ] ?? null);
</script>

<!-- Flacher Akzent statt Kachel. Eine Vorlage ohne Versandweg bekommt eine
     Warnung statt einer Anleitung. -->
{#if info}
	{#if info.platzhalter.length === 0}
		<div class="border-l-2 border-error py-1 pl-4">
			<h4 class="mb-1 text-sm font-bold text-on-surface">Ohne Wirkung</h4>
			<p class="text-xs leading-relaxed text-on-surface-variant">{info.verwendung}</p>
		</div>
	{:else}
		<div class="border-l-2 border-primary py-1 pl-4">
			<h4 class="mb-1 text-sm font-bold text-on-surface">Erlaubte Platzhalter</h4>
			<p class="text-xs leading-relaxed text-on-surface-variant">
				{info.verwendung}
				<br />
				{#each info.platzhalter as p (p)}
					<code
						class="bg-surface-container mt-1 mr-1 inline-block rounded px-1.5 py-0.5 text-on-surface"
						>{p}</code
					>
				{/each}
			</p>
		</div>
	{/if}
{/if}
