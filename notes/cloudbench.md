# Google Cloud Compute Benchmarks

Scripts have been created in order to run benchmarks on google cloud.

### Requirements
- A Google cloud compute account with a project that will be used for the benchmarks.
- An OAUTH2 json file downloaded from Google cloud to authenticate you.
- Large enough quotas on Google cloud compute for the desired benchmark configurations.
- An ssh key file that will be used to connect to the nodes.

### Generating the test configurations

The test configurations that will be run have to be coded in the
[gento.go](../cmd/gento/gento.go) main function.
See that file for examples, but the basic idea is that a set of folders will be created
within the [testconfigs](../testconfigs/) folder.
Each folder will contain a set of json files of encoded Testoptions
objects each describing a benchmark. Each folder will also contain a
`gensets` folder which will describe how the benchmark results graphs
will be generated.
  
### Set environment variables
Set the following environment variables:
  - $GITBRANCH - Git branch to use for creating the image for the benchmarks (after the image is
  created, the local source files will be synced using rsync with the remote nodes)
  - $KEYPATH - Path to ssh key file that will be used to connect to nodes **note this key will be copied to the image
  so be sure this key is just for the benchmarks**
  - $PROJECTID - Project ID of the Google cloud project
  - $OAUTHPATH - Path to the google cloud oauth2 file
  - $BENCHUSER - Linux username that will be used to connect to the benchmark nodes.

### Run the benchmark

From the project home folder call

``bash ./scripts/cloudscripts/fullrun.sh {tofolders}``

Where `{tofolders}` is the path of the folders generated in the previous step.

There are many other arguments that this script can take,
the following is the full list of ordered arguments
(default values are in parentheses).
- tofolders - folders that contain the test options created in the previous step
- regions - (us-central1) list of regions to run nodes in
- nodesperregion - (1) number of nodes to launch in each region
- nodecounts - (4) list of node counts to run each benchmark
- launchNodes - (1) Launch bench nodes
- shutdownNodes - (1) Shutdown bench nodes
- genimage - (0) Generate the image for building the benchmark
- deleteimage - (0) Delete the generated image at the end of the benchmark
- instancetype - (n1-standard-1) instance type of the nodes
- branch - ($GITBRANCH) git branch to use for creating the image
- singleZoneRegion - (0) run nodes in the same region in the same zone
- homezone - (us-central1-a) zone from where the benchmarks will be launched
- homeinstancetype - (n1-standard-2) instance type that will launch the benchmarks
- goversion - (1.5) version of go to use
- user - ($BENCHUSER) user name to log onto instances
- key - ($KEYPATH) key to use to log onto instances
- project - ($PROJECTID) google cloud project to use
- credentialfile - ($OAUTHPATH) credential file for google cloud

The benchmark consists of the following steps.

1. A single instance is launched and an image is created on this node. This will
install all the needed packages to compile and run the tests.
The image is the stored on google cloud.
Note that this step only needs to be run once (following
benchmarks can reuse this image).
2. A instance using the created image is launched, call this
instance the benchmark coordinator.
3. The local node's sorce files are synchronized withe the benchmark
coordinator instance. Binaries are compiled.
4. Instances that will run consensus are launched. Certain settings
on these nodes are configured and the binaries are copied to these nodes.
The command ```rpcbench``` is started on each node.
5. The participant register is started on the benchmark coordinator.
6. The benchmarks are run.
7. The consensus instances are shut down.
8. The graphs are generated at the benchmark coordinator.
9. Results are copied to the local node and stored in the
[benchresults](../benchresults) folder.
10. The benchmark coordinator is shut down.
11. The image created is deleted.

#### Retry failed bench
If a benchmark fails for some reason, cancel the command,
then run:

``bash ./scripts/cloudscripts/retryfullrun.sh``

This will be sure the nodes are started, then restart the
benchmark at the last configuration that finished successfully.
This avoids having to restart the benchmark from scratch.

#### Reboot benchmark nodes
You may want to reboot nodes in case something goes wrong.
For this run:

``bash scripts/cloudscripts/remoteresetnodes.sh``

#### Shutdown benchmark nodes
If you have not set the benchmark to shut down the nodes automatically,
you can run (note this does not shut down the benchmark coordinator node):

``bash scripts/cloudscripts/remoteshutdown.sh``

### Some other useful commands
Get the IP of the benchmark coordinator node:

``go run ./cmd/instancesetup/instancesetup.go -z {corrdinator-zone} -ii``

Get the IPs of the benchmark nodes of a multi-region benchmark (places them in file benchIPfile):

``bash scripts/cloudscripts/getips.sh --nlrz {benchmark-regions}``

Or for single region benchmarks:

``bash scripts/cloudscripts/getips.sh --lrz {benchmark-regions}``

#### Some examples
Some examples of benchmarks can be found in the
[paperbenches](../scripts/paperbenches/) folder.

#### Debugging
Nodes running the benchmark can be attached to through
the delve (https://github.com/go-delve/delve) debugger as follows:
1. Before running the bechmark, uncomment the line ``flags=(-gcflags="all=-N -l")``
in [buildgo.sh](../scripts/buildgo.sh).
2. Run the benchark as normal.
3. While a benchmark is running, run script
``bash ./scripts/cloudscripts/attachdebug.sh {ip}`` where {ip}
is the IP address of the node to attach to.

To attach to the log output of a node run:
``bash ./scripts/cloudscripts/attachlog.sh {ip}`` where {ip}
is the IP address of the node to attach to.

#### Profiling

First uncomment the lines needed to enable profiling (see (profile)[profile.md]).

Now run the command:

``bash scripts/cloudscripts/attachprofile.sh {ip} {profile-type}``
Where `{profile-type}` is either `heap`, `profile` or `allocs`.

# Running tests one at a time on Google Cloud Compute

Tests can be run on Google Cloud Compute one by one.

First launch the nodes with the command:

``
bash scripts/cloudscripts/just/justsetup.sh "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$genimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"
``

This will launch the set of nodes, update the files from git, and copy the binaries.
The input options are similar to the ones described above.

**Note that `$branch` is only used if making a new image.
For the benchmark the files will be synched with the local node, and the binares will be
built on the remote machine.**


Note that the command stores the configuration for the test in the local file `.lastjustsetup`,
so you can only run one configuration at a time.

Running the same command again with ``$launchNodes"`` set to `0` will update the files from git
and rebuild the binaries. Do this for example to update the code without having to shutdown and
relaunch the nodes.

Then to run a single benchmark run:

``
bash scripts/cloudscripts/just/justbench.sh "$config" "$doInitialSetup"
``

Where `$config` is the name of a json test configuration file on the local machine.
If `$doInitialSetup` is `1` then the benchmark nodes and binarys will be started/restarted.
This must be `1` the first time a test is run, afterwards it can be set to `0` allowing
the config to be run without restarting the nodes/binaries.

Finally, to shut down the nodes run:

``
bash scripts/cloudscripts/just/justshutdown.sh 
``