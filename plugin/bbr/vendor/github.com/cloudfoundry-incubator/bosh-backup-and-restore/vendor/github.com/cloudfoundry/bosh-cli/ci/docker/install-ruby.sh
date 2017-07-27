#!/bin/bash

set -ex

RUBY_ROOT=/usr/local/ruby
RUBY_ARCHIVE_URL=http://cache.ruby-lang.org/pub/ruby/2.3/ruby-2.3.1.tar.gz
RUBY_ARCHIVE=$(basename $RUBY_ARCHIVE_URL)
RUBY_NAME=$(basename -s .tar.gz $RUBY_ARCHIVE_URL)
RUBY_DOWNLOAD_SHA256=b87c738cb2032bf4920fef8e3864dc5cf8eae9d89d8d523ce0236945c5797dcd
echo "Downloading ruby..."
curl -fSL -o $RUBY_ARCHIVE $RUBY_ARCHIVE_URL
echo "$RUBY_DOWNLOAD_SHA256 $RUBY_ARCHIVE" | sha256sum -c -

tar xf $RUBY_ARCHIVE

echo "Installing ruby..."
mkdir -p $(dirname $RUBY_ROOT)
cd $RUBY_NAME
./configure --prefix=$(dirname $RUBY_ROOT) --disable-install-doc --with-openssl-dir=/usr/include/openssl
make
make install
ln -s /usr/local/$RUBY_NAME $RUBY_ROOT

export PATH=$RUBY_ROOT/bin:$PATH
