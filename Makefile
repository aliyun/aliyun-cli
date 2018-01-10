all: build

testdeps:
	go get -v github.com/onsi/ginkgo/ginkgo
	go get -v github.com/onsi/gomega
	go install github.com/onsi/ginkgo/ginkgo

build:
	go build -o aliyun main/main.go