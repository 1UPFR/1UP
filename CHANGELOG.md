# Changelog

## v1.3.9

### Corrections
- Version Web : detection BDInfo et upload API pour les ISO (manquait les endpoints)
- Version Web : validation de la team appliquee aussi cote serveur

## v1.3.8

### Corrections
- Version Web : 404 sur /login avec --auth (handlers enregistres sur le mauvais mux)

## v1.3.7

### Nouveautes
- ISO : extraction des infos depuis le BDInfo compagnon (resolution, codec, audio, HDR, langues, sous-titres) affichee comme MediaInfo
- Validation du nom de release : refus si pas de team a la fin ou team "notag"/"noteam" (CLI, Desktop, Web)

### Corrections
- Version Web : panic au demarrage avec --auth quand le path racine etait demande

## v1.3.6

### Nouveautes
- ISO : detection automatique du fichier BDInfo compagnon (meme nom .txt ou fichier contenant "bdinfo")
- ISO : upload API avec BDInfo dans le champ bdinfo_full quand le fichier est present

### Interface
- Upload manuel : suppression du champ BDInfo Mini (simplifie en un seul champ BDInfo)

## v1.3.5

### Interface
- Selection multiple de fichiers (Ctrl+clic) dans le selecteur natif

## v1.3.4

### Corrections
- Journal : ecriture asynchrone par batch (plus de pertes quand ca va vite)
- SQLite WAL mode pour les ecritures non-bloquantes

## v1.3.3

### Interface
- Taille et position de la fenetre sauvegardees entre les sessions
- Fenetre reduite a 1280x800 par defaut pour compatibilite scaling DPI

## v1.3.2

### Nouveautes
- Fallback TMDB sur l'API officielle si le proxy est indisponible
- Parser de nom de release integre (extraction titre, annee, type serie/film)
- Recherche automatique film ou serie avec bascule si aucun resultat

## v1.3.1

### Interface
- Support 1280x720 : minimum abaisse a 1024x600, paddings adaptatifs
- Journal : pagination dynamique adaptee a la taille de la fenetre

## v1.3.0

### Nouveautes
- Journal persistant SQLite (24 dernieres heures)
- Page Journal refaite : groupement par date, niveaux colores (info/warn/error)
- Le journal survit a la fermeture de l'application
- Nettoyage automatique des entrees > 24h

### Interface
- MediaInfo : barre verticale separatrice entre les 2 colonnes
- Mise a jour : l'application se ferme apres ouverture du lien de telechargement
- Journal : plus recents en haut, pagination, groupement par date

## v1.2.4

### Nouveautes
- Bouton Tester dans les reglages Usenet (test connexion NNTP + authentification)

## v1.2.3

### Corrections
- Bouton mise a jour ouvre le navigateur correctement sur toutes les plateformes

## v1.2.2

### Nouveautes
- Version Web : authentification login/pass optionnelle (--login --pass)
- Page de login integree avec cookie de session 30 jours

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
