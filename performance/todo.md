[x] refactor code
[x] 3 go routines - append, read, update state,
[x] update logic - min, max offset in atomic -> don't use the entire range -> N-first elements
[] add docker-compose with mem/cpu limits
[] http server to trigger
[] grafana/prom middleware?
[] test cases config - read interval, update interval, append count and stop time
[] pprof for compare
