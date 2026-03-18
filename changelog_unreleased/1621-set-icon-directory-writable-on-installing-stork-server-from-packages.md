[bug] slawek

    Fixed the authentication-methods directory permissions in
    packaging scripts. The stork-server process could not write
    LDAP hook icons because the directory was owned by root after
    package installation. Added mkdir and chown in all post-install
    scripts and os.MkdirAll in Go code as defense-in-depth.
    (Gitlab #1621)
