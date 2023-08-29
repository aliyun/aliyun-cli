# First check OS.
OS="$(uname)"
if [[ "${OS}" == "Linux" ]]
then
  CLI_ON_LINUX=1
elif [[ "${OS}" == "Darwin" ]]
then
  CLI_ON_MACOS=1
else
  abort "Currently is only supported on macOS and Linux."
fi

if [[ -n $0 ]]
then
VERSION="$0"
else
VERSION="latest"
fi

# if [[ -n "${CLI_ON_MACOS-}" ]] && [[ "${HAVE_SUDO_ACCESS}" -ne 0 ]]
# then
#   abort "Need sudo access on macOS (e.g. the user ${USER} needs to be an Administrator)!"
# fi

if [[ -n "${CLI_ON_MACOS-}" ]]
then
  curl -O -fsSL https://aliyuncli.alicdn.com/aliyun-cli-macosx-$VERSION-universal.tgz
  tar zxf aliyun-cli-macosx-$VERSION-universal.tgz
  mv ./aliyun /usr/local/bin/
fi

if [[ -n "${CLI_ON_LINUX-}" ]]
then
  UNAME_MACHINE="$(/usr/bin/uname -m)"
  if [[ "${UNAME_MACHINE}" == "arm64" ]]
  then
    curl -O -fsSL https://aliyuncli.alicdn.com/aliyun-cli-linux-$VERSION-arm64.tgz
    tar zxf aliyun-cli-linux-$VERSION-arm64.tgz
  else
    curl -O -fsSL https://aliyuncli.alicdn.com/aliyun-cli-linux-$VERSION-amd64.tgz
    tar zxf aliyun-cli-linux-$VERSION-amd64.tgz
  fi
  mv ./aliyun /usr/local/bin/
fi
