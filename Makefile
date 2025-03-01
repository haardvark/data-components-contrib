SHELL = /bin/bash

.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	go test ./...

.PHONY: generate-acknowledgements
generate-acknowledgements:
	echo -e "# Open Source Acknowledgements\n\nSpice.ai would like to acknowledge the following open source projects for making this project possible:\n\nGo Modules\n" > ACKNOWLEDGEMENTS.md
	go get github.com/google/go-licenses
	pushd dataconnectors && go-licenses csv . 2>/dev/null >> ../ACKNOWLEDGEMENTS.md && popd
	pushd dataprocessors && go-licenses csv . 2>/dev/null >> ../ACKNOWLEDGEMENTS.md && popd

	sed -i 's/\"//g' ACKNOWLEDGEMENTS.md
	sed -i 's/,/, /g' ACKNOWLEDGEMENTS.md
	sed -i 's/,  /, /g' ACKNOWLEDGEMENTS.md