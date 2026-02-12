[x] refactor code
[x] 3 go routines - append, read, update state,
[x] update logic - min, max offset in atomic -> don't use the entire range -> N-first elements
[x] add docker-compose with mem/cpu limits
[x] http server to trigger
[x] grafana/prom middleware?
[x] test cases config - read interval, update interval, append count and stop time
[x] fix clean-up logic && make sure no mem leaks
[x] compare map vs sync.map with using just logs
[x] add state pre-fill for all versions -> test read-heavy
[x] compare the same with chans implementation
[x] refactor test to run separately -> make sure tests are not affecting one another
[x] compare with pprof

<!-- DONE W/ PARTITION STATE -->

[x] string compare -> concat vs builder
[x] figure out why builder allocates -> and how to avoid
[x] custom metrics
[x] how to measure throuput -> compare JSON vs proto
[x] gomaxprocs effect on tests
[] add sync.Pool to improve further && test JSON

<!-- GRAFANA/PROM -->

[] compare all in grafana/prom
