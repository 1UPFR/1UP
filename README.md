# 1UP

Application multiplateforme pour poster sur Usenet.

![1UP](1up.png)

## Fonctionnalites

- **Post Usenet** automatise avec [ParPar](https://github.com/animetosho/ParPar) (par2) et [Nyuu](https://github.com/Antidote2151/Nyuu-Obfuscation) (post + obfuscation)
- **MediaInfo** integre via WebAssembly (aucune dependance externe)
- **TMDB** : identification automatique des films/series avec poster, genres, synopsis
- **File d'attente** : ajout de plusieurs fichiers, traitement sequentiel, progression en temps reel
- **Upload manuel** : envoi de NZB existants avec support MediaInfo JSON et BDInfo
- **Verification duplicat** : check automatique avant envoi sur l'API
- **Historique** : suivi de tous les traitements avec recherche, filtres et statistiques (SQLite)
- **Verification des mises a jour** automatique

## Plateformes

| Plateforme | Type |
|---|---|
| macOS (Apple Silicon + Intel) | Application desktop |
| Windows (64-bit) | Application desktop |
| Linux (64-bit) | Application desktop |
| Linux (amd64 + arm64) | CLI (terminal) |
| Linux (amd64 + arm64) | Web (serveur HTTP) |

## Installation

Telecharger la derniere version depuis les [Releases](https://github.com/1UPFR/1UP/releases).

### macOS

1. Decompresser le `.zip`
2. Glisser `1UP.app` dans le dossier **Applications**
3. Au premier lancement, macOS peut bloquer l'application
4. Aller dans **Reglages Systeme** > **Confidentialite et securite**
5. Dans la section **Securite**, cliquer sur **Ouvrir quand meme**

### Windows

1. Double-cliquer sur le `.exe`
2. Cliquer sur **Informations complementaires** > **Executer quand meme**

### Linux (CLI)

```bash
chmod +x 1up-cli-*-linux-amd64
./1up-cli-*-linux-amd64 fichier.mkv
```

### Linux (Web)

```bash
chmod +x 1up-web-*-linux-amd64
./1up-web-*-linux-amd64 --host 0.0.0.0 --port 8080
# Ouvrir http://votre-ip:8080
```

## Configuration

Au premier lancement, configurer dans **Reglages** :

- **Serveur Usenet** : host, port, user, password, connexions, SSL
- **ParPar** : taille slice, memoire, threads, redondance
- **API** : activer/desactiver, URL et cle API
- **Dossier de sortie** : emplacement des NZB et MediaInfo JSON

La configuration est sauvegardee automatiquement dans `~/.config/1up/config.json`.

## Formats supportes

- `.mkv`, `.mp4` : traitement complet (MediaInfo + par2 + post + API)
- `.iso` : par2 + post uniquement (pas de MediaInfo, pas d'envoi API)

## Licence

MIT
