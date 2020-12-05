#!/bin/bash
inip=${1}
user=${2}
key=${3}

echo Running rsync with inital instance
rsync -arce "ssh -o StrictHostKeyChecking=no -i ${key}" --include="*/" --include="*.go" --include="*.sh" --include="*.gp" --exclude="*" ./ ${user}@${inip}:"~/go/src/github.com/tcrain/cons/"
