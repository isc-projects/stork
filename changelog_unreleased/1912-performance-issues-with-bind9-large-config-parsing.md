[func] marcin

    Improved performance of BIND 9 configuration parsing by the agent.
    Added support for annotating parts of the BIND 9 configuration to
    skip parsing them. These annotations are useful in large deployments
    when parsing the BIND 9 configuration file can take significant
    amount of time. Use //@stork:no-parse:scope and
    //@stork:no-parse:end to skip parsing selected part of the
    configuration file. Use //@stork:no-parse:global to skip parsing
    the rest of the configuration file following the annotation.
    (Gitlab #1912)
