import { execSync } from 'node:child_process';
export const q = (sql) => execSync(`docker exec -i bibliothek-db-demo psql -U postgres -d bibliothek -tA -v ON_ERROR_STOP=1`, { input: sql }).toString().trim();
export function demoDaten() {
  const D = {};
  D.S_OK = q(`select s.barcode_id from schueler s where not s.ist_gesperrt and not s.ist_abgaenger
    and (select count(*) from ausleihen a where a.schueler_id=s.id and a.rueckgabe_am is null)=1
    and not exists (select 1 from schadensfaelle d where d.schueler_id=s.id)
    and not exists (select 1 from vormerkungen v where v.schueler_id=s.id)
    and not exists (select 1 from ausleihen a where a.schueler_id=s.id and a.rueckgabe_am is null and a.rueckgabe_frist<now())
    order by s.barcode_id limit 1`);
  const frei = (extra, n) => q(`select e.barcode_id from buecher_exemplare e join buecher_titel t on t.id=e.titel_id
    where not e.ist_ausgesondert and e.ist_ausleihbar and t.titel not like 'LMF%'
    and not exists (select 1 from ausleihen a where a.exemplar_id=e.id and a.rueckgabe_am is null) ${extra}
    order by t.titel, e.barcode_id limit ${n}`).split('\n');
  const ohneVormerkung = frei(`and not exists (select 1 from vormerkungen v where v.titel_id=t.id)`, 40);
  // drei verschiedene Titel
  const seen = new Set(); D.B_FREE = [];
  for (const b of ohneVormerkung) { const t = q(`select t.titel from buecher_exemplare e join buecher_titel t on t.id=e.titel_id where e.barcode_id='${b}'`); if (!seen.has(t)) { seen.add(t); D.B_FREE.push(b); } if (D.B_FREE.length >= 3) break; }
  D.B_RESERVED = frei(`and exists (select 1 from vormerkungen v where v.titel_id=t.id and v.status='wartend')`, 1)[0];
  D.B_RESERVED_TITEL = q(`select t.titel from buecher_exemplare e join buecher_titel t on t.id=e.titel_id where e.barcode_id='${D.B_RESERVED}'`);
  D.S_OVERDUE_NAME = q(`select s.vorname||' '||s.nachname from ausleihen a join schueler s on s.id=a.schueler_id where a.rueckgabe_am is null and a.mahnstufe=2 and not s.ist_gesperrt limit 1`);
  D.S_LOCKED = q(`select barcode_id from schueler where ist_gesperrt order by barcode_id limit 1`);
  D.S_FEE_NAME = q(`select s.nachname from schadensfaelle d join schueler s on s.id=d.schueler_id where not d.ist_bezahlt and not s.ist_abgaenger limit 1`);
  D.INV = q(`select e.barcode_id from buecher_exemplare e join buecher_titel t on t.id=e.titel_id where t.signatur like 'Jug %' and not e.ist_ausgesondert and not exists (select 1 from ausleihen a where a.exemplar_id=e.id and a.rueckgabe_am is null) order by e.barcode_id limit 3`).split('\n');
  return D;
}
