# Changelog

## v1.2.1

### Nouveautes
- Blocage de l'application si une mise a jour est disponible
- Ecran de mise a jour avec lien de telechargement

### Corrections
- ISO : bouton Lancer visible, plus de boucle d'analyse infinie
- Historique : rechargement auto quand la page devient visible
- MediaInfo JSON : injection CompleteName et @ref dans le JSON natif

## v1.2.0

### MediaInfo
- Remplacement mediainfo.js npm par MediaInfoWasm natif (Emscripten)
- JSON full identique a mediainfo CLI (CompleteName, @ref)

### Securite
- URLs API et proxy TMDB retirees du code source
- Injection via secrets GitHub au build (ldflags)

### Interface
- Menu contextuel natif (copier/coller)
- Annonce Discord automatique a chaque release
- Changelog integre aux releases GitHub

## v1.1.0

### Reglages
- SSL deplace a cote du port
- Icone oeil pour afficher/masquer le mot de passe
- Chemin du fichier de configuration affiche
- Copier/coller au clic droit active

### Windows
- Les fenetres CMD de ParPar et Nyuu sont masquees

### CI
- Annonce Discord automatique a chaque release

## v1.0.0

- Version initiale
- Application desktop (macOS, Windows, Linux)
- Version CLI et Web pour Linux
- ParPar + Nyuu embarques (zero dependance)
- MediaInfo via WebAssembly
- Identification TMDB automatique
- File d'attente avec traitement sequentiel
- Progression en temps reel (vitesse, ETA)
- Upload manuel (NZB + MediaInfo/BDInfo)
- Verification des doublons API
- Historique SQLite
- Gestion ISO (par2 + post, pas d'API)
