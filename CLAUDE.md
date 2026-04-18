# 1UP - Usenet Poster

## Description
Application Go multiplateforme (Linux CLI/Web, macOS, Windows x64/ARM) pour poster sur Usenet.

## Architecture

```
1UP/
├── cmd/
│   └── 1up/
│       └── main.go              # Point d'entrée
├── internal/
│   ├── config/
│   │   └── config.go            # Gestion configuration (YAML)
│   ├── parpar/
│   │   └── parpar.go            # Wrapper ParPar
│   ├── nyuu/
│   │   └── nyuu.go              # Wrapper Nyuu
│   ├── mediainfo/
│   │   └── mediainfo.go         # Extraction MediaInfo JSON
│   ├── api/
│   │   └── upload.go            # Upload NZB + MediaInfo vers API
│   └── web/
│       └── server.go            # Interface web (optionnel)
├── binaries/                    # Binaires embarqués ParPar + Nyuu par plateforme
│   ├── linux-amd64/
│   ├── darwin-amd64/
│   ├── darwin-arm64/
│   └── windows-amd64/
├── .github/
│   └── workflows/
│       └── build.yml            # CI multi-plateforme
├── go.mod
├── go.sum
├── config.example.yml
└── CLAUDE.md
```

## Stack technique
- **Langage** : Go 1.22+
- **Config** : YAML (viper ou similaire)
- **ParPar** : binaire embarqué via embed ou extraction au runtime
- **Nyuu** : binaire embarqué via embed ou extraction au runtime
- **MediaInfo** : bibliothèque Go native (pas de dépendance externe)
- **API** : upload multipart vers `https://unfr.pw/api-upload_v2`
- **Web** : interface web optionnelle pour Linux

## Interfaces
- **CLI** : Linux — progression en terminal (barre de progression texte, pourcentage, ETA)
- **Desktop (GUI)** : Linux, macOS, Windows — interface graphique avec progression visuelle (barres, pourcentage, ETA)
- **Web** : Linux — serveur HTTP intégré, accessible via `ip:port`, même UI que le desktop
- Framework desktop/web : **Wails** (Go backend + frontend web HTML/CSS/JS, même UI pour desktop et web)

## UX
- **Progression en temps réel** pour ParPar (parsing stderr : pourcentage, vitesse, ETA)
- **Progression en temps réel** pour Nyuu (parsing stderr via `--progress=stderr` : articles postés, vitesse, ETA)
- CLI : affichage texte inline / barre de progression terminal
- Desktop : barres de progression graphiques, logs en temps réel

## Conventions
- Commits en **français**, sans mention d'IA
- Push automatique après chaque commit
- GitHub : https://github.com/1UPFR/1UP
- Compilation via GitHub Actions pour toutes les plateformes

## Paramètres par défaut

### ParPar
```
-s10M -S -m4096M -t16 -r20%
```

### Nyuu
```
-S (SSL)
-g alt.binaries.boneless
--obfuscate-articles
Format from: {rand(14)} {rand(14)}@{rand(5)}.{rand(3)}
Format message-id: {rand(32)}@{rand(8)}.{rand(3)}
Format subject: {rand(32)}
Format nzb-subject: [{0filenum}/{files}] - "{filename}" yEnc ({part}/{parts})
```

### API Upload
```
URL: https://unfr.pw/api-upload_v2?apikey=<APIKEY>
Champs: rlsname, generated_nfo_json (fichier), nzb (fichier), upload=upload
```
