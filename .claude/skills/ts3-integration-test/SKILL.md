---
name: ts3-integration-test
description: Use when running ts3afkmover end-to-end against a real TeamSpeak 3 server locally — spinning up the docker-compose test stack, wiring TS3_URL/WebQuery ports, and getting an API key to exercise the daemon.
---

# Local integration testing

`test/docker-compose.test.yaml` spins up a real `teamspeak` server (+ MariaDB +
ts3-manager UI on :8080). WebQuery HTTP is on **:10080**. Point `TS3_URL` at
`http://127.0.0.1:10080` and grab an API key from the ts3 server admin to exercise
the daemon end-to-end. `test/query_ip_allowlist.txt` is mounted to allow query
access from private ranges.
