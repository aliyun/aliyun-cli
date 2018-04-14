# build aliyun-cli (aliyun)
FROM golang as builder
ADD . src/github.com/aliyun/aliyun-cli/
WORKDIR /go/src/github.com/aliyun/aliyun-cli
RUN make deps; \
  make testdeps; \
  make build;

# build aliyun image from alpine
FROM debian
MAINTAINER 叶泽宇/Zeyu Ye <zeyu.ye@airwallex.com>
COPY --from=builder /go/src/github.com/aliyun/aliyun-cli/out/aliyun /usr/local/bin
CMD ["/usr/local/bin/aliyun"]
