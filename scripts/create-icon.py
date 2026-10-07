"""Rebuild editable MetaClean icon assets with Pillow (builds do not need Python)."""
from pathlib import Path
from PIL import Image, ImageDraw

root = Path(__file__).resolve().parents[1]
scale = 4
image = Image.new("RGBA", (256 * scale, 256 * scale))
draw = ImageDraw.Draw(image)
def polygon(points, color):
    draw.polygon([(x * scale, y * scale) for x, y in points], fill=color)

draw.rounded_rectangle((8*scale, 8*scale, 248*scale, 248*scale), radius=48*scale, fill="#172333")
# A violet privacy shield and an amber eraser: distinct silhouettes at small sizes.
shield = [(128,38),(205,65),(199,128),(175,165),(128,202),(81,165),(57,128),(51,65)]
polygon(shield, "#B7A2F8")
inner = [(128,62),(181,80),(177,120),(159,148),(128,174),(97,148),(79,120),(75,80)]
polygon(inner, "#66519B")
eraser = [(151,102),(205,146),(155,206),(119,207),(85,178)]
polygon(eraser, "#172333")
eraser_fill = [(151,113),(193,147),(151,197),(123,197),(97,177)]
polygon(eraser_fill, "#F2BC67")
polygon([(110,162),(152,195),(123,197),(97,177)], "#FFF0CF")
image = image.resize((256,256), Image.Resampling.LANCZOS)
image.save(root / "build/appicon.png")
image.save(root / "build/windows/icon.ico", sizes=[(16,16),(20,20),(24,24),(32,32),(40,40),(48,48),(64,64),(96,96),(128,128),(256,256)])
polygons = [(shield,"#B7A2F8"),(inner,"#66519B"),(eraser,"#172333"),(eraser_fill,"#F2BC67"),([(110,162),(152,195),(123,197),(97,177)],"#FFF0CF")]
svg = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256">\n<rect x="8" y="8" width="240" height="240" rx="48" fill="#172333"/>\n'
for points,color in polygons:
    svg += '<polygon points="' + ' '.join(f'{x},{y}' for x,y in points) + f'" fill="{color}"/>\n'
svg += '</svg>\n'
(root / "build/appicon.svg").write_text(svg, encoding="utf-8")
