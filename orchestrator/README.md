Docker Devnet Setup
===================

Programm zum Aufsetzen eines Devnets mit Docker.

Das go Programm (`orchestrator` oder einfach `go run .`) startet die entsprechenden Services und nimmt die Konfigurationen vor. Da dafür allerdings RPC Calls in die Container nötig sind, und die Container keine Ports auf den Host gemappt haben muss dieses Programm innerhalb des Docker-Netzwerks ausgeführt werden. Hierzu dient das `dev` Script, erstellt einen Container in den der Projekt-Ordner auf `/proj` gemapt ist und öffnet eine Shell in diesem.

Ablauf zum starten also:

``` sh
./dev
cd /proj
go run .
```

