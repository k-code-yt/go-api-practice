[x] refactor code
[x] 3 go routines - append, read, update state,
[x] update logic - min, max offset in atomic -> don't use the entire range -> N-first elements
[] test cases config - read interval, update interval, append count and stop time
[] pprof for compare
[] http server to trigger
[] grafana/prom middleware?
[] add docker-compose with mem/cpu limits
