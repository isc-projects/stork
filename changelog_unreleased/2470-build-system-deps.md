[func] tomek, slawek

    The build system no longer downloads specific NodeJS and Go compiler
    and uses the version available in the OS instead. This makes it
    possible to build Stork with different NodeJS and Go versions. Also,
    the compilation should be a bit faster and smaller as there is a
    couple less items to download.
    (Gitlab #2470)
