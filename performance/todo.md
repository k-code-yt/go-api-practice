[x] refactor code
[x] 3 go routines - append, read, update state,
[x] update logic - min, max offset in atomic -> don't use the entire range -> N-first elements
[x] add docker-compose with mem/cpu limits
[x] http server to trigger
[x] grafana/prom middleware?
[x] test cases config - read interval, update interval, append count and stop time
[x] fix clean-up logic && make sure no mem leaks
[x] compare map vs sync.map with using just logs
[] add state pre-fill for all versions -> test read-heavy
[] compare the same with chans implementation

[] compare with pprof
[] compare all in grafana/prom
