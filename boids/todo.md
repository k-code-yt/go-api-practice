[x] rework to single game loop
[x] get nearest boids in a loop
[x] check a, c, s formulas
[x] wall rejection
[x] add actual fish
[x] add gif background

<!-- PERFORMANCE -->

[x] add tests -> testing.B
[x] cpu#1: rework to spiral grid
[x] add worker pool
[x] cpu#2: math.pow && path.sqrt -> pre-calc sqrt for radiuses -> compare sqrt vs sqrt
[x] move results to worker
[x] how to pre-calc fish rotations
[x] how to skip fish rotation if direction didn't change

<!-- SHEEP -->

[x] change bg
[x] render sheep sprite(remove fish)
[x] investigate mem usage -> why 330mb(static, does not depend on sheep count)
[x] how to bounce on bush border -> how to check for collision with bush

[] add playble character
[] add collision with sheep
[] add push on collision

[] vertical tilt -> make sheep apear closer and further away
[] fix sheep angle

<!-- PENDING -->

[] investigate race condition error -> https://claude.ai/chat/db32d764-e9fa-4409-8d3f-2f82ffd5600e
