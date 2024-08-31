
# Stable and Log stores for Raft
HashiCorp's Raft implementation, used in projects like Consul and Vault, relies on a few key components for ensuring the distributed consensus protocol operates correctly. Among these are the Stable Store and Log Store, both of which are critical for maintaining the state and history of operations in a Raft-based system.
Here's an overview of each:

1. Stable Store

The Stable Store is responsible for storing persistent state information necessary for the Raft protocol. This includes:

- Current Term: The latest term the node has seen, used to identify the leader for a term.
- Voted For: The candidate that the node voted for in the current term. This is crucial to prevent the node from voting for more than one candidate in a given term.

These pieces of information must be stored persistently because they are necessary to maintain the correctness of the Raft protocol across node restarts. If a node crashes and restarts, it should resume from the last known state without losing its voting record or the term number.
1. Log Store

The Log Store is responsible for storing the Raft log entries, which are crucial for ensuring consistency across the nodes in the cluster. The log entries typically contain the following:

- Command: The actual command or operation to be applied to the state machine.
- Term: The term in which the log entry was created.
- Index: The position of the log entry in the sequence of operations.

The Log Store also supports operations like:

- Appending Entries: Adding new entries to the log.
- Retrieving Entries: Fetching entries based on the index or term.
- Compaction: Truncating the log to remove old entries that are no longer necessary, typically after a snapshot is taken.

Raft's Log Store in HashiCorp Implementation

In HashiCorp's Raft, the Log Store can be implemented using different backends depending on the persistence and performance requirements. The default implementation typically uses an embedded key-value store like BoltDB or LevelDB, but it can be replaced with other storage backends if necessary.

## Benchmarking

To run DOQ server we need to go to a `cmd/server` direcory and run:

```bash
go run main.go localhost 8001 9001
```

To run the actual benchmarks we need to go to `/testing/load` and then run `k6 run -u 10 -d 10s queue.js`


### Benchmark results for DOQ with [BoldDB](https://github.com/hashicorp/raft-boltdb) as a Stable Store and Log Store

#### Run 10 users for 10 seconds

```bash
> $ k6 run -u 10 -d 10s queue.js

        /\      |‾‾| /‾‾/   /‾‾/
    /\  /  \     |  |/  /   /  /
    /  \/    \    |     (   /   ‾‾\
/          \   |  |\  \ |  (‾)  |
/ __________ \  |__| \__\ \_____/ .io

    execution: local
        script: queue.js
        output: -

    scenarios: (100.00%) 1 scenario, 10 max VUs, 40s max duration (incl. graceful stop):
            * default: 10 looping VUs for 10s (gracefulStop: 30s)


    ✓ enqueued

    checks.........................: 100.00% ✓ 5211       ✗ 0
    data_received..................: 3.8 MB  378 kB/s
    data_sent......................: 1.5 MB  146 kB/s
    http_req_blocked...............: avg=3.3µs   min=0s     med=2µs     max=980µs    p(90)=4µs     p(95)=4µs
    http_req_connecting............: avg=691ns   min=0s     med=0s      max=404µs    p(90)=0s      p(95)=0s
    http_req_duration..............: avg=19.07ms min=6.03ms med=17.08ms max=217.95ms p(90)=24.97ms p(95)=25.57ms
    { expected_response:true }...: avg=19.07ms min=6.03ms med=17.08ms max=217.95ms p(90)=24.97ms p(95)=25.57ms
    http_req_failed................: 0.00%   ✓ 0          ✗ 5212
    http_req_receiving.............: avg=50.28µs min=8µs    med=37µs    max=1.88ms   p(90)=85µs    p(95)=110µs
    http_req_sending...............: avg=13.8µs  min=2µs    med=10µs    max=501µs    p(90)=22µs    p(95)=28µs
    http_req_tls_handshaking.......: avg=0s      min=0s     med=0s      max=0s       p(90)=0s      p(95)=0s
    http_req_waiting...............: avg=19.01ms min=6ms    med=17.02ms max=217.93ms p(90)=24.9ms  p(95)=25.5ms
    http_reqs......................: 5212    519.574045/s
    iteration_duration.............: avg=19.19ms min=1.83µs med=17.21ms max=218.19ms p(90)=25.1ms  p(95)=25.69ms
    iterations.....................: 5211    519.474357/s
    vus............................: 10      min=10       max=10
    vus_max........................: 10      min=10       max=10


running (10.0s), 00/10 VUs, 5211 complete and 0 interrupted iterations
default ✓ [======================================] 10 VUs  10s
```


#### Run 100 users for 10 seconds

```bash
> $ k6 run -u 100 -d 10s queue.js

        /\      |‾‾| /‾‾/   /‾‾/
    /\  /  \     |  |/  /   /  /
    /  \/    \    |     (   /   ‾‾\
/          \   |  |\  \ |  (‾)  |
/ __________ \  |__| \__\ \_____/ .io

    execution: local
        script: queue.js
        output: -

    scenarios: (100.00%) 1 scenario, 100 max VUs, 40s max duration (incl. graceful stop):
            * default: 100 looping VUs for 10s (gracefulStop: 30s)


    ✓ enqueued

    checks.........................: 100.00% ✓ 46719       ✗ 0
    data_received..................: 34 MB   3.4 MB/s
    data_sent......................: 13 MB   1.3 MB/s
    http_req_blocked...............: avg=3.13µs  min=0s     med=1µs     max=2.94ms  p(90)=2µs     p(95)=2µs
    http_req_connecting............: avg=2µs     min=0s     med=0s      max=2.92ms  p(90)=0s      p(95)=0s
    http_req_duration..............: avg=21.35ms min=5.38ms med=21.63ms max=54.16ms p(90)=27.26ms p(95)=30.38ms
    { expected_response:true }...: avg=21.35ms min=5.38ms med=21.63ms max=54.16ms p(90)=27.26ms p(95)=30.38ms
    http_req_failed................: 0.00%   ✓ 0           ✗ 46720
    http_req_receiving.............: avg=23.08µs min=7µs    med=13µs    max=6.87ms  p(90)=37µs    p(95)=56µs
    http_req_sending...............: avg=7.76µs  min=2µs    med=4µs     max=2.18ms  p(90)=10µs    p(95)=15µs
    http_req_tls_handshaking.......: avg=0s      min=0s     med=0s      max=0s      p(90)=0s      p(95)=0s
    http_req_waiting...............: avg=21.32ms min=5.35ms med=21.6ms  max=54.15ms p(90)=27.19ms p(95)=30.35ms
    http_reqs......................: 46720   4658.023136/s
    iteration_duration.............: avg=21.41ms min=1µs    med=21.68ms max=54.19ms p(90)=27.32ms p(95)=30.43ms
    iterations.....................: 46719   4657.923435/s
    vus............................: 100     min=100       max=100
    vus_max........................: 100     min=100       max=100


running (10.0s), 000/100 VUs, 46719 complete and 0 interrupted iterations
default ✓ [======================================] 100 VUs  10s
```
