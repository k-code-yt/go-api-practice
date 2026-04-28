<!-- NEXT TODOs -->

<!-- finilizing -->

[x] add menu -> select new_game, settings, exit
[] end menu/screen -> restart, go main menu
- game end screen
-- fix letter M
-- score not working for left 
-- no scene transition end screen menu


[] settings menu item
- boid count
- screen size

[] add good looking score -> invert on right side && add animation
[] add AI character mode
[] some music

<!-- refactor -->
[] Go over all TODOs

<!-- performance -->

[] go over per-ce todos

<!-- HANGING/PENDING -->

[] add wolf hiding in bush event
[] add some throwable item -> mud or something
[] add char select screen
[] add more playble characters
[] add collision unstuck logic -> N-ticks same pos -> disable coll
[] improve collision -> https://claude.ai/chat/e6ac8b3a-e7a6-4f2f-ba99-62ee0418e295
[] investigate race condition error -> https://claude.ai/chat/db32d764-e9fa-4409-8d3f-2f82ffd5600e
[] convert game to 2.5d

<!-- DONE -->

<!-- add 2nd player -->

[x] split arrows && WASD -> arr to right, WASD to left
[x] add chars enum && add sprite map for each
[x] dimentiontions map based on sprite/char
[x] make chars same size
[x] add 2nd player:

<!-- init -->

[x] rework to single game loop
[x] get nearest boids in a loop
[x] check a, c, s formulas
[x] wall rejection
[x] add actual fish
[x] add gif background

<!-- perf-ce -->

[x] add tests -> testing.B
[x] cpu#1: rework to spiral grid
[x] add worker pool
[x] cpu#2: math.pow && path.sqrt -> pre-calc sqrt for radiuses -> compare sqrt vs sqrt
[x] move results to worker
[x] how to pre-calc fish rotations
[x] how to skip fish rotation if direction didn't change

<!-- sheep -->

[x] change bg
[x] render sheep sprite(remove fish)
[x] investigate mem usage -> why 330mb(static, does not depend on sheep count)
[x] how to bounce on bush border -> how to check for collision with bush
[x] add collision with sheep

<!-- player -->

[x] add playble character
[x] coll/row usage detection -> detect current frame to render from sprite
[x] bg collision -> isBush && collision box draw for debug

<!-- tiles -->

[x] add sheep bush bounce
[x] render barn && gate collision
[x] add logic that checks if sheep collided with barn
[x] add sheep counter && remove sheep from render once in barn

<!-- ram -->

[x] add ram charging at a player
[x] direction switch
[x] sleep animation
[x] rework animation to be generic impact effect
[x] add animation to events -> make them visibly different
[x] time limit for ram? -> or N-charges then remove him?

<!-- events -->
[x] add trailing on speed up
[x] add daze effect on collision w/ RAM
[x] check why no collision box for energy event
[x] add banana slip event
[x] make bananas spawn only on oppisite side
[-] collide with feet only???
[x] add energy/speed event