[bug] marcin

    Fixed an issue with getting Kea configurations containing
    consecutive quotes from the database. It resulted in using
    invalid configuration strings (missing quotes) when trying
    to update Kea configurations.
    (Gitlab #2112)
