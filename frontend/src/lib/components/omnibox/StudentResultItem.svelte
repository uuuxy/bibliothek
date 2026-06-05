<script>
  let { student, index, selected, onSelect } = $props();
</script>

<div id="dropdown-item-{index}"
     role="option"
     aria-selected={selected}
     aria-label="Schüler: {student.vorname} {student.nachname}, Klasse {student.klasse}, Barcode {student.barcode_id}"
     tabindex="-1"
     class="px-4 py-3 rounded-xl flex items-center justify-between cursor-pointer transition-all {selected ? 'bg-blue-600 shadow-md text-white' : 'hover:bg-slate-100 text-slate-700'}"
     onclick={() => onSelect(index)}
     onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onSelect(index); } }}>
  <div class="flex items-center space-x-3">
    <div class="w-10 h-10 rounded-full flex items-center justify-center font-bold {selected ? 'bg-white/20 text-white' : 'bg-blue-100 text-blue-700'}" aria-hidden="true">
      {student.vorname.charAt(0)}{student.nachname.charAt(0)}
    </div>
    <div>
      <div class="font-bold {selected ? 'text-white' : 'text-slate-900'}">{student.vorname} {student.nachname}</div>
      <div class="text-xs {selected ? 'text-blue-100' : 'text-slate-500'}">
        {student.klasse}
        {#if student.geburtsdatum} · {new Date(student.geburtsdatum).toLocaleDateString('de-DE')}{/if}
        · {student.barcode_id}
      </div>
    </div>
  </div>
</div>
