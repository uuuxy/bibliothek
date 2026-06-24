# 🎨 Flat & Edge-to-Edge Sweep — Abhakliste

> Ziel: Karten-/Kachel-Anti-Pattern auf **Layout-Blöcken** beseitigen → flach auf
> Seitenhintergrund, edge-to-edge (`max-w-6xl/7xl`/`w-full`), Trennung durch
> `border-b border-gray-200` + großzügigen Abstand, Labels `text-sm font-medium
> text-gray-600`, wichtige Werte/Felder hochskaliert (`text-lg`/`text-xl`).
> Regeln: DRY (Snippets), ≤200 Zeilen/Datei, **nur UI** (keine Logik/API/State).

## Konventionsregeln (Scope-Abgrenzung)
- **KONVERTIEREN**: Seiten-/Sektions-Karten, die Formulare/Tabellen/Infoblöcke umschließen.
- **BEWAHREN** (kein Anti-Pattern): Modals/Overlays, Toasts, Dropdowns, echte
  Cover-Galerie-Kacheln (Flachlegen einer Cover-Galerie ist eine separate, riskante
  UX-Entscheidung — vor Umbau klären).

## ✅ Erledigt (verifiziert, Build grün)
- [x] `lib/MailTemplates.svelte` — doppelte Karten (bg-white rounded-3xl shadow + bg-slate-50 rounded-2xl) entfernt; max-w-6xl; Labels text-sm/medium; Betreff text-lg, Textarea text-base; Platzhalter als Links-Akzent.
- [x] `lib/OverdueWidget.svelte` — Alert-Karte → flaches Links-Akzent-Alert; Header-Kästchen raus; Label hochskaliert.

## 🔒 Bewahren (bewusst NICHT angefasst)
Modals: `DamageReportModal`, `KioskChecklistModal`, `KioskDamageModal`, `KioskReservationModal`, `StrichcodeScannerOverlay` · Toasts/Indikatoren: `OfflineIndicator` · Cover-Galerien/Kacheln (Design-Frage offen): `BuchKarte`, `KlassenBuchKachel(Startseite)`, `KlassenUebersicht(Startseite)`, `OpacSearch`-Trefferraster.

## ✅ Bestellungen — erledigt (Batch 2)
- [x] `OrderCreationPanel` — Panel-Karte → flach, Labels uppercase→text-sm/medium, Felder bg-white, Titel text-lg, Staging-Block → Links-Akzent ohne Schatten.
- [x] `OrderRecommendations`, `IncomingShipments` — Panel-Karten → flach, Header via border-b.
- [x] `SupplierManager` — beide Karten flach, Labels + Tabellenkopf entkapitalisiert.
- [x] `BestellWorkspace` — war auf Layout-Ebene bereits flach (Tableiste/PDF-Button = Controls, bleiben).

## ✅ Schüler-Listen & Audit — erledigt (Batch 3)
- [x] `ActiveStudentList` — Tabellen-Karte → flach; Tabellenkopf entkapitalisiert.
- [x] `DeletedStudentList` — Karte → flach mit rosa Links-Akzent (Papierkorb-Identität); Header/Kopf entkapitalisiert.
- [x] `AuditLog`, `AdminAuditLog` — Tabellen-Karten → flach; Köpfe entkapitalisiert.
- [x] `MahnwesenTable` — Tabelle war bereits flach; nur Modal (bleibt Karte).

## ✅ Buch-Akte-Tabs — erledigt (Batch 4)
- [x] `BookAkteMeta` — Karten-Chrome (bg-white border shadow) entfernt; Stat-Eyebrows entkapitalisiert. (Stat-Chips bg-slate-50 bewusst belassen = Metrik-Chips, keine Layout-Karte.)
- [x] `BookHistoryTab`, `BookBorrowersTab`, `BookBorrowersList` — Tabellen-/Panel-Karten → flach; Köpfe/Labels entkapitalisiert.
- [x] `BookVormerkungenTab` — Eingabe-Labels uppercase → text-sm font-medium text-gray-600.
- [~] `BookExemplareTab` — Exemplar-Auswahl-Kacheln sind interaktive Auswahl-Items (kein Layout-Block); nur Schatten reduzierbar. >200 Zeilen (296) → Split offen.

## ✅ Dashboards/Portale & Sonstiges — erledigt (Batch 5)
- [x] `StatsDashboard` — KPI-Karten → flach (Trennung via Grid + dezente Spaltenlinie); Tabellen-Container flach.
- [x] `LehrerPortal` — Tabellen-Karte → flach.
- [x] `UnifiedInventory` — Empty-State- und Panel-Karte → flach (Scan-Input bleibt).
- [x] `StudentVormerkungenCard` — Item-Karte → flache Zeile (border-b).
- [x] `ClassPrintStation` — Panel-Karte → flach.
- [x] `LabelSettings` — Info-Panel → Links-Akzent; weiße Panels → flach (Dropdown bleibt).
- [~] `KioskIdle`/`KioskActiveSession` — bewusst belassen: Self-Service-Vollbild-Fokuselement (anderer Kontext, wie der Ausweis-Render).

## ✅ Inventur-Modul — erledigt (Batch 6)
- [x] `routes/settings/+page` — 3 Panel-Karten → flache Sektionen (border-b).
- [x] `routes/scanner/+page` — Panel-Karte → flach.
- [x] `admin/BookTable` — Tabellen-Karte → flach.
- [x] `admin/BuchFormular` — Container-Karte → flach.
- [~] `StartseitenFilter`, `StartseitenHeaderJahrgaenge/Klassen`, `admin/KlassenUebersicht` — Treffer sind **Selects/Inputs/Dropdowns** (Form-Controls, kein Layout-Block) → bewahrt.

## 🔒 Bewahrt (bewusst, kein Layout-Anti-Pattern)
- `designer/PropertiesPanel` — schwebendes Inspector-Panel (shadow-xl, max-h, scroll) wie ein Modal.
- `OpacSearch` — öffentliche Cover-Galerie (Trefferraster bleibt; Galerie-Flachlegung ist separate UX-Entscheidung).
- Galerie-Kacheln: `BuchKarte`, `KlassenBuchKachel(Startseite)`, `KlassenUebersichtStartseite`.

## Hinweis 200-Zeilen-Regel
Visueller Sweep priorisiert. Verbleibende >200-Zeilen-Dateien (Omnibox, Graduates,
BookExemplareTab, settings/+page, BestellWorkspace, PermissionManager …) für
separate Split-Refactorings vorgemerkt — Logiknähe macht Aufteilung riskant, daher
nicht im selben Schritt wie die rein-visuelle Umstellung.

## 📏 Zusätzlich: Dateien > 200 Zeilen (Split nötig)
`inventur/routes/settings/+page` (304), `Omnibox` (299), `Graduates` (297), `BookExemplareTab` (296), `inventur/routes/+page` (287), `BestellWorkspace` (275), `PermissionManager` (263), u. a. — beim jeweiligen Bereichs-Batch mit Snippets/Subkomponenten aufteilen.
