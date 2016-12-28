GOVERSION="1.7.3"
ZMQVERSION="4.1.5"
OS=linux
ARCH=amd64
sudo rm -rf /usr/local/go
cd ~
echo "Downloading go$GOVERSION.$OS-$ARCH.tar.gz..."
wget --quiet https://storage.googleapis.com/golang/go$GOVERSION.$OS-$ARCH.tar.gz
mv go$GOVERSION.$OS-$ARCH.tar.gz golang.tar.gz
sudo tar -C /usr/local -xzf golang.tar.gz
rm golang.tar.gz
cd /home/ubuntu/src/github.com/ursiform/sleuth
wget --quiet https://github.com/zeromq/zeromq4-1/releases/download/v$ZMQVERSION/zeromq-$ZMQVERSION.tar.gz
tar xzf zeromq-$ZMQVERSION.tar.gz
rm zeromq-$ZMQVERSION.tar.gz
cd zeromq-$ZMQVERSION
sudo apt-get install libtool pkg-config build-essential autoconf automake
./autogen.sh
./configure
make
sudo make install
sudo ldconfig
cd /home/ubuntu/src/github.com/ursiform/sleuth
rm -rf zeromq-$ZMQVERSION
go get
go build
go get github.com/axw/gocov/gocov
go get github.com/mattn/goveralls
go test -race && goveralls -service drone.io -repotoken $COVERALLS_TOKEN
