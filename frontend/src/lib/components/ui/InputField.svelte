<script>
  /**
   * @typedef {Object} Props
   * @property {string} id - HTML ID for the input and label.
   * @property {string} label - Display label text.
   * @property {string} [type] - HTML input type (e.g. text, date, number, email).
   * @property {string|number} value - The bound value.
   * @property {string} [placeholder] - Placeholder text.
   * @property {string|null} [maxlength] - Maximum string length.
   * @property {string} [extraClasses] - Additional utility classes for the input field.
   */

  /** @type {Props} */
  let {
    id,
    label,
    type = "text",
    value = $bindable(),
    placeholder = "",
    maxlength = null,
    extraClasses = ""
  } = $props();
</script>

<div>
  <label for={id} class="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">{label}</label>
  <input
    {id}
    {type}
    bind:value={value}
    placeholder={placeholder}
    maxlength={maxlength}
    class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-800 
           focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all {extraClasses}"
  />
</div>
