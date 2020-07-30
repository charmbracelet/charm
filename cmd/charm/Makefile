logfile := debug.log

.PHONY: default run linux log enable_logging

default:
	rm -f ./charm
	go build

run: enable_logging
	rm -f ./charm
	go build
	./charm

linux:
	rm -f ./charm
	GOOS=linux go build

log:
	>$(logfile)
	tail -f $(logfile)

enable_logging:
	export CHARM_LOGFILE=$(logfile)
