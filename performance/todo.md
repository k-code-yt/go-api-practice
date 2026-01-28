[] refactor code
[] 3 go routines - append, read, update state,
[] update logic - min, max offset in atomic -> don't use the entire range -> N-first elements
[] test cases config - read interval, update interval, append count and stop time
[] http server to trigger
[] grafana/prom middleware?
