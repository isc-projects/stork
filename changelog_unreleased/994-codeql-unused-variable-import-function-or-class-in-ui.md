[build] andrei

    UI linters for unused variables and unused imports are now run with
    "rake lint:ui". Setting the FIX environment variable enables autofixing, but
    it only works with unused imports. The reported lint errors were fixed.
    (Gitlab #994)
