### Profiling

Add to main file:

	go func() {
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
	 } ()

Import:

	import _ "net/http/pprof"

From the console run:

	go tool pprof http://localhost:6060/debug/pprof/heap
	
	go tool pprof http://localhost:6060/debug/pprof/profile	