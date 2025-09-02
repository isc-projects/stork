## installation
pip install -r requirements.txt
playwright install # install playwright tools and browsers


## record new test
playwright codegen https://stork.lab.isc.org/
playwright codegen https://stork.lab.isc.org/ --output test_example.py --target python-pytest

# generated code copy into tests/ui/playwright/test_basic.py 



## run tests
pytest tests/ui/playwright/tests_poc/test_example.py --headed --slowmo=200 -q -s
# options can be added to pytest.ini 


## debug failing tests
2.PWDEBUG=1 pytest tests/ui/playwright/tests_poc/test_example.py -s


