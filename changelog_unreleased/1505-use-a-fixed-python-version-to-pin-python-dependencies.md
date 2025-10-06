[build] slawek

    Split Python requirements in separate files for each Python version.
    The Python requirement files are now generated dynamically if they
    do not already exist. This improves building, especially of the UI,
    on systems with old Python versions.
    (Gitlab #1505)
