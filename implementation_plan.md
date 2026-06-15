# Implementierungsplan: Wareneingang und Etiketten-Management

Das Ziel ist es, den finalen Workflow für den Wareneingang abzuschließen und eine Reparatur-Funktion für Fremdbestands-Etiketten hinzuzufügen.

## Proposed Changes

### Backend (Go)

#### [MODIFY] api/routes_orders.go
- Registrierung der bereits vorhandenen Route `ReleaseOrdersHandler` als `POST /api/orders/release`.

#### [MODIFY] api/barcode.go
- Implementierung eines neuen Handlers `NextBarcodeHandler` (`GET /api/barcode/next`).
  - Sucht in der Datenbank nach dem höchsten vergebenen Barcode im Format `B-XXXXX`.
  - Gibt die nächste freie Nummer als JSON zurück (z. B. `{"next_barcode": "B-10042"}`).

#### [MODIFY] api/routes_system.go
- Registrierung der neuen Route `GET /api/barcode/next`.
- Registrierung einer neuen Route `GET /api/print/etikett/{id}` für den Einzel-Etikettendruck.

#### [MODIFY] api/order_pdf.go (oder api/print.go)
- Implementierung der Funktion `GenerateSingleLabelPDFA6(label BarcodeLabelDetail) ([]byte, error)`:
  - Erzeugt ein DIN-A6 PDF-Dokument (im Hoch- oder Querformat, passend für einen Label-Drucker).
  - Verwendet die gleiche Optik wie die Naacher-Etiketten (Titel, Autor, QR-Code, lesbarer Barcode-Text).
- Implementierung des `PrintErsatzEtikettHandler`:
  - Liest die Daten des Exemplars aus der Datenbank aus.
  - Ruft `GenerateSingleLabelPDFA6` auf.
  - Gibt das PDF zum direkten Download/Druck zurück.

---

### Frontend (Svelte)

#### [MODIFY] frontend/src/lib/components/bestellungen/IncomingShipments.svelte
- Integration eines auffälligen Buttons **"Lieferung freigeben (Naacher)"**.
- Ruft `POST /api/orders/release` über den `apiClient` auf.
- Zeigt bei Erfolg eine Erfolgsmeldung an (z. B. "Lieferung freigegeben. X Exemplare sind nun im aktiven Bestand.").
- Löst ein Update der UI-Status aus (die Exemplare sind danach ausleihbar).

#### [MODIFY] frontend/src/lib/BookExemplareTab.svelte
- **Interne ID generieren:** Neben dem Eingabefeld für den Barcode (im Bearbeitungsmodus) wird ein Button "Interne ID generieren" platziert. Ein Klick ruft `GET /api/barcode/next` auf und füllt das Feld.
- **Ersatz-Etikett drucken:** Wenn das Exemplar einen Barcode besitzt, der mit `B-` beginnt, wird ein Button "Ersatz-Etikett drucken" eingeblendet.
- Ein Klick auf diesen Button öffnet das vom Backend generierte DIN-A6 PDF (`/api/print/etikett/{id}`) in einem neuen Tab zum Drucken.

## User Review Required

> [!IMPORTANT]
> **Format für das Ersatz-Etikett:** Für den Einzeldruck (Ersatz-Etikett) soll ein DIN-A6 PDF erstellt werden. Soll dieses Layout identisch zu einem einzelnen Kästchen auf dem DIN-A4 Bogen der Naacher-Bestellungen sein, oder soll das Label den gesamten DIN-A6 Platz einnehmen? Aktuell plane ich, es relativ groß und zentriert auf dem A6-Format zu platzieren.

> [!NOTE]
> Die Route für den "blind freigeben" Button (`/api/bestellungen/freigeben`) wird in der UI durch den neuen Naacher-Freigabe Workflow (`/api/orders/release`) ersetzt bzw. ergänzt. Ich werde den neuen Button neben den bestehenden platzieren oder den alten ersetzen, falls gewünscht.

## Verification Plan

### Manual Verification
1. Öffnen des Bestell-Workspaces. Prüfen, ob der neue Freigabe-Button vorhanden ist und ordnungsgemäß `POST /api/orders/release` aufruft.
2. In der Exemplar-Ansicht eines Buches: Klick auf "Interne ID generieren" prüfen (Feld muss mit nächstem `B-XXXXX` Wert befüllt werden).
3. Für ein Exemplar mit `B-XXXXX` Barcode auf "Ersatz-Etikett drucken" klicken und sicherstellen, dass das PDF im A6-Format im Browser angezeigt wird.
