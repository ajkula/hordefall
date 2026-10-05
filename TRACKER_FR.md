# La musique de Hordefall : un tracker maison

## L'idée

Je voulais une musique à l'ancienne, façon Amiga, et surtout une musique qui réagit au jeu. Plus il y a d'ennemis, plus ça s'emballe ; un boss arrive, une voix grave rentre. Du coup j'ai pensé aux trackers : un tracker, c'est déjà une musique découpée en pistes, donc parfait pour allumer ou couper des pistes selon ce qui se passe à l'écran.

Et en bonus : pas un seul fichier MP3 ou WAV dans le jeu. Le thème tient dans un petit fichier texte, et les autres morceaux sont composés à chaque lancement.

## En gros, comment ça marche

Il y a quatre étages, du plus bas au plus haut :

1. **Le synthé** fabrique le son, échantillon par échantillon.
2. **Le tracker** lit la partition et dit au synthé quelle note jouer, quand, sur quelle piste.
3. **La couche jeu** décide quelles pistes on entend, selon l'intensité de la partie.
4. **Le mixage** additionne tout et envoie le résultat à la carte son.

Tout le code est dans `internal/audio`.

## Le synthé (`synth.go`)

Pas de samples : chaque son est calculé, 44 100 fois par seconde. Une `Voice`, c'est :

- **un oscillateur** : carré (avec une largeur réglable, le son typique 8 bits), dent de scie, triangle, sinus, ou bruit pour la batterie ;
- **une enveloppe ADSR** : comment le volume monte, redescend, tient, puis s'éteint (attaque, déclin, maintien, relâchement) ;
- **un filtre passe-bas** pour adoucir le son ;
- **des options** : la note qui glisse (le kick qui descend), une sous-octave pour les basses des boss, une deuxième voix légèrement désaccordée pour épaissir, de la saturation pour les guitares de Grey Transmission ;
- **un panoramique** qui garde le même volume au centre et sur les côtés.

## Le tracker (`tracker.go`)

C'est exactement le principe des trackers Amiga.

Un morceau, c'est des **patterns** joués dans un **ordre**. Un pattern, c'est des **lignes**, et chaque ligne a une case par piste (16 pistes). Une case, c'est `NOTE INSTRUMENT EFFET`, par exemple :

```
A-4 05 037
```

Ça veut dire : la note La 4, avec l'instrument 5, et l'effet `037`. `---` veut dire « rien », `===` coupe la note.

Le temps avance par **ticks**, comme sur Amiga : un tick dure 2,5 / tempo secondes, et on passe à la ligne suivante tous les *speed* ticks.

Deux effets :

- **`0xy`, l'arpège** : la piste alterne entre trois notes à chaque tick. Avec une seule voix, on entend un accord qui scintille. C'est LE son Amiga.
- **`Cxx`, le volume** de la note, en hexadécimal de 00 à 40.

Point important : l'horloge du tracker, c'est le nombre d'échantillons produits, pas le temps du jeu. C'est pour ça que la musique continue quand on déplace la fenêtre et que tout le reste se fige.

## La couche jeu

### Les couches adaptatives

Chaque piste est branchée sur un signal du jeu, avec un seuil :

| Signal | Ce qu'il mesure |
|--------|-----------------|
| `always` | toujours à 1 : la base (batterie, basse, shaker, pad) |
| `horde` | les ennemis autour de moi, de 0 à 1 |
| `boss` | le nombre de spider tanks vivants, de 0 à 3 |
| `reactions` | les infernos, gels, explosions en chaîne |
| `danger` | vie basse, ou laser d'un boss en charge |

Toutes les pistes jouent tout le temps, en rythme. On ne fait que monter ou baisser leur volume, en douceur (0,9 seconde). Du coup, quand une piste rentre, elle tombe pile au bon endroit du morceau. Et plus c'est intense, plus le tempo accélère un peu.

### Le compositeur (`composer.go`)

À chaque lancement, il écrit cinq morceaux à partir de styles : une gamme, une suite d'accords, des rythmes écrits comme `"x...x..x"`, un tempo et des timbres. Neon Pursuit, Frozen Wastes, Ember March, Skyline Rush et Grey Transmission (celui qui sonne post-punk).

### L'arrangeur (`arranger.go`)

Il ajoute les pistes 11 à 16 (shaker, clap, pad, arpège pluck, accords sur les contretemps, harmonie sous la mélodie). Il lit les accords de chaque pattern et la batterie du morceau, donc tout ce qu'il ajoute tombe juste : que des notes de l'accord, et des coups qui se calent sur le rythme existant.

## Le mixage (`engine.go`)

On additionne la musique et les bruitages, chacun avec son volume. La musique passe par une petite « pièce » stéréo (`stereo_room.go`) : deux échos très courts croisés gauche/droite, sans les basses. Ça donne de la largeur sans que le son penche d'un côté au casque. Les basses restent bien au centre. Enfin, un écrêtage doux évite que ça sature.

## Qui fait tourner tout ça ?

Ce n'est pas le jeu qui pousse le son, c'est la carte son qui le tire. Ebitengine crée un lecteur audio sur sa propre goroutine, qui vient lire l'`Engine` dès que son tampon de 60 ms baisse.

Le jeu, lui, ne fait que déposer des infos dans l'`Engine`, sous un verrou : « joue tel bruitage », « voilà l'intensité », « change de morceau », « coupe la musique ». Les bruitages attendent dans une petite file, et le son est fabriqué au passage suivant du lecteur.

La page `Moteur audio Hordefall.html` montre tout ça en animation, bloc par bloc.

## Modifier la musique

- Le thème, c'est `internal/audio/music/theme.trk`. On change une note, on recompile, c'est tout.
- En haut du fichier : le tempo, les instruments, le panoramique, et les couches (`layer <piste> <signal> <seuil>`).
- Si j'écris moi-même des notes dans les pistes 11 à 16, l'arrangeur ne touche plus à ces pistes.
- `go test ./internal/audio -run Theme` vérifie que le fichier est valide, avec la ligne exacte en cas d'erreur.
- Pour écouter les morceaux en WAV : `$env:HORDEFALL_WAV_DIR = "./music/previews"; go test ./internal/audio -run ExportSongPreviews`.

## La différence avec un vrai tracker Amiga

Un vrai fichier `.MOD` ne calcule pas ses sons : il joue des samples enregistrés (des petits sons en 8 bits) qu'il accélère ou ralentit pour changer la note, sur 4 pistes. Le mien fabrique tout avec des formes d'onde et des enveloppes, sur 16 pistes. C'est plus proche d'un synthé de console 8 bits que d'un vrai MOD.

Une piste pour plus tard : un lecteur de vrais `.MOD`, pour jouer des musiques Amiga existantes, et pourquoi pas les rendre adaptatives piste par piste.
