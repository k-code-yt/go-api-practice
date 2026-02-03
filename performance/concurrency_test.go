package main

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

func Benchmark_ReadOnlyConcurrency(b *testing.B) {

	gCounts := []int{2, 4, 8, 16, 32}
	numKeysSlice := []int{10_000, 100_000, 1_000_000}

	for _, numKeys := range numKeysSlice {
		fmt.Printf("\n")
		fmt.Printf("---RUNNING NUMKEYS = %d----\n", numKeys)

		for _, numGoroutines := range gCounts {
			b.Run(fmt.Sprintf("LOCK_G=%d", numGoroutines), func(b *testing.B) {
				mu := new(sync.RWMutex)
				m := make(map[int]MsgState)

				// pre-fill map
				for idx := range numKeys {
					m[idx] = MsgState_Pending
				}

				opsPerG := b.N / numGoroutines
				if opsPerG == 0 {
					opsPerG = 1
				}
				wg := new(sync.WaitGroup)
				wg.Add(numGoroutines)

				// start actual test
				b.ResetTimer()
				for range numGoroutines {
					go func() {
						defer wg.Done()
						for range opsPerG {
							key := rand.Intn(numKeys)
							mu.RLock()
							_ = m[key]
							mu.RUnlock()
						}
					}()

				}
				wg.Wait()
			})

			b.Run(fmt.Sprintf("SYNCMAP_G=%d", numGoroutines), func(b *testing.B) {
				m := new(sync.Map)

				for idx := range numKeys {
					m.Store(idx, MsgState_Pending)
				}

				opsPerG := b.N / numGoroutines
				if opsPerG == 0 {
					opsPerG = 1
				}
				wg := new(sync.WaitGroup)
				wg.Add(numGoroutines)

				// start actual test
				b.ResetTimer()

				for range numGoroutines {
					go func() {
						defer wg.Done()
						for range opsPerG {
							key := rand.Intn(numKeys)
							m.Load(key)
						}
					}()

				}
				wg.Wait()
			})
		}
	}

}

func Benchmark_MixedOnlyConcurrency(b *testing.B) {
	gCounts := []int{2, 4, 8, 16, 32}
	numKeysSlice := []int{10_000, 100_000, 1_000_000}
	moduloNum := 5

	for _, numKeys := range numKeysSlice {
		fmt.Printf("\n")
		fmt.Printf("---RUNNING NUMKEYS = %d----\n", numKeys)

		for _, numGoroutines := range gCounts {
			b.Run(fmt.Sprintf("LOCK_G=%d", numGoroutines), func(b *testing.B) {
				mu := new(sync.RWMutex)
				m := make(map[int]MsgState)

				// pre-fill map
				for idx := range numKeys {
					m[idx] = MsgState_Pending
				}

				opsPerG := b.N / numGoroutines
				if opsPerG == 0 {
					opsPerG = 1
				}
				wg := new(sync.WaitGroup)
				wg.Add(numGoroutines)

				// start actual test
				b.ResetTimer()
				for range numGoroutines {
					go func() {
						defer wg.Done()
						for i := range opsPerG {
							key := rand.Intn(numKeys)
							if i%moduloNum == 0 {
								mu.RLock()
								_ = m[key]
								mu.RUnlock()
							} else {
								mu.Lock()
								m[key] = MsgState_Success
								mu.Unlock()
							}
						}
					}()

				}
				wg.Wait()
			})

			b.Run(fmt.Sprintf("SYNCMAP_G=%d", numGoroutines), func(b *testing.B) {
				m := new(sync.Map)

				for idx := range numKeys {
					m.Store(idx, MsgState_Pending)
				}

				opsPerG := b.N / numGoroutines
				if opsPerG == 0 {
					opsPerG = 1
				}
				wg := new(sync.WaitGroup)
				wg.Add(numGoroutines)

				// start actual test
				b.ResetTimer()

				for range numGoroutines {
					go func() {
						defer wg.Done()
						for i := range opsPerG {
							key := rand.Intn(numKeys)
							if i%moduloNum == 0 {
								m.Load(key)
							} else {
								m.Swap(key, MsgState_Success)
							}
						}
					}()
				}
				wg.Wait()
			})
		}
	}
}

// ----
const (
	numKeys       = 10_000
	numGoroutines = 16
)

func Benchmark_MapReadOnly(b *testing.B) {
	mu := new(sync.RWMutex)
	m := make(map[int]MsgState)

	// pre-fill map
	for idx := range numKeys {
		m[idx] = MsgState_Pending
	}

	opsPerG := b.N / numGoroutines
	if opsPerG == 0 {
		opsPerG = 1
	}
	wg := new(sync.WaitGroup)
	wg.Add(numGoroutines)

	// start actual test
	b.ResetTimer()
	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range opsPerG {
				key := rand.Intn(numKeys)
				mu.RLock()
				_ = m[key]
				mu.RUnlock()
			}
		}()

	}
	wg.Wait()
}

func Benchmark_SyncMapReadOnly(b *testing.B) {
	m := new(sync.Map)

	for idx := range numKeys {
		m.Store(idx, MsgState_Pending)
	}

	opsPerG := b.N / numGoroutines
	if opsPerG == 0 {
		opsPerG = 1
	}
	wg := new(sync.WaitGroup)
	wg.Add(numGoroutines)

	// start actual test
	b.ResetTimer()

	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range opsPerG {
				key := rand.Intn(numKeys)
				m.Load(key)
			}
		}()

	}
	wg.Wait()
}
