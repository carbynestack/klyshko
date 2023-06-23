sudo apt-get install -y --no-install-recommends \
automake \
build-essential
\
clang-11 \
cmake \
git \
libboost-dev \
libboost-thread-dev \
libclang-dev
\
libgmp-dev \
libntl-dev \
libsodium-dev \
libssl-dev \
libtool \
vim \
gdb
\
valgrind

pip install --upgrade pip ipython

git submodule update --init --recursive

make init cd MP-SPDZ make -j clean boost cowgear-offline.x
