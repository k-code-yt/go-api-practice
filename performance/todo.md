[x] refactor code
[x] 3 go routines - append, read, update state,
[x] update logic - min, max offset in atomic -> don't use the entire range -> N-first elements
[x] add docker-compose with mem/cpu limits
[x] http server to trigger
[x] grafana/prom middleware?
[x] test cases config - read interval, update interval, append count and stop time
[x] fix clean-up logic && make sure no mem leaks

[] compare map vs sync.map with using just logs
[] compare with pprof
[] compare the same with chans implementation
[] compare all in grafana/prom
