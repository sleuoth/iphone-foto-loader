# iPhone Foto Loader

Lokales macOS-Tool, das Fotos und Videos von einem per USB angesteckten iPhone auf den Rechner kopiert. Erkennt das iPhone automatisch, wählt dafür einen konfigurierten Zielordner, kopiert nur neue Dateien, erhält alle Metadaten und legt eine JPEG-Kopie neben HEIC-Originalen ab.

## Funktionsumfang

- **Geräteerkennung**: Erkennt angesteckte iPhones anhand ihrer UUID
- **Pro iPhone eigener Zielordner**: Konfigurierbar, bei neuen Geräten interaktive Abfrage
- **Dedup**: SQLite-Status-DB pro Gerät verhindert erneutes Kopieren bereits importierter Dateien
- **Metadaten-Erhaltung**: Originaldateien werden 1:1 byteweise kopiert
- **HEIC zu JPEG**: Kamera-HEICs werden zusätzlich als JPEG konvertiert, Original bleibt erhalten
- **Live Photos**: HEIC+MOV-Paare werden zusammen erkannt und importiert
- **Ordnerstruktur**: `YYYY/` mit `heic/` und `sonstiges/` Unterordnern
- **Kamera vs. Sonstiges**: EXIF-basierte Klassifizierung (Screenshots/WhatsApp-Bilder landen in `sonstiges/`)

## Voraussetzungen

- macOS (getestet auf macOS 26.5.1, Apple Silicon)
- Go 1.21+ (zum Bauen)
- Swift 5.9+ (zum Bauen des Helpers, im Xcode Command Line Tools enthalten)
- Optional: `exiftool` (`brew install exiftool`) für vollständigen Metadaten-Transfer ins JPEG

## Schnellstart

### 1. Bauen

```bash
make all
```

Erzeugt zwei Binaries in `bin/`:
- `bin/iphone-loader` — Go-CLI, wird vom User aufgerufen
- `bin/iphone-ic-helper` — Swift-Helper, spricht mit ImageCaptureCore

### 2. iPhone anschließen

iPhone per USB-Kabel an den Mac anschließen. Auf dem iPhone den Trust-Dialog bestätigen ("Vertrauen" antippen).

### 3. Import starten

```bash
./bin/iphone-loader
```

Beim ersten Aufruf mit einem unbekannten iPhone wird der Zielordner interaktiv abgefragt:

```
Neues iPhone erkannt: "Sebs iPhone" (UUID a1b2c3d4-...)
Zielordner für dieses Gerät: ~/Pictures/iphone-seb
```

Die Auswahl wird in der Config gespeichert — beim nächsten Mal automatisch erkannt.

### 4. Ergebnis prüfen

```
~/Pictures/iphone-seb/Sebs iPhone/
  2026/
    06_27_1234.jpg         # JPEG aus HEIC konvertiert
    06_27_1234.mov         # Live-Photo-Video
    06_27_1235.jpg         # Original-JPEG der Kamera
    heic/
      06_27_1234.heic      # HEIC-Original mit allen Metadaten
    sonstiges/
      06_27_whatsapp-image.jpg   # WhatsApp-Bild (keine Kamera-EXIF)
```

## Architektur

Zwei Binaries aus einem Repo, klare JSON-Schnittstelle dazwischen:

```
iphone-loader (Go)                    iphone-ic-helper (Swift)
┌──────────────────────┐              ┌──────────────────────────┐
│ Config (TOML)        │              │ ImageCaptureCore         │
│ SQLite Status-DB     │  subprocess  │  - identify (JSON)       │
│ EXIF-Klassifizierung │ ──────────>  │  - list (JSON)           │
│ HEIC→JPEG Conversion │  <────────── │  - download (Bytes)      │
│ Routing/Dateinamen   │   JSON/Bytes │                          │
│ CLI/Interaktion      │              │                          │
└──────────────────────┘              └──────────────────────────┘
```

### Warum zwei Binaries?

iPhones mounten auf macOS nicht als Laufwerk. Der einzige robuste, abhängigkeitsfreie Weg an die Dateien ist Apples `ImageCaptureCore`-Framework. Swift spricht dieses nativ; Go uber einen Subprocess-Wrapper.

### Paketstruktur (Go)

| Paket | Verantwortung |
|-------|---------------|
| `internal/config` | TOML-Config laden/speichern, Device-Mapping |
| `internal/helper` | JSON-Typen + Subprocess-Client fur Swift-Helper |
| `internal/db` | SQLite Status-DB (Open, IsImported, Insert) |
| `internal/exif` | exiftool-Wrapper, Kamera-Klassifizierung |
| `internal/routing` | Dateinamen-Prefix (MM_DD_, IMG_-Entfernung), Pfad-Berechnung |
| `internal/convert` | HEIC→JPEG via sips + exiftool |
| `internal/importer` | Single-File-Import, Live-Photo-Paare, Device-Flow-Orchestrierung |
| `internal/cli` | CLI-App: Flags, Config, Interaktion, Device-Processing |

### Swift-Helper

| Datei | Verantwortung |
|-------|---------------|
| `Models.swift` | Codable JSON-Typen (Device, FileEntry, Responses) |
| `Commands.swift` | CLI-Dispatch (identify, list, download) |
| `DeviceManager.swift` | ImageCaptureCore-Integration (ICDeviceBrowser, ICCameraDevice) |
| `main.swift` | Entry Point, Argument-Parsing |

## Konfiguration

Config-Datei: `~/.config/iphone-loader/config.toml`

```toml
[helper]
path = "/pfad/zu/iphone-ic-helper"  # optional; Fallback: neben iphone-loader

[devices."a1b2c3d4-uuid-001"]
name = "Sebs iPhone"
target = "/Users/seb/Pictures/iphone-seb"

[devices."e5f6a7b8-uuid-002"]
name = "Firmen-iPhone"
target = "/Users/seb/Pictures/iphone-firma"
```

Die Config wird automatisch erstellt/erweitert, wenn ein neues iPhone erkannt wird.

## CLI-Optionen

```bash
iphone-loader [flags]
```

| Flag | Standard | Beschreibung |
|------|----------|--------------|
| `-config <pfad>` | `~/.config/iphone-loader/config.toml` | Pfad zur Config-Datei |
| `-target <pfad>` | (leer) | Einmaliger Override des Zielordners (ohne Config-Speicherung) |

Beispiel:

```bash
# Abweichende Config verwenden
./bin/iphone-loader -config ~/meine-config.toml

# Einmalig anderen Ordner verwenden
./bin/iphone-loader -target /tmp/test-import
```

## Dateinamen-Regeln

Originalnamen vom iPhone (`IMG_1234.HEIC`, `whatsapp-image.jpg`) werden transformiert:

1. `IMG_`-Prefix wird entfernt
2. `MM_DD_`-Prefix aus Aufnahmedatum wird vorne angehangen
3. Dateiendung wird kleingeschrieben

| Original | Aufnahmedatum | Zielname |
|----------|---------------|----------|
| `IMG_1234.HEIC` | 27.06.2026 | `06_27_1234.heic` (+ `06_27_1234.jpg`) |
| `IMG_9999.MOV` | 05.01.2026 | `01_05_9999.mov` |
| `whatsapp-image.jpg` | 27.06.2026 | `06_27_whatsapp-image.jpg` |

## Dedup-Strategie

Pro iPhone-Zielordner liegt eine versteckte SQLite-DB (`.iphone-loader.sqlite`). Lookup-Schluessel: `(filename, size)` — Dateiname ist der praefixte Zielname, Size ist die Originalgroße in Bytes.

- Bereits importiert → Datei wird ubersprungen
- Noch nicht importiert → Download + Verarbeitung + DB-Eintrag
- Fehlgeschlagener Import → kein DB-Eintrag → nachster Versuch beim nachsten Lauf

Kein Hashing, kein Dateisystem-Scan. Fängt >99 % der Faelle ab.

## Build-System

### Makefile-Targets

```bash
make all         # Beide Binaries bauen (bin/iphone-loader + bin/iphone-ic-helper)
make build-go    # Nur Go-Binary
make build-swift # Nur Swift-Binary
make test        # Go-Tests ausfuhren
make test-swift  # Swift-Tests ausfuhren
make smoke       # End-to-End-Smoke-Test mit Stub-Helper (kein echtes iPhone notig)
make clean       # Build-Artifacts loschen
make install     # Binaries nach /usr/local/bin installieren
```

### Manueller Build

```bash
# Go
go build -o bin/iphone-loader .

# Swift
cd swift-helper && swift build -c release
cp .build/release/iphone-ic-helper ../bin/
```

## Testen

### Unit-Tests

```bash
make test          # Go
make test-swift    # Swift (Model-Tests)
```

### Smoke-Test (ohne echtes iPhone)

```bash
make smoke
```

Fuhrt den kompletten Go-Flow mit einem Stub-Helper aus, der Fixture-JSON zuruckgibt. Verifiziert:
- Device-Erkennung
- Datei-Listing
- Download (Stub schreibt Fake-Bytes)
- EXIF-Klassifizierung (Kamera vs. sonstiges)
- HEIC→JPEG-Konvertierung (Stub-sips)
- Live-Photo-Paar-Handling
- Ordnerstruktur und Dateinamen

Erwartete Ausgabe:
```
Fertig: 3 neu, 0 ubersprungen, 0 Fehler
```

### Mit echtem iPhone

1. iPhone per USB anschließen
2. Auf dem iPhone "Vertrauen" bestatigen
3. `./bin/iphone-loader` ausfuhren
4. Bei Erstnutzung Zielordner angeben

## Installation

### Systemweit verfugbar machen

```bash
make install
```

Kopiert beide Binaries nach `/usr/local/bin/`. Danach aus jedem Verzeichnis aufrufbar:

```bash
iphone-loader
```

### Eigenes Build-Verzeichnis

Beide Binaries mussen im selben Verzeichnis liegen (oder der Helper-Pfad muss in der Config oder via `IPHONE_IC_HELPER`-Env-Variable gesetzt werden):

```bash
export IPHONE_IC_HELPER=/pfad/zu/iphone-ic-helper
iphone-loader
```

## Fehlersuche

### "Kein iPhone gefunden"

- iPhone per USB angeschlossen?
- Trust-Dialog am iPhone bestatigt?
- `iphone-ic-helper` liegt neben `iphone-loader` (oder Pfad in Config/Env gesetzt)?

### "iPhone nicht vertraut"

iPhone anschließen, Trust-Dialog erscheint, "Vertrauen" + Code eingeben, dann erneut aufrufen.

### JPEG hat keine/vollstandige Metadaten

Ohne `exiftool` nutzt das Tool nur `sips` fur die HEIC→JPEG-Konvertierung. `sips` ubertragt nur Basis-EXIF. Installation:

```bash
brew install exiftool
```

Das HEIC-Original in `heic/` hat immer alle Metadaten — unabhangig von exiftool.

### Download schlagt fehl

Einzelne Datei-Fehler brechen den Lauf nicht ab. Fehlgeschlagene Dateien werden nicht in die DB eingetragen und beim nachsten Lauf erneut versucht. Fehlerursachen stehen in der Zusammenfassung am Ende.

## Entwicklung

### Projektstruktur

```
iphone-foto-loader/
├── main.go                          # Entry Point
├── Makefile                         # Build-System
├── go.mod / go.sum                  # Go-Dependencies
├── internal/                        # Go-Pakete (nicht exportiert)
│   ├── cli/                         # CLI-App
│   ├── config/                      # TOML-Config
│   ├── convert/                     # HEIC→JPEG
│   ├── db/                          # SQLite Status-DB
│   ├── exif/                        # exiftool-Wrapper
│   ├── helper/                      # Swift-Helper-Client
│   ├── importer/                    # Import-Logik
│   └── routing/                     # Dateinamen/Pfade
├── swift-helper/                    # Swift-Helper-Binary
│   ├── Package.swift
│   └── Sources/iphone-ic-helper/
│       ├── main.swift
│       ├── Commands.swift
│       ├── DeviceManager.swift
│       └── Models.swift
├── scripts/
│   ├── smoke.sh                     # Smoke-Test
│   └── stub-helper.sh               # Stub fur Smoke-Test
└── docs/superpowers/
    ├── specs/                       # Design-Spec
    └── plans/                       # Implementierungsplan
```

### Abhangigkeiten

| Bibliothek | Zweck |
|------------|-------|
| `github.com/BurntSushi/toml` | TOML-Config parsen/schreiben |
| `modernc.org/sqlite` | Pure-Go SQLite (kein cgo) |
| `sips` | macOS-Boardmittel, HEIC→JPEG |
| `exiftool` | Optional, Metadaten-Transfer |

### Design-Dokumente

- `docs/superpowers/specs/2026-06-27-iphone-foto-loader-design.md` — Vollstandige Design-Spec
- `docs/superpowers/plans/2026-06-27-iphone-foto-loader.md` — Implementierungsplan mit 20 TDD-Tasks
