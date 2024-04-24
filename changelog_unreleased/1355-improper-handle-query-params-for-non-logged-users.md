[bug] slawek

    Fixed a problem with improper redirecting after login. If the
    non-logged user entered any subpage rather than the root page, it
    was stuck on the login page after signing in.
    (Gitlab #1355)
