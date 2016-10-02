VERSION="4.1.5"
wget --quiet https://github.com/zeromq/zeromq4-1/releases/download/v$VERSION/zeromq-$VERSION.tar.gz
tar xzf zeromq-$VERSION.tar.gz
rm zeromq-$VERSION.tar.gz
cd zeromq-$VERSION
./autogen.sh
./configure
make check
sudo make install
sudo ldconfig
cd /home/ubuntu/src/github.com/ursiform/sleuth
rm -rf zeromq-$VERSION
go get
go build
go get github.com/axw/gocov/gocov
go get github.com/mattn/goveralls
go test -race && goveralls -service drone.io -repotoken $COVERALLS_TOKEN
