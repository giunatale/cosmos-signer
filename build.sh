mkdir -p build
echo "Building cosmos-signer"
go build -C cmd/cosmos-signer -o ../../build/cosmos-signer
pushd plugins > /dev/null
./build.sh
popd > /dev/null

