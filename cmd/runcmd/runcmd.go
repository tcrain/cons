/*
github.com/tcrain/cons - Experimental project for testing and scaling consensus algorithms.
Copyright (C) 2020 The project authors - tcrain

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.

*/
/*
Code for a binary that will run a command on an list of nodes through ssh.
It has an option to run rsync directly instead.
*/
package main

import (
	// "os"
	"bytes"
	"flag"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/tcrain/cons/consensus/logging"
	"github.com/tcrain/cons/consensus/utils"
)

func main() {
	var ipFile string
	var keyFile string
	var userName string
	var runRsync bool
	var retries int
	flag.StringVar(&ipFile, "f", "ipfile", "Path to file containing list of rpc node addresses, 1 per line, NOTE: If we find ++ in the command we replace it with the ip (++ is a special command)")
	flag.StringVar(&keyFile, "k", "key.pem", "Path to key file")
	flag.StringVar(&userName, "u", "ec2-user", "User name")
	flag.BoolVar(&runRsync, "r", false, "Run rsync instead of ssh, the first input is the local directory to rsync, the second is the remote")
	flag.IntVar(&retries, "rt", 3, "Number of times to retry failed commands")
	flag.Parse()

	logging.Info("Reading ip file: ", ipFile)
	ips, err := utils.ReadTCPIPFile(ipFile, false)
	if err != nil {
		logging.Error(err)
		panic(err)
	}
	logging.Print("Loaded ips: ", ips)

	toRun := flag.Args()

	var cmdList []string
	if !runRsync {
		logging.Print("Command to run: ", toRun)
		cmdList = []string{"ssh", "-o", "StrictHostKeyChecking no", "-i", keyFile, "place user@ip here", "the command to run will be here"}
	} else {
		if len(toRun) != 2 {
			logging.Error("Input for rsync must be 2 items, the local directory to rsync and the remote directory")
			panic("invalid input")
		}
		logging.Print("Rsync path: ", toRun)
		// path := strings.Join(toRun, " ")
		cmdList = []string{"rsync", "-arce", fmt.Sprintf("ssh -o StrictHostKeyChecking=no -i %s", keyFile), "--exclude",
			"*.store", "--exclude", "*.out", toRun[0], toRun[1]}
	}

	var wg sync.WaitGroup
	for i, ip := range ips {
		// we only run 100 call at once
		if (i+1)%101 == 0 {
			wg.Wait()
		}
		newCmd := make([]string, len(cmdList))
		copy(newCmd, cmdList)

		if !runRsync {
			// If we find ++ we replace it with the ip (++ is a special command)
			newToRun := make([]string, len(toRun))
			copy(newToRun, toRun)
			for j, nxt := range newToRun {
				if nxt == "++" {
					newToRun[j] = ip
				}
			}
			newCmd[6] = fmt.Sprintf("%s", strings.Join(newToRun, " "))
		}

		if !runRsync {
			newCmd[5] = fmt.Sprintf("%s@%s", userName, ip)
		} else {
			newCmd[8] = fmt.Sprintf("%s@%s:%s", userName, ip, newCmd[8])
		}

		wg.Add(1)
		go func(newCmd []string) {
			var err error
			for i := 0; i < retries; i++ {
				cmd := exec.Command(newCmd[0], newCmd[1:]...)
				logging.Print("Running command: ", newCmd)

				var out bytes.Buffer
				err = nil
				cmd.Stdout = &out
				cmd.Stderr = &out
				if err = cmd.Run(); err != nil {
					logging.Errorf("Error running command %s: %v, %v", cmd.Args, err, cmd.Stderr)
					time.Sleep(10 * time.Second)
				} else {
					logging.Print("Ran command successfully", cmd.Args)
					break
				}
			}
			if err != nil {
				panic(err)
			}
			wg.Done()
		}(newCmd)
	}
	wg.Wait()
}
