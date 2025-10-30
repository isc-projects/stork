[func] marcin

    Limit the number of tokens in a single line of the PowerDNS
    configuration to 500. It prevents attempts to parse malformed
    configurations and excessive use of memory during parsing.
    (Gitlab #2131)
