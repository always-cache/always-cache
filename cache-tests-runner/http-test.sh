#!/bin/bash

npm run cli --base=http://localhost:8080 --id=$1

inotifywait -q -m -e modify .. |
while read -r filename event; do
  npm run cli --base=http://localhost:8080 --id=$1
done
