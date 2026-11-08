// Draws HQ's PWA icons (apps/hq/static/icons/*.png) with no image deps:
// "813" in a blocky 5x7 font, cream on troop green. Re-run after a design change:
//
//   bun scripts/icons.ts
import { deflateSync } from "node:zlib";
import { repoRoot } from "./ports";

const GREEN = [0x1f, 0x4d, 0x2b] as const;
const CREAM = [0xf6, 0xf1, 0xe1] as const;

const GLYPHS: Record<string, string[]> = {
  "8": ["01110", "10001", "10001", "01110", "10001", "10001", "01110"],
  "1": ["00100", "01100", "00100", "00100", "00100", "00100", "01110"],
  "3": ["11110", "00001", "00001", "01110", "00001", "00001", "11110"],
};

const crcTable = Array.from({ length: 256 }, (_, n) => {
  let c = n;
  for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
  return c >>> 0;
});
function crc32(bytes: Uint8Array): number {
  let c = 0xffffffff;
  for (const b of bytes) c = (crcTable[(c ^ b) & 0xff] ?? 0) ^ (c >>> 8);
  return (c ^ 0xffffffff) >>> 0;
}

function chunk(type: string, data: Uint8Array): Buffer {
  const body = Buffer.concat([Buffer.from(type, "ascii"), data]);
  const out = Buffer.alloc(body.length + 8);
  out.writeUInt32BE(data.length, 0);
  body.copy(out, 4);
  out.writeUInt32BE(crc32(body), body.length + 4);
  return out;
}

// RGBA pixels → PNG bytes.
function png(size: number, rgba: Uint8Array): Buffer {
  const header = Buffer.alloc(13);
  header.writeUInt32BE(size, 0);
  header.writeUInt32BE(size, 4);
  header[8] = 8; // bit depth
  header[9] = 6; // RGBA
  const raw = Buffer.alloc(size * (size * 4 + 1));
  for (let y = 0; y < size; y++) {
    raw[y * (size * 4 + 1)] = 0; // filter: none
    Buffer.from(rgba.buffer, y * size * 4, size * 4).copy(
      raw,
      y * (size * 4 + 1) + 1,
    );
  }
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", header),
    chunk("IDAT", deflateSync(raw)),
    chunk("IEND", new Uint8Array()),
  ]);
}

// maskable: full-bleed background, text inside the 80% safe zone.
// Otherwise a rounded square with transparent corners.
function draw(size: number, maskable: boolean): Buffer {
  const ss = 4; // supersampling for smooth edges
  const text = "813";
  const cols = text.length * 6 - 1;
  const textWidth = size * (maskable ? 0.62 : 0.74);
  const cell = textWidth / cols;
  const left = (size - textWidth) / 2;
  const top = (size - cell * 7) / 2;
  const radius = size * 0.2;

  const inShape = (x: number, y: number) => {
    if (maskable) return true;
    const cx = Math.min(Math.max(x, radius), size - radius);
    const cy = Math.min(Math.max(y, radius), size - radius);
    return (x - cx) ** 2 + (y - cy) ** 2 <= radius ** 2;
  };
  const inText = (x: number, y: number) => {
    const col = Math.floor((x - left) / cell);
    const row = Math.floor((y - top) / cell);
    if (col < 0 || row < 0 || row > 6 || col >= cols || col % 6 === 5)
      return false;
    const glyph = GLYPHS[text[Math.floor(col / 6)] ?? ""];
    return glyph?.[row]?.[col % 6] === "1";
  };

  const rgba = new Uint8Array(size * size * 4);
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      let shape = 0;
      let ink = 0;
      for (let sy = 0; sy < ss; sy++) {
        for (let sx = 0; sx < ss; sx++) {
          const px = x + (sx + 0.5) / ss;
          const py = y + (sy + 0.5) / ss;
          if (inShape(px, py)) {
            shape++;
            if (inText(px, py)) ink++;
          }
        }
      }
      const i = (y * size + x) * 4;
      const t = shape ? ink / shape : 0;
      for (let c = 0; c < 3; c++) {
        rgba[i + c] = Math.round(
          (GREEN[c] ?? 0) * (1 - t) + (CREAM[c] ?? 0) * t,
        );
      }
      rgba[i + 3] = Math.round((255 * shape) / (ss * ss));
    }
  }
  return png(size, rgba);
}

if (import.meta.main) {
  const dir = `${repoRoot}apps/hq/static/icons/`;
  const icons: [string, number, boolean][] = [
    ["icon-192.png", 192, false],
    ["icon-512.png", 512, false],
    ["maskable-512.png", 512, true],
    ["apple-touch-icon.png", 180, true],
  ];
  for (const [name, size, maskable] of icons) {
    await Bun.write(dir + name, draw(size, maskable));
    console.log(`wrote ${dir}${name}`);
  }
}
