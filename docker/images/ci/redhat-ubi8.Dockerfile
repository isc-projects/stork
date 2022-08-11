FROM redhat/ubi8:8.6

WORKDIR /repo
RUN dnf install -y \
    gcc-8.5.0 \
    git-2.31.1 \
    java-11-openjdk-headless-1:11.0.16.0.8 \
    make-1:4.2.1 \
    procps-ng-3.3.15 \
    python3-virtualenv-15.1.0 \
    rpm-build-4.14.3 \
    rubygem-rake-12.3.3 \
    ruby-devel-2.5.9 \
    unzip-6.0 \
    wget-1.19.5
