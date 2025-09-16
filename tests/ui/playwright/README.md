## installation
pip install -r requirements.txt
playwright install # install playwright tools and browsers


## record new test
playwright codegen https://stork.lab.isc.org/
playwright codegen https://stork.lab.isc.org/ --output test_example.py --target python-pytest

# generated code copy into tests/ui/playwright/test_basic.py 



## run tests
rake systemtestui:down
rake systemtestui:build
rake systemtestui:up
rake systemtestui:test
rake 'systemtestui:test[,--headed -s]' 
rake systemtestui:test['tests/ui/playwright/tests_poc/test_example.py','--headed --slowmo=200 -q -s']
rake systemtestui:test_debug['tests/ui/playwright/tests_poc/test_example.py','-s']
pytest tests/ui/playwright/tests_poc/test_example.py --headed --slowmo=200 -q -s
# options can be added to pytest.ini 


## debug failing tests
2.PWDEBUG=1 pytest tests/ui/playwright/tests_poc/test_example.py -s


