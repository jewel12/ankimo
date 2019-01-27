export GOOS := linux
export GOARCH := arm

CRED := service_account.json

.PHONY: clean recorder

record: recorder
	./recorder -c ${CRED}

recorder: recorder.go
	go build recorder.go

clean:
	-rm recorder