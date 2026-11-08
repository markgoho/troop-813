// HQ is installable as a PWA (spec #17, issue #19): the manifest Chrome reads
// on the Kiosk tablet and Leaders' phones.
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

// jsdom gives import.meta.url an http: scheme, so resolve from dirname.
const read = (path: string) =>
  readFileSync(join(import.meta.dirname, "..", path));

interface Icon {
  src: string;
  sizes: string;
  type: string;
  purpose?: string;
}
const manifest = JSON.parse(read("static/manifest.webmanifest").toString()) as {
  name: string;
  short_name: string;
  start_url: string;
  scope: string;
  display: string;
  icons: Icon[];
};

// Width and height from a PNG's IHDR chunk.
function pngSize(bytes: Buffer): string {
  expect(bytes.subarray(1, 4).toString()).toBe("PNG");
  return `${bytes.readUInt32BE(16)}x${bytes.readUInt32BE(20)}`;
}

describe("PWA manifest", () => {
  test("names the app Troop 813 HQ, 813 HQ under the icon", () => {
    expect(manifest.name).toBe("Troop 813 HQ");
    expect(manifest.short_name).toBe("813 HQ");
  });

  test("opens fullscreen at the root, so the Kiosk can be pinned", () => {
    expect(manifest.start_url).toBe("/");
    expect(manifest.scope).toBe("/");
    expect(manifest.display).toBe("fullscreen");
  });

  test("ships 192 and 512 PNG icons, plus a maskable one, at their stated sizes", () => {
    const sizes = manifest.icons.map((icon) => {
      expect(icon.type).toBe("image/png");
      expect(pngSize(read(`static${icon.src}`))).toBe(icon.sizes);
      return icon.sizes;
    });
    expect(sizes).toEqual(expect.arrayContaining(["192x192", "512x512"]));
    expect(manifest.icons.some((i) => i.purpose === "maskable")).toBe(true);
  });

  test("is linked from every page", () => {
    expect(read("src/app.html").toString()).toContain(
      '<link rel="manifest" href="/manifest.webmanifest" />',
    );
  });
});
