"""Generate teinte icon: dark rounded square + glowing "T"
in the app accent color. Outputs 1024px PNG + multi-size ICO."""
from PIL import Image, ImageDraw, ImageFont, ImageFilter

S = 1024
MARGIN = 64
RADIUS = 190

# --- background: rounded square with subtle diagonal dark gradient ---
bg_top, bg_bot = (27, 31, 39), (11, 13, 17)
grad = Image.new("RGB", (S, S))
px = grad.load()
for y in range(S):
    t = y / (S - 1)
    px_row = tuple(int(a + (b - a) * t) for a, b in zip(bg_top, bg_bot))
    for x in range(S):
        px[x, y] = px_row

mask = Image.new("L", (S, S), 0)
ImageDraw.Draw(mask).rounded_rectangle(
    [MARGIN, MARGIN, S - MARGIN, S - MARGIN], radius=RADIUS, fill=255)

icon = Image.new("RGBA", (S, S), (0, 0, 0, 0))
icon.paste(grad, (0, 0), mask)

# --- letter 'C' mask ---
font = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 640)
letter_mask = Image.new("L", (S, S), 0)
d = ImageDraw.Draw(letter_mask)
bbox = d.textbbox((0, 0), "T", font=font)
w, h = bbox[2] - bbox[0], bbox[3] - bbox[1]
pos = ((S - w) // 2 - bbox[0], (S - h) // 2 - bbox[1])
d.text(pos, "T", font=font, fill=255)

# --- letter fill: vertical gradient in accent tones (#e05d44 family) ---
c_top, c_bot = (240, 140, 110), (224, 93, 68)
letter_grad = Image.new("RGB", (S, S))
lpx = letter_grad.load()
for y in range(S):
    t = y / (S - 1)
    row = tuple(int(a + (b - a) * t) for a, b in zip(c_top, c_bot))
    for x in range(S):
        lpx[x, y] = row

# --- glow: blurred colored copy of the letter under the sharp one ---
glow = Image.new("RGBA", (S, S), (0, 0, 0, 0))
glow.paste((224, 93, 68, 255), (0, 0), letter_mask)
glow = glow.filter(ImageFilter.GaussianBlur(28))

icon.alpha_composite(glow)
letter = Image.new("RGBA", (S, S), (0, 0, 0, 0))
letter.paste(letter_grad, (0, 0), letter_mask)
icon.alpha_composite(letter)

icon.save("build/appicon.png")

# --- ICO with standard sizes ---
icon.resize((256, 256), Image.LANCZOS).save(
    "build/windows/icon.ico",
    sizes=[(256, 256)])  # real multi-size ICO is produced by ImageMagick in `make icon`
print("done")
