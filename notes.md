# Standort-Check – Entscheidungen im Überblick

Pro Thema: **was** entschieden wurde, **warum** – und welcher **Kompromiss** bewusst in Kauf genommen wurde.

---

## 1. Fachliche Entscheidungen – „Was baue ich eigentlich?“

| Entscheidung                                                                      | Warum                                                                                                                                                        | Kompromiss                                                                   |
| --------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------- |
| **Grundstück statt Haus/Wohnung** (Hausnummer optional, Feld „Flurstück/Hinweis“) | Planeco begleitet Baugenehmigungen – unbebaute Grundstücke haben oft keine Hausnummer                                                                        | Ohne Hausnummer ist die Lage ungenauer                                       |
| **Nur Grundstücke in Deutschland** (deutsche PLZ, Hinweis im Formular)            | Baurecht ist Landesrecht; Test mit „Innsbruck 60200“ zeigte: Nutzer erfinden sonst eine PLZ, wird darauf aufmerksam                                          | Ausländische Eigentümer können trotzdem anfragen (Telefon mit Ländervorwahl) |
| **Telefon + PLZ sind Pflicht**                                                    | Sales ruft an; „Neustadt“ ohne PLZ (Beispiel #2) gibt es ~30-mal                                                                                             | Etwas längeres Formular                                                      |
| **Duplikat = gleiche Adresse UND (gleiche E-Mail ODER gleiches Telefon)**         | Case-Beispiel #1/#4 (Thomas Ahrens) ist nur über die normalisierte Telefonnummer erkennbar – die ursprünglich geplante „nur E-Mail“-Regel hätte es übersehen | Kein Fuzzy-Matching; Tippfehler in der Straße werden nicht erkannt           |
| **Duplikate speichern & markieren, nie verwerfen**                                | Eine echte Anfrage darf nie verloren gehen; Sales entscheidet                                                                                                | Liste wird länger → Filter „Duplikate ausblenden“                            |

## 2. Architektur

| Entscheidung                                                    | Warum                                                                                                                      | Kompromiss                                                                        |
| --------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| **React + Go + Supabase, alles kostenlos**                      | Go besitzt die gesamte Logik (Validierung, Duplikate, Auth); Supabase ist nur Datenbank – React spricht nie direkt mit ihr | Zwei Sprachen                                                                     |
| **Ein Vercel-Projekt, gleiche Domain, `/api/*`**                | Kein CORS, Login-Cookie funktioniert ohne Sonderregeln                                                                     | Abhängig von Vercels Go-Runtime                                                   |
| **Kleiner modularer Monolith** (Handler → Service → Repository) | Erklärbar, angemessen für die Größe; keine Microservices                                                                   | Muss bei starkem Wachstum umgebaut werden                                         |
| **Eine Tabelle `leads`** statt `addresses`/`sources`            | Alle Beziehungen 1:1, Adresse/Herkunft sind Momentaufnahmen der Anfrage                                                    | Später sinnvoll: Tabelle `properties`, wenn das Grundstück zum Arbeitsobjekt wird |
| **Supabase über Transaction-Pooler**                            | Direktverbindung nur IPv6 – Vercel kann das nicht                                                                          | Keine Prepared Statements                                                         |
| **RLS aktiviert**                                               | Sonst wären Leads über Supabases öffentliche REST-API lesbar                                                               | –                                                                                 |

## 3. Datenqualität

| Entscheidung                                                                                              | Warum                                                                      |
| --------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| **Rohwert + normalisierter Wert getrennt**                                                                | Sales sieht die Eingabe des Kunden, der Abgleich nutzt die bereinigte Form |
| **Deterministische Normalisierung** (`ß→ss`, `str.→strasse`, Telefon `+49…`, Unicode-NFC für Mac-Umlaute) | Gleiches rein → gleiches raus; erklärbar und testbar                       |
| **Ungültiges UTF-8 wird abgelehnt**                                                                       | Go hätte Umlaute sonst still in `�` verwandelt (beim Testen gefunden)      |
| **UTM-Werte kleingeschrieben, Kampagnen-Kürzel erzwungen**                                                | „Google“ und „google“ dürfen keine zwei Quellen in der Auswertung sein     |
| **Attribution bremst nie** (zu lange Werte werden gekürzt, nicht abgelehnt)                               | Die Anfrage zählt mehr als perfektes Tracking                              |

## 4. Formular

| Entscheidung                                                         | Warum                                                                 |
| -------------------------------------------------------------------- | --------------------------------------------------------------------- |
| **Validierung doppelt**: Frontend für Bedienung, Backend verbindlich | Frontend wird nie vertraut                                            |
| **Ländervorwahl als Auswahl, nur Ziffern, mind. 7**                  | Festes „+49“ mit „0170…“ dahinter hätte falsche Nummern erzeugt       |
| **Autofill-Korrektur** („Osterstraße 88“ → Straße + Nr.)             | Browser füllen beides oft in ein Feld                                 |
| **Honeypot statt CAPTCHA**                                           | Keine Hürde für echte Nutzer; Bots bekommen ein scheinbares OK        |
| **Attribution im sessionStorage**                                    | UTM-Parameter gehen beim Neuladen nicht verloren                      |
| **Schrift selbst gehostet**                                          | Google Fonts überträgt IP-Adressen an Google (DSGVO, LG München 2022) |
| **Admin-Bereich lazy geladen**                                       | Besucher von Handy-Anzeigen laden 88 kB statt 203 kB                  |

## 5. Sicherheit

| Entscheidung                                                         | Warum                                                            | Kompromiss                                             |
| -------------------------------------------------------------------- | ---------------------------------------------------------------- | ------------------------------------------------------ |
| **Ein Admin, bcrypt, HMAC-signiertes Cookie ohne Session-Tabelle**   | Passt zu Serverless, schnell gebaut                              | Logout nicht serverseitig widerrufbar; geteiltes Konto |
| **Cookie HttpOnly + Secure + SameSite=Lax; Mutationen nur als JSON** | Schutz gegen XSS-Diebstahl und CSRF                              | –                                                      |
| **bcrypt läuft auch bei falscher E-Mail**                            | Antwortzeit verrät nicht, ob die Admin-E-Mail existiert          | –                                                      |
| **Sortierung über Whitelist, Suche parametrisiert**                  | Kein SQL-Injection-Einfallstor                                   | –                                                      |
| **Öffentlicher Endpunkt verrät nichts** (ob Duplikat, keine ID)      | Niemand kann testen, ob eine E-Mail schon bekannt ist            | –                                                      |
| **CSV mit Schutz gegen Formel-Injection**                            | Ein Name wie `=HYPERLINK(…)` würde in Excel sonst ausgeführt     | Zellen beginnen mit `'`                                |
| **Löschen nur mit ID im Log**                                        | Gelöschte personenbezogene Daten sollen nicht im Log weiterleben | –                                                      |

## 6. Auswertung – „Welche Kampagnen taugen etwas?“

| Entscheidung                                                                           | Warum                                                              |
| -------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| **Status als Qualitätssignal** (neu → kontaktiert → qualifiziert / nicht qualifiziert) | Macht „taugt etwas“ messbar                                        |
| **Quote = qualifiziert ÷ bewertet, ohne Duplikate**                                    | Doppelte Anfragen schönen keine Kampagne                           |
| **Quote mit Zahlen: „100 % (1/1)“**                                                    | Kleine Stichproben sollen nicht überzeugender wirken, als sie sind |
| **`gclid` / `fbclid` werden gespeichert**                                              | Grundlage, um qualifizierte Leads an Google/Meta zurückzumelden    |
| **Kampagnen-Verwaltung ohne Fremdschlüssel zu Leads**                                  | Umbenennen/Löschen einer Kampagne ändert nie einen Lead            |
| **Recharts, Farben auf Farbenblindheit geprüft, Direktbeschriftung**                   | Lesbar für alle, Information nie nur über Farbe                    |

## 7. Geo-Erweiterung (Geoapify)

| Entscheidung                                                           | Warum                                                                                       |
| ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| **Strukturierte Suche, nur Deutschland, nur Adresse wird übertragen**  | Genauer, datensparsam, API-Key nie im Log                                                   |
| **Einzugsgebiet wird beim Lesen berechnet** (`SERVICE_AREA_STATES`)    | Änderung gilt sofort für alle alten Leads                                                   |
| **Geoapify blockiert nie einen Lead** (max. 3 s, Rest wird nachgeholt) | Geoapify brauchte im Test 3–15 s                                                            |
| **Unsichere Treffer unter fremder PLZ = „nicht gefunden“**             | Innsbruck wurde sonst in Baden-Württemberg verortet                                         |
| **Hinweise „PLZ passt nicht“ / „Lage ungenau“**                        | **Ergebnis: 2 von 5 Case-Beispielen haben eine falsche PLZ** (Dresden 01097, Hamburg 20259) |

## 8. E-Mail

| Entscheidung                                                              | Warum                                                                                            |
| ------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| **Brevo, reiner Text, mit Zusammenfassung der Eingaben + Antwortadresse** | Kunden sehen Tippfehler sofort; Text landet seltener im Spam und ist sicher gegen HTML-Injection |
| **Mailfehler kostet nie den Lead**                                        | Die Anfrage ist das Geschäftsereignis, die Mail ist Zusatz                                       |

## 9. Durch Testen gefundene Fehler

Passend zur Frage im Case: _„… ob jemand merkt, wenn etwas nicht stimmt“_

- Umlaute wurden still zu `�` beschädigt
- Recharts zeigte Balkenwerte eine Zeile versetzt (Hamburg „1/1“ statt „0/2“)
- Kampagnen-Platzhalter `{creative}`: kein einziger Lead wäre zugeordnet worden
- Eigener CSRF-Schutz blockierte das Löschen per API (415)
- Geoapify-Timeout: auch der Fehlerstatus wurde nicht gespeichert
- Brevo lehnte Mails nachträglich ab, die App meldete trotzdem „gesendet“ (falscher Absender)

## 10. Bewusst offen – nächste Schritte vor Produktion

1. Getrennte Prod-/Test-Datenbank, Backups, Migrations-Tool
2. Datenschutzerklärung und Impressum, Auftragsverarbeitungsverträge (AVV) mit allen Dienstleistern, Löschfristen
3. Eigene Mail-Domain (SPF/DKIM), Zustell-Fehler erfassen
4. Login-Rate-Limit, CAPTCHA, Security-Header, persönliche Konten mit Audit-Log
5. Job-Queue für Mail und Geodaten, Monitoring/Alerting
6. Qualifizierte Leads automatisch an Google/Meta zurückmelden (Offline-Conversions)
