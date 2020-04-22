- Given that data centers use a lot of energy,
the benchmarks are set up so that most testing can be done locally,
or on a few physical nodes while using lots of "virtual nodes", and where signatures validations are set to be sleeps instead of computations.
This way the time spent running the large scale benchmarks can be kept to minimum (this also helps keep down the
benchmark budget).

- The benchmarks support using preemptable instances on Google Cloud
  - These instances are signficantly cheaper than the normal ones
  - When a node gets preempted druing a benchmark, kill the benchmark, and run ``bash ./scripts/retryfullrun.sh''
    - This wait for all nodes to be unpreempted, and restart the benchmark from where it left off
  - Unfortunately I found that when running more than around 10 instances, preemptions would be happening so frequently
  that a single configuration would not even be able to run (where a configuration took around 1 minute to run), so
  preemtable instances seem only useful for small benchmarks.
  - (Previously I had run experiments on Amazon Spot instances with 300+ instances with very few being shut down
  so I was surprised that the Google instances were preempted so frequently)

- The vCPUs on the n-type Google cloud instances are signficantly slower than my couple year old portable laptop with a
  low power CPU. This statement doesn't mean much in isolation, but I suppose it at least means that if benchmark results
  seem slow then this is one way to improve them.

- Even though most things are done through scripts, I am frequently making new images and trying different configurations.
  For this I found Google cloud easier to manage than Amazon EC2. On EC2 the different regions are managed in a
  seemingly completely decentralized manner which made things slightly more tedious (though this may be a good thing for some cases).

- Why did I use Google cloud?
  - Originally I was setting up things to be run on DigitalOcean, but given they are a smaller provider, and me being
  an unknown client, it was, understandably, becoming difficult to raise my quotas.
  - So I (unfortunately) decided to go with one of the major providers.
    - And Google Cloud gives you $300 free credits when you sign up. xD
