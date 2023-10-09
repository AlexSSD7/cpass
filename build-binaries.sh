rm -rf build
mkdir build

version=$1

if [ -z "$version" ]; then
    echo "Version is not specified (first positional argument)"
    exit 1
fi

function build() {
    name="cpass_${1}_${2}_${version}"
    binary_name="$name"
    if [ $1 == "windows" ]; then
        binary_name="$binary_name.exe"
    fi
    
    CGO_ENABLED=0 GOOS=$1 GOARCH=$2 go build -trimpath -o build/$binary_name
    cd build

    if [ $1 == "windows" ] || [ $1 == "darwin" ]; then
        zip $name.zip $binary_name
    else
        tar czvf $name.tar.gz $binary_name
    fi

    rm $binary_name
    cd ..
}

build linux amd64
build linux arm64
build linux 386
build linux arm

build darwin amd64
build darwin arm64

build windows amd64
build windows arm64
build windows 386
build windows arm

build freebsd amd64
build freebsd arm64
build freebsd 386
build freebsd arm

build openbsd amd64
build openbsd arm64
build openbsd 386
build openbsd arm

build netbsd amd64
build netbsd arm64
build netbsd 386
build netbsd arm

cd build

hashes_file="cpass_sha256_$version.txt"

sha256sum * > $hashes_file
gpg --output ${hashes_file}.sig --detach-sign --local-user F7231DFD3333A27F71D171383B627C597D3727BD --armor $hashes_file