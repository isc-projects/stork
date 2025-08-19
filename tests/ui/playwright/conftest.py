# tests/ui/playwright/conftest.py
# Make repo root importable so 'from tests.ui.playwright.pages ...' works.
import os
import sys

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../.."))
if REPO_ROOT not in sys.path:
    sys.path.insert(0, REPO_ROOT)
