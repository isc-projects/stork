[bug] wlipinski

    Fixed a bug causing Kea to reject updated subnet configuration sent
    by Stork when option value was specified as numeric 1 or 0 in Kea,
    and the definition was unspecified for that option. In that case,
    Stork converted these values to "true" or "false". Kea rejected
    these values because it could not determine whether they were
    appropriate given that original values were numeric. If Stork does
    not know option definition, it treats 1 and 0 as numeric values
    and does not convert them to boolean. Conversion to boolean only
    performed for known option formats containing explicit boolean
    types.
    (Gitlab #2652)
