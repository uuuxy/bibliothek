<!-- @component BackupStatusBadge — Wächter für die nächtlichen Datenbank-Backups.
     Der Backup-Job überspringt sich still, wenn BACKUP_ENCRYPTION_KEY fehlt;
     dieses Badge macht das (und veraltete Backups) in der Admin-Sidebar
     unübersehbar. ok = dezente Bestätigungszeile, warning/critical = Alert. -->
<script>
  import { onMount } from "svelte";
  import { apiFetch } from "../../apiFetch.js";

  /** @type {{ collapsed?: boolean }} */
  let { collapsed = false } = $props();

  /** @type {{ last_backup_at: string | null, encryption_key_set: boolean, status: 'ok'|'warning'|'critical' } | null} */
  let status = $state(null);

  onMount(async () => {
    try {
      const res = await apiFetch("/api/admin/system/backup-status");
      if (res.ok) status = await res.json();
    } catch { /* Netzfehler: kein Badge statt falscher Entwarnung */ }
  });

  const message = $derived.by(() => {
    if (!status) return "";
    if (!status.encryption_key_set) return "Backup-Verschlüsselungs-Key fehlt!";
    if (!status.last_backup_at) return "Noch kein Backup vorhanden!";
    const last = new Date(status.last_backup_at);
    const ageH = (Date.now() - last.getTime()) / 3600000;
    if (status.status === "ok") {
      const heute = new Date().toDateString() === last.toDateString();
      const zeit = last.toLocaleTimeString("de-DE", { hour: "2-digit", minute: "2-digit" });
      return heute ? `Letztes Backup: heute ${zeit}` : `Letztes Backup: ${last.toLocaleDateString("de-DE")} ${zeit}`;
    }
    const tage = Math.floor(ageH / 24);
    return tage >= 1 ? `Seit ${tage === 1 ? "über 1 Tag" : `${tage} Tagen`} kein Backup!` : `Letztes Backup vor ${Math.round(ageH)} Stunden!`;
  });
</script>

{#if status}
  {#if collapsed}
    <!-- Eingeklappte Sidebar: nur bei Problemen ein unübersehbarer Punkt -->
    {#if status.status !== "ok"}
      <div class="flex justify-center py-2" title="⚠️ {message}">
        <span class="w-2.5 h-2.5 rounded-full animate-pulse {status.status === 'critical' ? 'bg-rose-500' : 'bg-amber-500'}"></span>
      </div>
    {/if}
  {:else if status.status === "ok"}
    <div class="px-4 py-2 flex items-center gap-1.5 text-[10px] font-semibold text-emerald-700">
      <svg class="w-3 h-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" /></svg>
      {message}
    </div>
  {:else}
    <div class="mx-3 my-2 px-3 py-2.5 rounded-xl border text-xs font-bold flex items-start gap-2
                {status.status === 'critical' ? 'bg-rose-50 border-rose-200 text-rose-700' : 'bg-amber-50 border-amber-200 text-amber-800'}"
         role="alert">
      <span class="shrink-0">⚠️</span>
      <span>{message}</span>
    </div>
  {/if}
{/if}
