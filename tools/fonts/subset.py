# Builds the fallback fonts embedded by the game, keeping only the characters
# its language files use. Run it again after editing a language file:
#   py -3 tools/fonts/subset.py
import json
import pathlib
import urllib.request

from fontTools import subset
from fontTools.ttLib import TTFont
from fontTools.varLib import instancer

ROOT = pathlib.Path(__file__).resolve().parents[2]
LANGUAGES = ROOT / "internal" / "i18n" / "lang"
OUTPUT = ROOT / "internal" / "game" / "fonts"
CACHE = pathlib.Path(__file__).resolve().parent / "cache"
SOURCE = "https://github.com/google/fonts/raw/main/ofl/"

FONTS = [
    {"name": "Exo2", "source": "exo2/Exo2%5Bwght%5D.ttf", "scripts": ["cyrillic"]},
    {"name": "NotoSansKR", "source": "notosanskr/NotoSansKR%5Bwght%5D.ttf", "scripts": ["hangul"]},
    {"name": "NotoSansJP", "source": "notosansjp/NotoSansJP%5Bwght%5D.ttf", "scripts": ["han-jp"]},
    {"name": "NotoSansSC", "source": "notosanssc/NotoSansSC%5Bwght%5D.ttf", "scripts": ["han-sc"]},
]
WEIGHTS = {"Medium": 500, "Bold": 700}
ALWAYS = "".join(chr(code) for code in range(0x20, 0x7F)) + "♪…—–·×"


def load_languages():
    return [json.loads(path.read_text(encoding="utf-8")) for path in sorted(LANGUAGES.glob("*.json"))]


def characters_for(scripts, languages):
    text = ALWAYS + "".join(language["name"] for language in languages)
    for language in languages:
        if language.get("script") in scripts:
            text += "".join(language["strings"].values())
    return "".join(sorted(set(text)))


def cached(source):
    CACHE.mkdir(exist_ok=True)
    path = CACHE / source.split("/")[-1].replace("%5B", "[").replace("%5D", "]")
    if not path.exists():
        urllib.request.urlretrieve(SOURCE + source, path)
    return path


def build(font, characters):
    for weight_name, weight in WEIGHTS.items():
        variable = TTFont(cached(font["source"]))
        static = instancer.instantiateVariableFont(variable, {"wght": weight})
        options = subset.Options()
        options.layout_features = ["*"]
        options.name_IDs = ["*"]
        subsetter = subset.Subsetter(options)
        subsetter.populate(text=characters)
        subsetter.subset(static)
        target = OUTPUT / ("%s-%s.ttf" % (font["name"], weight_name))
        static.save(target)
        print("%-28s %5d characters  %7.1f KB" % (target.name, len(characters), target.stat().st_size / 1024))


def main():
    languages = load_languages()
    for font in FONTS:
        build(font, characters_for(font["scripts"], languages))


if __name__ == "__main__":
    main()
