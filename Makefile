default: install

$(GOPATH)/bin/godeps:
	go get github.com/rogpeppe/godeps

godeps: $(GOPATH)/bin/godeps
	$(GOPATH)/bin/godeps -u dependencies.tsv

install: godeps
	go generate ./commands
	go install -v ./juju-1.25-upgrade 

.PHONY: default godeps install
