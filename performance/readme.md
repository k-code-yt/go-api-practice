# Memory Metrics Guide

This document explains every metric printed by `PrintMemStats()` and what they mean for your Go application.

---

## Table of Contents

1. [Core Memory Metrics](#core-memory-metrics)
2. [Memory Breakdown](#memory-breakdown)
3. [Allocation Statistics](#allocation-statistics)
4. [Garbage Collection Metrics](#garbage-collection-metrics)
5. [What Changes Mean](#what-changes-mean)
6. [Interpreting Your Results](#interpreting-your-results)

---

## Core Memory Metrics

### **Alloc** - Currently Allocated Memory

**What it is:** The total bytes of memory currently allocated and in use (live objects only).

**Formula:** `Alloc = Total heap memory - Memory freed by GC`

**What it tracks:**

- Objects currently alive in your program
- Maps, slices, structs, strings that haven't been freed
- Does NOT include freed memory

**Goes UP when:**

- You create new objects (maps, slices, structs)
- You append to slices
- You add entries to maps

**Goes DOWN when:**

- Garbage collection runs and frees unused objects
- Objects go out of scope and GC cleans them

**Example:**

```
Alloc = 1.54 MB

This means your program currently has 1.54 MB of live objects in memory.
If this number keeps growing, you might have a memory leak.
```

**In your code:**

- Each `PartitionStateLock` instance: ~200 bytes
- Each map entry: ~24 bytes (8 byte key + 16 byte value + overhead)
- With 1000 map entries: ~24 KB per instance

---

### **TotalAlloc** - Cumulative Allocations

**What it is:** The total bytes allocated since the program started (NEVER decreases).

**Formula:** `TotalAlloc = Sum of all allocations ever made`

**What it tracks:**

- Every single allocation your program has made
- Includes memory that has been freed
- Useful for measuring allocation rate

**Goes UP when:**

- Any allocation happens (always)

**Goes DOWN when:**

- NEVER (this is cumulative)

**Example:**

```
START:      TotalAlloc = 538 KB
AFTER RUN:  TotalAlloc = 1.69 MB
AFTER GC:   TotalAlloc = 1.69 MB (unchanged - GC doesn't affect this)

Allocation during run = 1.69 MB - 538 KB = 1.15 MB
This tells you your code allocated 1.15 MB total during the test.
```

**In your code:**

```
If TotalAlloc grows by 1 MB during 5 seconds:
- Allocation rate = 1 MB / 5 sec = 200 KB/sec
- For read-heavy: Low allocation rate is good
- For write-heavy: Higher allocation rate is expected
```

**Key insight:** The DIFFERENCE between two measurements shows how much work was done.

---

### **Sys** - Total Memory from OS

**What it is:** Total memory obtained from the operating system.

**Formula:** `Sys = Heap + Stack + GC metadata + Runtime structures`

**What it tracks:**

- What you see in `top` or Task Manager
- Memory reserved for future use
- Rarely returns to OS

**Goes UP when:**

- Your program needs more memory than currently reserved
- OS allocates more memory to your process

**Goes DOWN when:**

- Almost never (Go keeps memory for efficiency)

**Example:**

```
Sys = 8.12 MB

This is what the OS thinks your program is using.
Even if Alloc = 1 MB, Sys = 8 MB means Go reserved 8 MB "just in case".
```

**Why Sys > Alloc:**

```
Sys (8 MB)          ┌─────────────────────────┐
                    │  Alloc (1 MB)          │ ← Your objects
                    │  ────────────────────   │
                    │  Idle Heap (3 MB)      │ ← Reserved but unused
                    │  ────────────────────   │
                    │  Stack (0.5 MB)        │ ← Goroutine stacks
                    │  ────────────────────   │
                    │  GC metadata (1 MB)    │ ← GC bookkeeping
                    │  ────────────────────   │
                    │  Runtime (2.5 MB)      │ ← Go runtime
                    └─────────────────────────┘
```

**In your code:**

- Sys usually stays constant around 8-12 MB
- Only grows if you allocate LOTS of memory (100+ MB)
- If Sys keeps growing, you have a real memory leak

---

## Memory Breakdown

### **HeapInuse** - Active Heap Memory

**What it is:** Heap memory currently holding objects.

**Goes UP when:**

- You allocate new objects
- Objects are created but not yet freed

**Goes DOWN when:**

- GC frees objects
- Memory moves to HeapIdle

**Example:**

```
HeapInuse = 2.72 MB

This is heap memory actively used for your objects.
```

---

### **HeapIdle** - Reserved but Unused Heap

**What it is:** Heap memory reserved but not currently used.

**Purpose:**

- Ready for fast allocations
- Avoids asking OS for memory repeatedly

**Goes UP when:**

- GC frees objects (moves from HeapInuse to HeapIdle)
- OS gives more memory

**Goes DOWN when:**

- You allocate new objects (moves to HeapInuse)

**Example:**

```
START:      HeapInuse = 1 MB,   HeapIdle = 2 MB
AFTER RUN:  HeapInuse = 2.7 MB, HeapIdle = 1.7 MB

During run, 1.7 MB moved from Idle → Inuse (you allocated objects)
```

**Key relationship:**

```
Total Heap = HeapInuse + HeapIdle
```

---

### **StackInuse** - Goroutine Stack Memory

**What it is:** Memory used by all goroutine stacks.

**Each goroutine:**

- Starts with ~2-8 KB stack
- Grows if needed (rare)

**Goes UP when:**

- You create new goroutines
- Goroutines' stacks grow (deep recursion, large local variables)

**Goes DOWN when:**

- Goroutines exit

**Example:**

```
StackInuse = 576 KB with 8 instances × 3 goroutines = 24 goroutines
576 KB / 24 = 24 KB per goroutine (normal)

If StackInuse = 10 MB, you might have:
- Too many goroutines
- Goroutines with large local variables
- Deep recursion
```

**In your code:**

```
Each PartitionStateLock creates 3 goroutines:
- appendLoop goroutine
- updateLoop goroutine
- commitLoop goroutine

8 instances × 3 = 24 goroutines ≈ 500-600 KB stack memory
```

---

## Allocation Statistics

### **HeapObjects** - Live Object Count

**What it is:** Number of objects currently allocated on the heap.

**What counts as an object:**

- Each map entry = 2 objects (key + value)
- Each struct instance = 1 object
- Each slice = 1 object
- Each string = 1 object
- Each interface value = 1 object (usually)

**Goes UP when:**

- You create new objects
- Add map entries
- Append to slices

**Goes DOWN when:**

- GC frees objects

**Example:**

```
START:      HeapObjects = 1314
AFTER RUN:  HeapObjects = 1889  (+575 objects)
AFTER GC:   HeapObjects = 1421  (-468 objects freed by GC)

If HeapObjects after GC > HeapObjects at start:
→ You have leaked objects!
```

**In your code:**

```
8 instances, each with ~10 map entries:
8 × 10 entries × 2 objects/entry = 160 objects just for map

Plus:
- 8 PartitionStateLock structs = 8 objects
- 8 sync.RWMutex = 8 objects
- 8 atomic.Int64 = 8 objects
- Runtime overhead = ~1300 objects

Total expected: ~1484 objects
If you see 1889, you have 405 extra objects
```

---

### **Mallocs** - Total Allocations

**What it is:** Total number of heap allocations since program start.

**Goes UP when:**

- Any allocation happens (always)

**Goes DOWN when:**

- NEVER (cumulative counter)

**Example:**

```
START:      Mallocs = 2731
AFTER RUN:  Mallocs = 3320  (+589 allocations during test)
AFTER GC:   Mallocs = 3343  (+23 allocations from GC itself)
```

**Allocation rate:**

```
589 allocations in 5 seconds = 118 allocations/second
```

**In your code:**

```
Per instance:
- appendLoop at 2ms interval: 5000ms / 2ms = 2500 ticks
  Each tick: ~3 allocations (map entry + internals)
  Total: 2500 × 3 = 7500 allocations per instance

8 instances × 7500 = 60,000 allocations expected

If you see only 589, that means:
- Tickers are slower than expected
- Or most allocations were optimized away
```

---

### **Frees** - Objects Freed by GC

**What it is:** Total number of objects freed by garbage collector.

**Goes UP when:**

- GC runs and frees dead objects

**Goes DOWN when:**

- NEVER (cumulative counter)

**Example:**

```
START:      Mallocs = 2731, Frees = 1417
            Live = 1314 objects

AFTER RUN:  Mallocs = 3320, Frees = 1431
            Live = 1889 objects (added 575 objects)

AFTER GC:   Mallocs = 3343, Frees = 1922
            Live = 1421 objects (freed 468 objects)
```

**Key relationship:**

```
Live Objects = Mallocs - Frees
1421 = 3343 - 1922 ✓
```

---

### **Live** - Current Object Count

**What it is:** Objects currently alive (not freed).

**Formula:** `Live = Mallocs - Frees`

**This is the same as HeapObjects** (just calculated differently)

**Ideal scenario:**

```
START:    Live = 1314
AFTER GC: Live = 1314  (back to baseline - no leak!)
```

**Memory leak:**

```
START:    Live = 1314
AFTER GC: Live = 1500  (+186 objects leaked)
```

---

## Garbage Collection Metrics

### **NumGC** - GC Run Count

**What it is:** Number of times garbage collection has run.

**Goes UP when:**

- GC runs (triggered by memory pressure or manually)

**Goes DOWN when:**

- NEVER

**Example:**

```
START:      NumGC = 2   (ran during program startup)
AFTER RUN:  NumGC = 2   (no GC during test - low memory pressure)
AFTER GC:   NumGC = 3   (we called runtime.GC() manually)
```

**What it means:**

- Low NumGC = Low memory pressure (good for performance)
- High NumGC = High allocation rate or low memory (bad for performance)

**GC triggers when:**

- Heap grows to 2× previous GC size (default)
- You call `runtime.GC()` manually
- Every 2 minutes at minimum (background GC)

---

### **GCCPUFraction** - GC CPU Usage

**What it is:** Percentage of CPU time spent in garbage collection.

**Formula:** `GCCPUFraction = (Time in GC) / (Total CPU time)`

**Measured across ALL CPU cores**

**Goes UP when:**

- GC runs frequently
- GC takes longer (large heap)

**Goes DOWN when:**

- Less GC activity
- More non-GC work (dilutes the percentage)

**Example:**

```
START:    GCCPUFrac = 2.42%
          Program ran for 100ms, GC used 2.42ms

AFTER 5s: GCCPUFrac = 0.0022%
          Program ran for 5100ms, GC still used ~2.42ms
          2.42 / 5100 = 0.047% (very low!)
```

**Good values:**

- < 1%: Excellent (GC barely impacting performance)
- 1-5%: Good (normal for most apps)
- 5-10%: Concerning (GC overhead getting high)
- > 10%: Bad (GC thrashing, need optimization)

**In your code:**

```
0.0022% means GC is basically free
Your allocation rate is very low
```

---

## What Changes Mean

### Memory Growing During Run ✓ EXPECTED

```
START:      Alloc = 382 KB
AFTER RUN:  Alloc = 1.54 MB  ← This is normal!

You're running code that allocates memory.
```

### Memory NOT Returning After GC ⚠️ PROBLEM

```
START:      Alloc = 382 KB,  Live = 1314
AFTER GC:   Alloc = 401 KB,  Live = 1421  ← Objects leaked!

Expected: Should return to ~382 KB
Actual: 19 KB higher = memory leak
```

### What Causes Leaks in Your Code

**1. Goroutines Still Running**

```go
ps.Cancel()
<-ps.ctx.Done()  // Returns immediately!
// But goroutines might still be running for 10-100ms
// They hold references to ps.State, ps.Mu, etc.
```

**2. Map Entries Not Deleted**

```go
// Your map might have 100-1000 entries when test ends
// Each entry = 2 objects
// 1000 entries = 2000 objects leaked
```

**3. Timers Not Stopped**

```go
// If time.Ticker isn't stopped, it leaks
t := time.NewTicker(interval)
defer t.Stop()  // ← Must have this!
```

## Quick Reference

| Metric        | What It Means          | Should Go Back to Baseline After GC? |
| ------------- | ---------------------- | ------------------------------------ |
| Alloc         | Current memory in use  | YES ✓                                |
| TotalAlloc    | Cumulative allocations | NO (always grows)                    |
| Sys           | OS memory reserved     | NO (rarely shrinks)                  |
| HeapObjects   | Live object count      | YES ✓                                |
| HeapInuse     | Active heap            | YES ✓                                |
| HeapIdle      | Reserved heap          | Inverse of HeapInuse                 |
| StackInuse    | Goroutine stacks       | YES ✓ (if goroutines exit)           |
| Mallocs       | Total allocations      | NO (always grows)                    |
| Frees         | Total frees            | NO (always grows)                    |
| Live          | Current objects        | YES ✓                                |
| NumGC         | GC run count           | NO (always grows)                    |
| GCCPUFraction | GC CPU usage           | Changes based on workload            |

---

## Memory Leak Checklist

If memory doesn't return to baseline after GC, check:

- [ ] Are all goroutines properly stopped?
- [ ] Did you wait for `sync.WaitGroup`?
- [ ] Are all `time.Ticker`s stopped with `defer t.Stop()`?
- [ ] Are maps cleared?
- [ ] Are channels closed?
- [ ] Did you give enough time between `wg.Wait()` and `runtime.GC()`?
- [ ] Did you run GC multiple times?
- [ ] Are there any global variables holding references?

---

## Further Reading

- [Go Memory Model](https://go.dev/ref/mem)
- [runtime.MemStats Documentation](https://pkg.go.dev/runtime#MemStats)
- [Garbage Collection Guide](https://tip.golang.org/doc/gc-guide)
