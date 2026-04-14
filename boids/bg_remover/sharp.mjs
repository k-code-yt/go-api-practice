import sharp from "sharp";

const [, , input, output] = process.argv;

if (!input || !output) {
  console.log("Usage: node sharp.mjs <input> <output>");
  process.exit(1);
}

const { data, info } = await sharp(input)
  .ensureAlpha()
  .raw()
  .toBuffer({ resolveWithObject: true });
const [bgR, bgG, bgB] = [data[0], data[1], data[2]];
const tolerance = 30;

for (let i = 0; i < data.length; i += 4) {
  const dist = Math.sqrt(
    (data[i] - bgR) ** 2 + (data[i + 1] - bgG) ** 2 + (data[i + 2] - bgB) ** 2,
  );
  if (dist <= tolerance) data[i + 3] = 0;
}

await sharp(data, {
  raw: { width: info.width, height: info.height, channels: 4 },
})
  .png()
  .toFile(output);

console.log(`Done! Saved → ${output}`);
