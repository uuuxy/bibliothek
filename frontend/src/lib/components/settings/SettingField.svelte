<script>
  /**
   * @component SettingField
   * Flaches Einstellungs-Eingabefeld (Label + Unterstrich-Input) im Edge-to-Edge-Stil,
   * konsistent zur Mail-Server-Konfiguration. Bewusst ohne Box-/Kachel-Hintergrund —
   * liegt direkt auf dem Seiten-Grund.
   *
   * Zwei-Wege-Bindung erfordert eine Komponente (Snippets können `bind:` nicht
   * zurückpropagieren), daher hier statt eines {#snippet} gekapselt.
   *
   * @prop {string|number} value - Gebundener Wert (bindable).
   * @prop {string} label - Großgeschriebene Feldbeschriftung.
   * @prop {'number'|'text'|'email'|'date'} [type='number'] - Eingabetyp.
   * @prop {string} [hint=''] - Optionaler Hilfetext unter dem Feld.
   * @prop {number} [min] - Minimalwert (number).
   * @prop {number} [max] - Maximalwert (number).
   * @prop {string} [placeholder=''] - Platzhaltertext.
   * @prop {string} [pattern] - Validierungs-Pattern (text).
   * @prop {number} [maxlength] - Maximale Zeichenanzahl (text).
   */

  /** @type {{ value: string|number, label: string, type?: 'number'|'text'|'email'|'date', hint?: string, min?: number, max?: number, placeholder?: string, pattern?: string, maxlength?: number }} */
  let { value = $bindable(), label, type = 'number', hint = '', min, max, placeholder = '', pattern, maxlength } = $props();

  const inputClass = "bg-transparent border-b border-slate-200 py-2 text-slate-800 focus:border-blue-600 focus:outline-none transition-colors w-full";
</script>

<label class="flex flex-col">
  <span class="text-xs font-bold text-slate-500 uppercase tracking-wider mb-2">{label}</span>
  {#if type === 'number'}
    <input type="number" {min} {max} {placeholder} bind:value class={inputClass} />
  {:else}
    <input {type} {placeholder} {pattern} {maxlength} bind:value class={inputClass} />
  {/if}
  {#if hint}
    <span class="text-[11px] text-slate-500 mt-2">{hint}</span>
  {/if}
</label>
