FROM redhat/ubi8

WORKDIR /repo
RUN dnf -y install gcc git java-11-openjdk-headless libffi-devel make procps-ng python3-pip rpm-build ruby ruby-devel rubygems rubygem-rake unzip wget
RUN gem install --no-document fpm
RUN pip3 install --upgrade pip
RUN pip3 install sphinx sphinx-rtd-theme
