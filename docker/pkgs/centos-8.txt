FROM centos:8

WORKDIR /repo
RUN \
        dnf -y install --enablerepo=PowerTools ruby ruby-devel rubygems rubygem-rake gcc make rpm-build libffi-devel git wget unzip java-11-openjdk-headless python3-sphinx && \
        gem install --no-document fpm
