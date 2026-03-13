[x] rework to single game loop
[x] get nearest boids in a loop
[x] check a, c, s formulas
[x] wall rejection
[x] add actual fish
[x] add gif background

[] learn how to render gif

<!-- PEFORMANCE -->

[x] add tests -> testing.B
[x] cpu#1: rework to spiral grid
[x] add worker pool
[] cpu#2: math.pow && path.sqrt -> pre-calc sqrt for radiuses -> compare sqrt vs sqrt
[] move results to worker
[] how to pre-calc fish rotations
[] how to skip fish rotation if direction didn't change

[] review mem -> how to fix GIF mem usage(for later)

<!-- curr.mem = 737.85MB -->

<!-- curr.cpu -->

flat flat% sum% cum cum%
1.35s 30.54% 30.54% 2.37s 53.62% math.pow
0.41s 9.28% 39.82% 0.41s 9.28% runtime.cgocall
0.39s 8.82% 48.64% 0.57s 12.90% image/draw.drawRGBA
0.33s 7.47% 56.11% 0.33s 7.47% runtime.memmove
0.32s 7.24% 63.35% 0.33s 7.47% math.modf
0.24s 5.43% 68.78% 0.42s 9.50% math.ldexp
0.21s 4.75% 73.53% 0.25s 5.66% compress/lzw.(*Reader).decode
0.21s 4.75% 78.28% 2.58s 58.37% github.com/k-code-yt/go-api-practice/boids.Vector2D.Distance
0.15s 3.39% 81.67% 0.18s 4.07% image.(*Paletted).RGBA64At
0.15s 3.39% 85.07% 0.15s 3.39% math.IsInf (inline)
0.12s 2.71% 87.78% 0.12s 2.71% math.normalize (inline)
0.11s 2.49% 90.27% 0.18s 4.07% github.com/hajimehoshi/ebiten/v2/internal/atlas.(*Image).writePixels.func2
0.07s 1.58% 91.86% 0.07s 1.58% math.IsNaN (inline)
0.07s 1.58% 93.44% 0.15s 3.39% math.frexp
0.03s 0.68% 94.12% 2.63s 59.50% github.com/k-code-yt/go-api-practice/boids.(*Boid).calcAcceleration
0.03s 0.68% 94.80% 0.03s 0.68% image/color.RGBA.RGBA
0.03s 0.68% 95.48% 0.03s 0.68% math.Abs (inline)
0.01s 0.23% 95.70% 0.03s 0.68% github.com/hajimehoshi/ebiten/v2/internal/mipmap.(\*Mipmap).DrawTriangles
0.01s 0.23% 95.93% 2.38s 53.85% math.Pow (inline)
