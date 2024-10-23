[build] marcin

    Changes in DEB packages to use useradd instead of adduser command.
    The former is available by default on deb-based distributions but
    the latter isn't, causing potential issues with installing Stork
    on these systems.
    (Gitlab #1537)