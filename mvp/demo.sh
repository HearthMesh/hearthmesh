#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")"
demo_dir=$(mktemp -d)
pids=""
cleanup() {
  for pid in $pids; do kill "$pid" 2>/dev/null || true; done
  for pid in $pids; do wait "$pid" 2>/dev/null || true; done
  rm -rf -- "$demo_dir"
}
trap cleanup EXIT INT TERM

go build -o "$demo_dir/hearthmesh" ./cmd/hearthmesh
mkdir -p "$demo_dir/source" "$demo_dir/a" "$demo_dir/b" "$demo_dir/c"
printf 'A small, signed, public HearthPack.\n' > "$demo_dir/source/hello.txt"

"$demo_dir/hearthmesh" serve -data "$demo_dir/a" -listen 127.0.0.1:18081 -peers http://127.0.0.1:18082,http://127.0.0.1:18083 > "$demo_dir/a.log" 2>&1 & a_pid=$!; pids="$pids $a_pid"
"$demo_dir/hearthmesh" serve -data "$demo_dir/b" -listen 127.0.0.1:18082 -peers http://127.0.0.1:18081,http://127.0.0.1:18083 > "$demo_dir/b.log" 2>&1 & pids="$pids $!"
"$demo_dir/hearthmesh" serve -data "$demo_dir/c" -listen 127.0.0.1:18083 -peers http://127.0.0.1:18081,http://127.0.0.1:18082 > "$demo_dir/c.log" 2>&1 & pids="$pids $!"

ready=0
for attempt in $(seq 1 30); do
  if curl -fsS http://127.0.0.1:18081/v1/status > /dev/null 2>&1; then ready=1; break; fi
  sleep 1
done
test "$ready" -eq 1 || { echo 'node A did not start' >&2; exit 1; }

"$demo_dir/hearthmesh" pack -src "$demo_dir/source" -out "$demo_dir/pack.zip" -key "$demo_dir/publisher.key"
pack_id=$("$demo_dir/hearthmesh" verify -pack "$demo_dir/pack.zip" | sed -n 's/^verified pack ID: //p')
"$demo_dir/hearthmesh" publish -pack "$demo_dir/pack.zip" -node http://127.0.0.1:18081

replicated=0
for attempt in $(seq 1 30); do
  if curl -fsS http://127.0.0.1:18082/v1/packs | grep -q "$pack_id" && curl -fsS http://127.0.0.1:18083/v1/packs | grep -q "$pack_id"; then replicated=1; break; fi
  sleep 1
done
test "$replicated" -eq 1 || { echo 'replication timed out' >&2; exit 1; }
echo 'All three nodes have verified the pack.'

kill "$a_pid"
wait "$a_pid" 2>/dev/null || true
echo 'Node A stopped. Nodes B and C retained the verified pack.'

printf 'deliberate corruption\n' > "$demo_dir/b/packs/$pack_id.zip"
repaired=0
for attempt in $(seq 1 30); do
  if "$demo_dir/hearthmesh" verify -pack "$demo_dir/b/packs/$pack_id.zip" > /dev/null 2>&1; then repaired=1; break; fi
  sleep 1
done
test "$repaired" -eq 1 || { echo 'recovery timed out' >&2; exit 1; }
echo 'Node B repaired its corrupt local copy from node C while node A remained offline.'
echo 'HearthMesh POSIX demo passed.'
