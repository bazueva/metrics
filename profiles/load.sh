mkdir -p profiles/load_results

hey -n 10000 -c 20 \
  -m POST \
  -H "Content-Type: application/json" \
  -D profiles/load_updates.json \
  http://localhost:8080/updates/ \
  > profiles/load_results/updates.txt 2>&1 &

hey -n 10000 -c 20 \
  http://localhost:8080/value/gauge/temperature \
  > profiles/load_results/value_url.txt 2>&1 &

hey -n 10000 -c 20 \
  http://localhost:8080/ \
  > profiles/load_results/all_metrics.txt 2>&1 &

hey -n 10000 -c 20 \
  -m POST \
  -H "Content-Type: application/json" \
  -D profiles/load_update.json \
  http://localhost:8080/update \
  > profiles/load_results/update.txt 2>&1 &

hey -n 10000 -c 20 \
  -m POST \
  -H "Content-Type: application/json" \
  -D profiles/load_value.json \
  http://localhost:8080/value/ \
  > profiles/load_results/value.txt 2>&1 &

hey -n 10000 -c 20 \
  -m POST \
  http://localhost:8080/update/gauge/load_update_url_gauge/25.5 \
  > profiles/load_results/update_url.txt 2>&1 &

hey -n 10000 -c 20 \
  http://localhost:8080/ping \
  > profiles/load_results/ping.txt 2>&1 &

wait

echo "Load test finished"