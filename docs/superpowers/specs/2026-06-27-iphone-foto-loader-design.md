# iPhone Foto Loader — Design-Spec

Datum: 2026-06-27
Status: Entwurf, wartet auf Freigabe

## Zweck

Lokales macOS-Tool, das Fotos und Videos von einem per USB angesteckten iPhone auf den Rechner kopiert. Erkennt das iPhone, wählt dafür einen konfigurierten Zielordner, kopiert nur neue Dateien, erhält alle Metadaten und legt eine JPEG-Kopie neben HEIC-Originalen ab.

## Kontext & technische Ausgangslage

- Zielplattform: macOS 26.5.1 auf Apple Silicon
- iPhones mounten auf macOS nicht als Laufwerk; sie expose PTP über USB
- Der robuste, abhängigkeitsfreie Weg an die Dateien ist Apples `ImageCaptureCore`-Framework
- Vorhandene Toolchain: Go 1.26.4, Swift (System), `ifuse` (ungenützt), kein `gphoto2`
- `exiftool` ist optional; fehlt es, wird nur `sips` für HEIC→JPEG verwendet (eingeschränkter Metadaten-Transfer)

## Architektur

Zwei Binaries aus einem Repo, klare JSON-Schnittstelle dazwischen.

### `iphone-ic-helper` (Swift-Binary, dumb & stabil)

Einzige Aufgabe: mit dem iPhone über `ImageCaptureCore` sprechen. CLI-Subcommands, antwortet mit JSON auf stdout, Fehler auf stderr + Exit-Code ≠ 0.

- `identify` → `{"devices":[{"uuid":"<stabil>","productName":"iPhone 15 Pro","deviceName":"Sebs iPhone","isTrusted":true}]}`
- `list --device <uuid>` → `{"files":[{"handle":"…","name":"IMG_1234.HEIC","size":1234567,"created":"2026-06-27T…","mimeType":"image/heic","livePhotoPair":"IMG_1234.MOV"}]}` (`livePhotoPair` = `null` wenn kein Live Photo)
- `download --device <uuid> --handle <h> --to <path>` → lädt Rohbytes ans Ziel, Metadaten erhalten (keine Konvertierung)

Wartet mit Timeout auf das Gerät, signalisiert Trust-Status. Write-once-Binary, danach selten angefasst.

### `iphone-loader` (Go-Binary, User-Facing-CLI)

Alles andere:

- Config lesen (TOML): Mapping `device-uuid → Zielordner`
- Helper `identify` aufrufen → verbundene iPhones finden
- Pro bekanntem iPhone: `list` → Dateien aufzählen
- Neue Dateien via Status-DB filtern
- Pro neuer Datei: Zielpfad (YYYY) berechnen, `download` aufrufen, ggf. HEIC→JPEG konvertieren, Status-DB aktualisieren
- Bei unbekanntem iPhone: interaktiv Zielordner abfragen, in Config speichern
- Fortschrittsanzeige + Zusammenfassung am Ende

### Schnittstelle

Helper spricht pro Subcommand ein JSON-Dokument auf stdout. Go ruft ihn als Subprocess auf. Helper-Pfad wird gesucht via Env `IPHONE_IC_HELPER` oder Fallback „gleiches Verzeichnis wie `iphone-loader`".

Beide Binaries werden aus demselben Repo gebaut (Go-Module + Swift Package); Release-Artifact sind beide Binaries nebeneinander.

## Zielordner-Struktur

Pro iPhone-Zielordner:

```
<zielordner>/<iPhone>/
  2026/
    06_27_1234.jpg       # Kamera-JPEG (aus HEIC konvertiert ODER Original-JPEG)
    06_27_1234.mov       # Live-Photo-Video, Kamera
    heic/
      06_27_1234.heic    # Original-HEIC (Kamera) behalten
    sonstiges/
      06_27_whatsapp-image.jpg   # keine Kamera-EXIF → als-is, keine Konvertierung
      06_27_bildschirmfoto.png
  2025/
    ...
```

### Dateinamen-Regel

- Originalname vom iPhone: `IMG_1234.HEIC`, `IMG_9999.MOV`, `whatsapp-image.jpg`, `Bildschirmfoto.png`
- Falls Originalname mit `IMG_` beginnt, wird `IMG_` entfernt
- `MM_DD_`-Präfix aus Aufnahmedatum (EXIF `DateTimeOriginal`, Fallback Dateidatum vom iPhone) vorne dran
- Dateiendung bleibt

Beispiele:
- `IMG_1234.HEIC` mit Aufnahme 27.06.2026 → `06_27_1234.jpg` (JPEG im Hauptordner) + `06_27_1234.heic` (in `heic/`)
- `IMG_1234.MOV` → `06_27_1234.mov`
- `whatsapp-image.jpg` → `06_27_whatsapp-image.jpg`

### Klassifizierung Kamera vs. sonstiges

Nach Download EXIF prüfen (`exiftool -Make -Model`). Datei gilt als Kamera-aufgenommen, wenn `Make` ein Kamera-Hersteller ist (z.B. „Apple"). Sonst → `sonstiges/`. Die `heic/`-Split-Regel greift nur für Kamera-HEICs; in `sonstiges/` wird nichts konvertiert oder gesplittet (dort landen meist ohnehin JPEG/PNG).

### Routing pro Datei

1. Rohbytes via Helper in Staging herunterladen
2. EXIF `Make`/`Model` lesen → Kamera oder sonstiges
3. Routing:
   - **Kamera HEIC** → JPEG konvertieren nach `YYYY/`, HEIC nach `YYYY/heic/`
   - **Kamera JPEG/Video** → direkt nach `YYYY/` (Live-Photo-MOV bleibt daneben)
   - **Sonstiges** → als-is nach `YYYY/sonstiges/`
4. Status-DB-Eintrag

## Status-DB & Dedup

Eine SQLite-DB pro iPhone-Zielordner, liegt unter `<zielordner>/<iPhone>/.iphone-loader.sqlite` (versteckt, gehört zum Archiv).

### Schema

Eine Tabelle `imported`:

| Spalte       | Typ    | Bedeutung                                          |
|--------------|--------|----------------------------------------------------|
| `filename`   | TEXT   | Präfixter Zielname, z.B. `06_27_1234.jpg`         |
| `size`       | INTEGER| Originalgröße in Bytes                             |
| `imported_at`| TEXT   | ISO-Timestamp des Imports                          |
| `target_path`| TEXT   | Relativer Pfad im Archiv, z.B. `2026/06_27_1234.jpg` |

PK über `(filename, size)`.

### Lookup beim Import

Für jede Datei aus `list`: `SELECT 1 FROM imported WHERE filename=? AND size=?`. Treffer → überspringen. Kein Hashing, kein Dateisystem-Scan. Fängt >99 % der Fälle ab.

### Schreiben

Eintrag erst *nach* erfolgreichem Download + Konvertierung + Ziel-Move. Schlägt was dazwischen fehl, bleibt die Datei nicht halb in der DB → beim nächsten Lauf neu versucht.

### Wartung

`iphone-loader verify` (optionales Subcommand) gleicht DB gegen Dateisystem ab (tote Einträge aufräumen), aber nicht Teil des Kern-Flows.

## Config-Format

Zentrale TOML-Datei unter `~/.config/iphone-loader/config.toml`:

```toml
[helper]
path = "/usr/local/bin/iphone-ic-helper"  # optional; Fallback: neben iphone-loader

[devices."a1b2c3d4-..."]
name = "Sebs iPhone"
target = "/Users/seb/Pictures/iphone-seb"

[devices."e5f6a7b8-..."]
name = "Firmen-iPhone"
target = "/Users/seb/Pictures/iphone-firma"
```

### Bei unbekanntem iPhone

UUID nicht in Config → Go fragt interaktiv:

```
Neues iPhone erkannt: "Sebs iPhone 15 Pro" (UUID a1b2c3d4-...)
Zielordner für dieses Gerät: _
```

Eingabe validieren (existiert / anlegbar, schreibbar), dann Block in Config schreiben und weiter.

### CLI-Overrides (optional)

- `--config <pfad>` für abweichende Config
- `--target <pfad>` als einmaliger Override für ein einzelnes iPhone (ohne Config zu schreiben)

## Import-Flow & Laufzeitverhalten

### End-to-End-Ablauf eines `iphone-loader`-Aufrufs

1. Config laden, Helper-Pfad auflösen
2. `iphone-ic-helper identify` → verbundene iPhones
   - Keines da → Meldung + Exit 0
   - Gerät untrusted (Trust-Dialog am iPhone offen) → Hinweis „auf dem iPhone ‚Vertrauen' antippen" + Exit 1
3. Pro Gerät:
   a. UUID in Config? Nein → interaktiv Zielordner abfragen, speichern
   b. `helper list --device <uuid>` → Dateiliste
   c. Status-DB öffnen (`<target>/<name>/.iphone-loader.sqlite`, ggf. anlegen)
   d. Neue Dateien = Liste minus DB-Treffer
   e. Sortiert nach Aufnahmedatum (älteste zuerst)
   f. Pro Datei: Download → EXIF-Klassifizierung → Routing/Konvertierung → DB-Eintrag
4. Zusammenfassung: `<n> neu, <m> übersprungen, <k> Fehler`, pro Fehler eine Zeile mit Datei + Grund

### Konvertierung

`sips` (macOS-Boardmittel) für HEIC→JPEG. Metadaten-Transfer via `exiftool`, falls installiert; fehlt `exiftool`, Warnung + JPEG ohne vollständige Metadaten (HEIC-Original in `heic/` bleibt aber immer mit allen Metadaten erhalten — Qualitätsverlust nur für die JPEG-Kopie).

### Fehlerverhalten

- Einzelne Datei-Fehler brechen nicht den Lauf
- Fehlgeschlagene Datei wird nicht in DB eingetragen → nächstes Mal erneut versucht
- Abbruch nur bei kritischen Fehlern (Helper nicht gefunden, kein Schreibzugriff auf Ziel, Geräte-Verlust mitten drin)

### Fortschrittsanzeige

Pro Datei eine Zeile: `[42/187] IMG_1234.HEIC → 2026/heic/06_27_1234.heic (+ 2026/06_27_1234.jpg)`. Bei Konvertierungen beide Zielpfade sichtbar.

### Gleichzeitige Dateien

Eine Datei nach der anderen (kein Parallel-Download). Erstmal robust statt schnell; kann bei Bedarf später geändert werden.

## Error-Handling, Metadaten, Testing

### Trust & Geräteprobleme

- iPhone angesteckt aber nicht „Vertrauen" getippt → `isTrusted:false` in `identify`, Go bricht mit klarem Hinweis ab, kein Warten/Polling
- Gerät verschwindet mitten im Import → aktueller Download abgebrochen, Datei nicht in DB → nächstes Mal neuer Versuch, Restübersicht dieses Laufs
- Helper-Crash (Exit ≠ 0 / kein valides JSON) → Fehler pro Subcommand, Lauf für dieses Gerät abbrechen, andere Geräte weiterlaufen lassen

### Metadaten-Erhaltung

- Originaldateien (HEIC, MOV, MP4, JPEG-Kamera) werden 1:1 byteweise vom Helper geschrieben → Metadaten erhalten
- JPEG-Kopie: `exiftool -tagsFromFile HEIC -all:all` kopiert vollständige EXIF/IPTC/XMP. Ohne `exiftool`: `sips` allein kopiert nur Basis-EXIF, Rest geht verloren → Warnung an User, HEIC-Original bleibt aber vollständig in `heic/`
- Live-Photo-Paar: `livePhotoPair` aus `list` beachten — beide Dateien zusammen downloaden, zusammen in DB eintragen (atomar: beide Erfolg nötig, sonst keiner)

### Testing-Strategie

- **Swift-Helper**: schwierig zu unittesten (ImageCaptureCore braucht echtes Gerät). Fokus auf JSON-Schema-Tests mit Fixtures (gelistete Dateien als JSON-Datei, Helper-Logik parst & serialisiert). Integrationstest manuell mit echtem iPhone.
- **Go-CLI**: echte Unit-Tests für alles Testbare — Config-Parsing, DB-CRUD, EXIF-Klassifizierung (Mock-`exiftool`-Output), Pfad-Routing-Logik (Kamera vs. sonstiges, HEIC-Split), Live-Photo-Paar-Logik. Helper-Aufrufe über Interface mocken.
- **Smoke-Test-Script** im Repo: `scripts/smoke.sh` — mit Dummy-Helper-JSON (Stub-Binary, das Fixtures zurückgibt) den ganzen Go-Flow durchspielen ohne echtes iPhone.

### Edge Cases (abgedeckt)

- IMG_xxxx-Namenskollision über 9999 → Dateinamen-Präfix mit Datum macht Kollisionen unwahrscheinlich; DB sichert den Rest (filename+size)
- Konvertierung schlägt fehl → HEIC-Original nicht verschoben, DB-Eintrag aus, nächste Datei weiter
- Zielordner nicht schreibbar → früher Abbruch mit klarem Hinweis
- gleiche Datei auf zwei iPhones → pro Gerät eigene DB, wird zweimal archiviert (gewollt)
