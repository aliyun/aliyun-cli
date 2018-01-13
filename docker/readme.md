![docker](https://github.frapsoft.com/top/docker-security.jpg)

# alpine-aliyuncli

[![Docker Automated Build](https://img.shields.io/docker/automated/ellerbrock/alpine-aliyuncli.svg)](https://hub.docker.com/r/ellerbrock/alpine-aliyuncli/) [![Docker Pulls](https://img.shields.io/docker/pulls/ellerbrock/alpine-aliyuncli.svg)](https://hub.docker.com/r/ellerbrock/alpine-aliyuncli/) [![Open Source Love](https://badges.frapsoft.com/os/v1/open-source.svg)](https://github.com/ellerbrock/open-source-badges/)

## Installation

`docker pull ellerbrock/alpine-aliyuncli`

## Usage

### Export your credentials

```
export ALICLOUD_ACCESS_KEY="your-access-key-here"
export ALICLOUD_SECRET_KEY="your-secret-key-here"
export ALICLOUD_REGION="your-region-here"

```

#### Run Docker

```
docker run \
  -e "ALICLOUD_ACCESS_KEY=${ALICLOUD_ACCESS_KEY}" \
  -e "ALICLOUD_SECRET_KEY=${ALICLOUD_SECRET_KEY}" \
  -e "ALICLOUD_REGION=${ALICLOUD_REGION}" \
ellerbrock/alpine-aliyuncli
```

*You can pass the variables directly without exporting them,
but since i will use this later with Travis CI and encrypted environment variables
this speraration makes it more easy for me.*
