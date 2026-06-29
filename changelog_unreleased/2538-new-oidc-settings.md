[func] piotrek

    Added new settings to configure OpenID Connect authentication.
    Now it is possible to set unique OpenID Provider identifier whenever
    it changes in Stork deployment. It is also possible to manually set
    OpenID Provider endpoints if the Provider doesn't support OIDC
    Discovery.
    (Gitlab #2538, #2536)
