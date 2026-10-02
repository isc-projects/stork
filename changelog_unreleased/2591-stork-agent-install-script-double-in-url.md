[bug] marcin

    Fixed the agent installation script returned in the
    "Installing Stork Server on a New Machine" view. This
    script could fail to download the Stork agent package
    bundled in the server, due to an invalid Stork server URL.
    The URL in the script is now properly sanitized. The
    script was also updated to correctly handle registering
    and starting the agent on Alpine Linux.
    (Gitlab #2591)
