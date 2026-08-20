# assetagent: Product Vision und MVP-to-Market-Plan

**Stand:** 1. August 2026  
**Ausgangspunkt:** funktionierender lokaler Prototyp mit Sparkassen-CSV, Transaktionssuche und tool-grounded Chat  
**Ziel:** aus dem technischen Demo ein Produkt mit wiederkehrendem, bezahlbarem Nutzwert machen

## Executive Decision

`assetagent` sollte **nicht** als „ChatGPT für Bankumsätze“ positioniert werden. Diese Kategorie ist bereits austauschbar: Finanzguru und Outbank decken in Deutschland Konten, Analysen und Budgets ab; Actual Budget und MoneyMoney besetzen Privacy/Local-first; Cleo besetzt conversational finance; und ChatGPT bietet in den USA inzwischen verknüpfte Finanzkonten, Dashboards und grounded Q&A.

Die beste Wedge lautet stattdessen:

> **Der private Finanz-CFO für deutsche Haushalte: Er macht aus Kontoumsätzen jeden Monat einen belastbaren Plan – lokal, nachvollziehbar und mit Euro-genauen Handlungsvorschlägen.**

Das Produkt verkauft nicht „Antworten“, sondern drei Ergebnisse:

1. **Klarheit:** Was ist wirklich frei verfügbar, nachdem monatliche und unregelmäßige Kosten korrekt berücksichtigt wurden?
2. **Voraussicht:** Was passiert in den kommenden 90 Tagen und welche Entscheidung ist finanzierbar?
3. **Umsetzung:** Welche eine Maßnahme verbessert die Lage messbar und wurde sie später tatsächlich umgesetzt?

Der erste kommerzielle Kern ist ein **Monthly Money Review**. Chat bleibt die primäre UX; der Review, die Entscheidung und ihr gemessener Effekt bilden jedoch den wiederkehrenden Produkt-Loop.

## 1. Ehrliche Bewertung des heutigen Produkts

### Was bereits wertvoll ist

- Die Antworten sind tool-grounded und verlinken auf Transaktionen. Das ist bei Finanzdaten wichtiger als eloquente Freitextantworten.
- Der Nutzer kann natürlich fragen, statt Filter und Reports erlernen zu müssen.
- Local-first und ein optional lokales Modell schaffen einen glaubwürdigen Vertrauensanker.
- Go, Postgres, OpenAPI, React und Langfuse ergeben eine solide technische Lern- und Produktbasis.
- Die Sparkassen-CSV ist eine konkrete Deutschland-Wedge und kein abstraktes „Wealth OS“.

### Warum das noch kein bezahlbares Produkt ist

- CLI-Import verhindert Aktivierung außerhalb technischer Nutzer.
- Die drei Tools erklären hauptsächlich die Vergangenheit. Sie erzeugen noch keine Entscheidung oder fortlaufende Verhaltensänderung.
- Ohne Kategorien, Transfers, Merchant-Normalisierung und wiederkehrende Zahlungen ist „wahre monatliche Belastung“ nicht zuverlässig berechenbar.
- Es gibt keinen wiederkehrenden Grund, nächsten Monat zurückzukommen.
- Es gibt keine Produkt-Memory für Ziele, Entscheidungen und bereits umgesetzte Maßnahmen.
- Ein generischer Chat-Verlauf würde diese Lücken nicht lösen.

**Produktdiagnose:** Die technische Machbarkeit ist validiert. Nicht validiert sind Problemintensität, Zahlungsbereitschaft und Retention.

## 2. Markt und Konkurrenz

| Kategorie | Starke bestehende Position | Konsequenz für assetagent |
| --- | --- | --- |
| Deutsches Multibanking | Finanzguru bietet Konten, Vertragserkennung, Prognosen, Budgets und Analysen; Outbank kostet für Privatkunden 3,99 €/Monat. | Übersicht und einfache Analyse allein tragen keinen Premiumpreis. |
| Local/Privacy | MoneyMoney speichert Kontodaten verschlüsselt lokal; Actual Budget ist explizit local-first, Open Source und unterstützt CSV, CAMT, OFX/QFX und Bank-Sync. | „Lokal“ ist wichtig, aber allein kein Burggraben. |
| Conversational finance | Cleo baut Marke und Nutzung um eine dialogorientierte Finanzfigur und Zielerreichung. | Ein Chatfenster ist UX, keine Positionierung. |
| General-purpose AI | ChatGPT Finances verbindet in den USA Konten, kategorisiert Daten, kennt Ziele und beantwortet komplexe Finanzfragen. | Eine reine LLM-Hülle wird von Plattformen absorbiert. |
| Vollständige PFM-Suites | Monarch und Copilot kombinieren Budgets, Cashflow, Investments, Net Worth und Haushaltsfunktionen. | Ein Feature-Rennen gegen komplette Suiten ist für ein Ein-Personen-Projekt falsch. |

### Der freie Platz

Die glaubwürdige Lücke liegt in der Kombination aus:

- deutscher Transaktionsrealität und Importformaten,
- lokalem Daten-Vault,
- deterministisch berechneten Finanzfakten,
- konkreten Haushaltsentscheidungen statt Dashboard-Pflege,
- nachvollziehbaren Belegen,
- einem monatlichen Review- und Action-Loop.

Das ist schmaler als „Wealth Platform“, aber deutlich leichter kaufbar.

## 3. Ideal Customer Profile

### Primärer ICP: der finanzkompetente, aber zeitknappe Haushalt

- Deutschland, grob 30–50 Jahre
- zwei Einkommen oder mehrere Konten
- überdurchschnittliches verfügbares Einkommen
- mindestens eine echte Planungsfrage: Immobilie/Kredit, Familienplanung, Sabbatical, Einkommenswechsel, größere Anschaffung oder Sparziel
- nutzt heute Banking-Apps plus Excel oder überschlägt vieles im Kopf
- möchte Kontrolle, aber kein tägliches Envelope-Budget pflegen
- ist bereit, für Datenschutz, Zeitersparnis und eine belastbare Entscheidung zu zahlen

Der emotionale Job lautet nicht „zeige mir meine Ausgaben“, sondern:

> „Sag mir, was wir uns sicher leisten können, ohne dass ich drei Konten, jährliche Kosten und alte Excel-Dateien selbst zusammenführen muss.“

### Nicht der erste ICP

- Menschen in akuter Überschuldung: hoher Beratungs- und Sicherheitsbedarf, geringe Zahlungsfähigkeit
- aktive Trader: benötigt Portfolio-, Markt- und regulatorischen Scope
- Steuer- oder Buchhaltungskunden: anderes Datenmodell und Haftungsprofil
- reine Self-hosting-Hobbyisten: gute Alpha-Tester, aber schlechte Referenz für Mainstream-Zahlungsbereitschaft
- Nutzer, die zwingend automatische Bank-Synchronisierung erwarten: erst nach Product-Market-Signal adressieren

### Sekundärer ICP nach B2C-Signal

Unabhängige Finanzcoaches oder Finanzierungsberater könnten später private, client-owned Reviews sponsern. Das kann den ARPU erhöhen, sollte aber noch nicht zu Mandantenverwaltung, Berater-Dashboard oder Mehrmandanten-SaaS führen.

## 4. Das erste kaufbare Produkt

### First Review

Nach Import von idealerweise 12 Monaten Transaktionen erhält der Nutzer innerhalb von fünf Minuten:

1. verlässliches monatliches Nettoeinkommen,
2. echte Fixkosten inklusive auf den Monat umgelegter Jahreszahlungen,
3. durchschnittliche variable Ausgaben,
4. realistische freie Spar-/Investitionskapazität,
5. erkannte Abos, Preissteigerungen und Anomalien,
6. 90-Tage-Liquiditätsvorschau,
7. eine priorisierte Maßnahme mit geschätztem Euro-Effekt,
8. Evidence Links auf alle zugrunde liegenden Buchungen.

Danach fragt der Copilot nach einem Ziel, zum Beispiel:

- „Können wir eine Kreditrate von 2.000 € dauerhaft tragen?“
- „Wie viel können wir bis Juli nächsten Jahres ansparen?“
- „Was ändert sich, wenn ein Einkommen sechs Monate ausfällt?“
- „Welche drei wiederkehrenden Kosten sind ohne großen Lebensqualitätsverlust reduzierbar?“

### Der Kern-Loop

1. **Importieren/aktualisieren**
2. **Review prüfen und zwei bis fünf Unsicherheiten bestätigen**
3. **eine Entscheidung oder Maßnahme wählen**
4. **im nächsten Monat Ergebnis verifizieren**
5. **Plan an neue Realität anpassen**

Chat unterstützt jeden Schritt, ersetzt aber nicht den strukturierten Zustand.

### North-star Metric

**Completed Money Reviews with a confirmed action** pro Monat.

Chat-Nachrichten, Tokenverbrauch und importierte Transaktionen sind Aktivitätsmetriken, aber kein Nutzwert.

## 5. Was als Nächstes gebaut wird

### Phase A – Productizable Alpha (2 Wochen)

**Ziel:** Ein nichttechnischer Nutzer kommt ohne CLI zum ersten korrekten Ergebnis.

- In-App-Dateiupload mit Drag-and-drop
- Importvorschau vor dem Speichern
- Kontoauswahl/-benennung und Zeitraumserkennung
- verständliche Fehler pro Zeile
- Import rückgängig machen
- Import-Run als eigene Entität mit Dateiquelle, Zeitraum und Qualitätsstatus
- Sparkassen-Fixtures aus mehreren realen Exportvarianten
- Onboarding mit drei Beispiel-Fragen

**Exit Gate:** 90 % der eingeladenen Sparkassen-Nutzer schaffen den Import ohne Hilfe; mindestens 98 % valider Zeilen werden korrekt übernommen.

### Phase B – Trusted Money Model (3 Wochen)

**Ziel:** Die Zahlen bilden den Haushalt ab, nicht nur einzelne Buchungen.

- Merchant-/Counterparty-Normalisierung
- hierarchische Kategorien mit Konfidenz
- Korrektur-Queue statt Vollautomatisierung bei Unsicherheit
- nutzerspezifische Kategorisierungsregeln
- Erkennung interner Transfers, damit Ausgaben nicht doppelt zählen
- Erkennung wiederkehrender Zahlungen, Intervalle und Preisänderungen
- Trennung fix, variabel, einmalig und Einkommen
- `get_cashflow_v2`, `get_recurring_costs`, `get_spending_changes`, `get_anomalies`
- deterministische Geldberechnungen außerhalb des LLM

**Exit Gate:** Eine manuell geprüfte Golden Dataset Suite reproduziert Einkommen, Ausgaben, Transfers und wiederkehrende Kosten ohne numerische LLM-Berechnung.

### Phase C – Outcome MVP (3 Wochen)

**Ziel:** Aus Daten entsteht eine Entscheidung.

- generierter Monthly Money Review als persistentes Artefakt
- 90-Tage-Cashflow-Prognose mit transparenten Annahmen
- deterministische Scenario Engine
- Ziele und zeitgebundene Verpflichtungen als strukturierte Facts
- Decision Ledger: Entscheidung, Annahmen, Zielwert, Datum, späteres Ergebnis
- bestätigbare Action Card mit Euro/Jahr-Schätzung
- Data-Freshness- und Confidence-Indikatoren
- Review-Historie statt generischer ChatGPT-Session-Kopie

**Exit Gate:** Mindestens 10 von 20 Testhaushalten bestätigen eine relevante Erkenntnis oder Entscheidung; mindestens 5 setzen eine Action um.

### Phase D – Paid Beta (4 Wochen)

**Ziel:** Zahlungsbereitschaft und Wiederkehr beweisen.

- Ein-Klick-Installation oder Desktop-Paket; Docker darf nicht Voraussetzung für den Zielkunden bleiben
- einfacher Kauf-/Lizenzfluss
- klare lokale und optionale Cloud-Modell-Einstellung
- Lösch-, Export- und Backup-Funktion
- Feedback direkt an Review, Zahl und Action
- Langfuse-Evalset mit mindestens 50 kanonischen Finanzfragen und erwarteten Tool-Ergebnissen
- Fehlertelemetrie ohne Transaktionsinhalte
- Founding-Customer-Onboarding

**Exit Gate:** 10 zahlende Kunden, mindestens 40 % führen im Folgemonat einen zweiten Review durch, und weniger als 10 % der Reviews enthalten eine gemeldete wesentliche Zahlenabweichung.

## 6. Was bewusst ignoriert wird

Bis zu den Paid-Beta-Gates werden nicht gebaut:

- PSD2/Open-Banking-Sync
- Trading oder Investmentempfehlungen
- vollständiges Net-Worth-/Portfolio-System
- Steuerberechnung oder Buchhaltung
- native Mobile Apps
- beliebiges SQL oder BI-Dashboard-Builder
- Zahlungsausführung oder autonomes Handeln
- Social Feed, Gamification oder Finfluencer-Features
- Multi-User-/Mandanten-SaaS
- zehn LLM-Provider oder weitere Model-Selector-Komplexität
- vollständige ChatGPT-artige Sessionverwaltung
- Beraterportal und White Label

Auch Auth ist bei einer lokalen Einzelplatzanwendung nicht P0. Wird zunächst eine Cloud-Beta gewählt, sind Auth, Verschlüsselung, Mandantentrennung und Security-Basics hingegen Startvoraussetzung und kein späteres Feature.

## 7. Local Privacy versus Cloud Convenience

### Empfehlung: Local data, optional intelligence

Das Transaktions-Vault bleibt lokal. Der Nutzer kann zwischen zwei klar bezeichneten Modi wählen:

| Modus | Datenfluss | Nutzen | Einschränkung |
| --- | --- | --- | --- |
| **Private Local** | Daten und Inferenz bleiben auf dem Gerät. | stärkste Privacy-Aussage, geringe variable AI-Kosten | schwächere Modelle, höherer Installations-/Supportaufwand |
| **Smart Cloud** | Datenbank bleibt lokal; nur die für eine Antwort benötigten, minimierten Tool-Ergebnisse gehen nach expliziter Wahl an das Cloud-Modell. | stärkere Antworten, schneller Einstieg | nicht vollständig lokal; muss sichtbar und ehrlich erklärt werden |

Wichtig: Sobald Buchungstexte oder Tool-Ergebnisse ein Cloud-Modell erreichen, darf das Produkt nicht mit „Ihre Finanzdaten verlassen niemals das Gerät“ werben. Die UI sollte vor jeder Sitzung den aktiven Modus zeigen.

### Warum nicht sofort Open Banking

CSV hat Friktion, aber strategisch drei Vorteile:

- kein Teilen von Online-Banking-Zugangsdaten,
- schnelle Unterstützung weiterer Banken über Templates oder Mapping,
- Validierung des eigentlichen Nutzens vor regulatorischem und operativem Aufwand.

Kontoinformationsdienste werden unter PSD2/ZAG beaufsichtigt. Vor direktem Kontozugriff braucht das Produkt daher einen geklärten Partner-/Lizenzpfad. Wenn Retention bewiesen ist, sollte zuerst ein lizenzierter Aggregator geprüft werden, statt selbst zur regulierten Kontoinformationsschnittstelle zu werden.

## 8. Pricing und Monetarisierung

### Preis-Hypothese

| Plan | Preis | Inhalt |
| --- | ---: | --- |
| **First Review** | kostenlos | ein Import, vollständiger erster Review, begrenzte Nachfragen |
| **Pro** | 9,90 €/Monat oder 89 €/Jahr | unbegrenzte Historie/Imports, Monthly Reviews, 90-Tage-Prognose, Szenarien, Ziele, Decision Ledger, beide Inferenzmodi innerhalb fairer Nutzung |
| **Founding Pro** | 59 € im ersten Jahr, maximal 100 Kunden | alle Pro-Funktionen plus persönliches Onboarding und Feedback-Zugang; kein Lifetime Deal |
| **Household** | später ca. 129 €/Jahr | zwei Personen, gemeinsame und private Konten, gemeinsame Ziele und Review-Freigabe |

Der kostenlose Review muss echten Wert liefern. Die Paywall liegt bei **Kontinuität, Zukunft und Umsetzung**, nicht bei der ersten brauchbaren Antwort.

### Bottom-up Business Case

Bei 89 € Jahrespreis inklusive deutscher Umsatzsteuer:

| Zahlende Kunden | Brutto-ARR | Einordnung |
| ---: | ---: | --- |
| 100 | 8.900 € | Zahlungsbeweis, kein tragfähiges Unternehmen |
| 500 | 44.500 € | solides Indie-Nebeneinkommen, Support muss sehr effizient sein |
| 2.000 | 178.000 € | potenziell tragfähiges kleines Softwareunternehmen |
| 10.000 | 890.000 € | relevantes B2C-Geschäft; benötigt starke Distribution und Operations |

**Beispiel-Unit-Economics pro Jahreskunde:**

- 89,00 € Bruttoumsatz
- 14,21 € enthaltene Umsatzsteuer
- ca. 2,67 € Payment-Kosten (Annahme: 3 %)
- maximal 18 € AI/Infra-Budget pro Jahr
- ca. 54 € Deckungsbeitrag vor Support, Gehältern, Marketing und Ertragsteuern

Das ergibt eine harte Produktanforderung: **CAC unter 25 €**, bevorzugt organisch/referral, und Self-Service-Support. Bezahlte Performance-Werbung ist bei diesem ARPU wahrscheinlich kein sinnvoller Startkanal.

### Der spätere ARPU-Hebel

Nach bewiesener Haushalts-Retention kann ein Coach-/Advisor-Sponsorplan bei etwa 99–199 €/Monat getestet werden. Der Kunde behält die Datenhoheit; der Berater erhält nur explizit freigegebene Reviews. Das ist ein Distributions- und Monetarisierungshebel, kein MVP-Feature.

## 9. 90-Tage-Go-to-Market

### Tage 1–14: Problem und Preis vor Feature-Bau testen

1. Landingpage mit nur einem Outcome: „Finde in fünf Minuten heraus, was monatlich wirklich frei verfügbar ist.“
2. 15 Interviews mit dem primären ICP; keine allgemeine Feature-Abfrage.
3. Fünf Concierge-Reviews mit echten Exporten durchführen.
4. Founding Pro für 59 € anbieten – nicht „würdest du zahlen?“ fragen.
5. Jede Frage und jedes Misstrauen codieren: Import, Korrektheit, Privacy, Relevanz, Handlung.

**Go-Gate:** mindestens 5 der ersten 15 qualifizierten Personen zahlen oder hinterlegen verbindlich eine Bestellung. Sonst Positionierung/ICP ändern, bevor weitere Plattformtechnik gebaut wird.

### Tage 15–45: Trusted Money Model und First Review

- In-App-Import und Qualitätsprüfung bauen
- zehn echte anonymisierte/erlaubte Golden Datasets aufbauen
- Reviews persönlich mit Nutzern durchgehen
- nur die drei häufigsten Entscheidungstypen unterstützen
- wöchentlich Demo und Preis erneut anbieten

### Tage 46–75: Outcome Loop

- Szenarien, Action Cards und Decision Ledger ausrollen
- ersten Folgeimport mit denselben Nutzern durchführen
- gemeldeten und tatsächlich bestätigten Euro-Nutzen trennen
- drei Fallstudien mit konkretem Vorher/Nachher dokumentieren

### Tage 76–90: Paid Beta

- 30 aktive Haushalte, davon mindestens 10 bezahlt
- monatlichen Import-Reminder testen
- Referral: „Lade deinen Partner zum gemeinsamen Review ein“ erst nach positiver Nutzung
- Inhalte für konkrete Suchintentionen veröffentlichen: Sparkassen-CSV lokal auswerten, echtes frei verfügbares Einkommen berechnen, unregelmäßige Kosten monatlich planen
- kein breiter Product-Hunt-/Konferenzlaunch vor Retention

## 10. Produktmetriken und Go/No-Go-Gates

| Stufe | Hauptmetrik | Zielwert |
| --- | --- | ---: |
| Acquisition | qualifizierte Landingpage → Importstart | > 20 % im warmen Traffic |
| Activation | Import → vollständiger First Review in 5 Minuten | > 70 % |
| Trust | Nutzer bestätigt die fünf Kernzahlen ohne wesentliche Korrektur | > 90 % |
| Value | Review führt zu relevanter Erkenntnis/Entscheidung | > 50 % |
| Payment | qualifizierte Beta-Nutzer → bezahlt | > 25 % |
| Retention | zweiter Review im Folgemonat | > 40 % in der Beta |
| Outcome | bestätigte Action pro abgeschlossenem Review | > 30 % |
| Quality | wesentliche gemeldete Zahlenabweichung | < 10 %, danach weiter senken |

### Kill/Pivot Criteria

- Weniger als 5 von 30 qualifizierten Nutzern zahlen trotz persönlichem Onboarding.
- Nutzer finden Fragen interessant, kommen aber nach einem Monat nicht wieder.
- Die meisten wünschen primär automatische Kontosynchronisation und akzeptieren CSV nicht einmal für einen hochqualitativen Review.
- Der Supportaufwand liegt dauerhaft über 30 Minuten pro aktivem Nutzer und Monat.
- Nutzer können den Unterschied zu Finanzguru/ChatGPT nicht in einem Satz wiedergeben.

Dann nicht mehr Features addieren. Entweder ICP zu Advisors/Coaches verschieben oder das Produkt als Open-Source-/Portfolio-Projekt behandeln.

## 11. Konkrete nächste zehn Product-Issues

1. `ImportRun` plus Upload-UI und Preview
2. Konto- und Importzeitraum-Onboarding
3. Transfer-Pairing und Ausschluss aus Ausgaben
4. Merchant-Normalisierung und nutzerspezifische Regeln
5. Recurrence- und jährliche-Kosten-Erkennung
6. Golden Dataset für 50 Kernfragen und Finanzsummen
7. `FinancialBaseline`-Artefakt mit fünf bestätigbaren Zahlen
8. `MoneyReview` mit Evidence und Confidence
9. deterministische 90-Tage- und Szenario-Engine
10. `Decision`/`Action` mit späterer Outcome-Verifizierung

Chat Sessions, Prompt Management und zusätzliche Modellanbieter kommen nur so weit, wie sie diese zehn Issues direkt unterstützen. Feedback/Evals sind wichtig; eine perfekte Chat-Historie ist es noch nicht.

## 12. Ein-Satz-Pitch und Launch Copy

### Pitch

> **assetagent ist dein privater Finanz-CFO: Importiere deine deutschen Kontoumsätze und erhalte einen belegbaren Monatsplan, eine 90-Tage-Vorschau und klare Antworten darauf, was du dir wirklich leisten kannst – ohne deine Bankzugänge abzugeben.**

### Landingpage Hero

**Was bleibt dir wirklich übrig?**  
Importiere deinen Kontoexport. assetagent erkennt laufende und jährliche Kosten, erklärt jede Zahl mit den zugrunde liegenden Buchungen und zeigt dir, welche finanzielle Entscheidung als Nächstes sinnvoll ist.

CTA: **Kostenlosen Money Review erstellen**

## Quellen und Marktsignale

- [ChatGPT Personal Finance](https://openai.com/index/personal-finance-chatgpt/): seit Juni 2026 für Plus und Pro in den USA; Kontoverknüpfung, Kategorisierung, Dashboard, Financial Memory und grounded Q&A.
- [Finanzguru Plus](https://hilfe.finanzguru.de/de/articles/1509506): Analysen, Prognosen, Budgets, Vertragserkennung und vollständige Historie; regulärer Preis laut Hilfe 0,99 €/Woche.
- [Outbank Preise](https://help.outbankapp.com/en/kb/articles/was-kostet-das-abo): 3,99 €/Monat bzw. 39,99 €/Jahr für Privatkunden.
- [Actual Budget](https://actualbudget.org/): local-first, optional E2E-verschlüsselt, Budgeting, Reports, Bank-Sync und mehrere Importformate.
- [MoneyMoney](https://moneymoney.app/): deutscher FinTS/HBCI-Zugang, lokale verschlüsselte Datenbank, Kategorien, Budgets und Wertpapierübersicht.
- [Cleo](https://web.meetcleo.com/): dialogorientiertes Money Coaching, Ziele und Automatisierung.
- [Copilot Money](https://www.copilot.money/): AI-Kategorisierung, Budgets, Cashflow, Investments und Net Worth.
- [Monarch Money](https://www.monarch.com/pricing): umfangreiche PFM-Suite; öffentlich sichtbares Preis-/Wertsignal um 99 US-Dollar/Jahr.
- [Deutsche Bundesbank zur PSD2](https://www.bundesbank.de/de/aufgaben/unbarer-zahlungsverkehr/psd2/psd2-775434): Kontoinformationsdienste, explizite Zustimmung und Aufsicht von Drittanbietern.

---

**Klare Empfehlung:** Die nächsten acht Wochen nicht in Assets, Bank-Sync oder allgemeine Chat-Infrastruktur investieren. Zuerst einen First Review bauen, fünf Personen dafür bezahlen lassen und denselben Nutzern einen Monat später nachweisbar erneut helfen. Erst dieser Loop macht aus `assetagent` ein Produkt statt eines guten Demos.
