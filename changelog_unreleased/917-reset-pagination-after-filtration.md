[bug] piotrek

    Fixed a bug in hosts reservation filtering.
    Now, whenever new filter is applied, pagination is reset
    by default to the first page of the filtered results. 
    (Gitlab #917)