logfile := debug.log

default:
	rm -f ./charm
	go build

run:
	export CHARM_LOGFILE=$(logfile)
	rm -f ./charm
	go build
	./charm

linux:
	rm -f ./charm
	GOOS=linux go build

log:
	>$(logfile)
	tail -f $(logfile)
